package analyzer

import (
	"context"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db"
	prom_metrics "github.com/cortze/eth-cl-state-analyzer/pkg/metrics"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec/metrics"
	"github.com/sirupsen/logrus"

	"github.com/cortze/eth-cl-state-analyzer/pkg/events"
	"github.com/pkg/errors"
)

type ChainAnalyzer struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Slot Range
	initSlot  phase0.Slot
	finalSlot phase0.Slot

	// Channels
	epochTaskChan       chan *EpochTask
	valTaskChan         chan *ValTask
	blockTaskChan       chan *BlockTask
	transactionTaskChan chan *TransactionTask

	// Connections
	cli       *clientapi.APIClient
	eventsObj events.Events
	dbClient  *db.PostgresDBService

	downloadMode       string
	validatorWorkerNum int
	metrics            DBMetrics

	// Control Variables
	stop              bool
	downloadFinished  bool
	processerFinished bool
	routineClosed     chan struct{}

	initTime    time.Time
	PromMetrics *prom_metrics.PrometheusMetrics
}

func NewChainAnalyzer(
	pCtx context.Context,
	httpCli *clientapi.APIClient,
	initSlot uint64,
	finalSlot uint64,
	idbUrl string,
	workerNum int,
	dbWorkerNum int,
	downloadMode string,
	metrics string,
	prometheusPort int) (*ChainAnalyzer, error) {
	log.Infof("generating new Block Analyzer from slots %d:%d", initSlot, finalSlot)
	// gen new ctx from parent
	ctx, cancel := context.WithCancel(pCtx)

	// calculate the list of slots that we will analyze
	slotRanges := make([]uint64, 0)

	if downloadMode == "hybrid" || downloadMode == "historical" {
		log.Debug("slotRanges are:", slotRanges)
	}

	metricsObj, err := NewMetrics(metrics)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read metric.")
	}

	idbClient, err := db.New(ctx, idbUrl, db.WithWorkers(dbWorkerNum))
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate DB Client.")
	}

	idbClient.Connect()

	// generate the central exporting service
	promethMetrics := prom_metrics.NewPrometheusMetrics(ctx, "0.0.0.0", prometheusPort)

	analyzer := &ChainAnalyzer{
		ctx:                 ctx,
		cancel:              cancel,
		initSlot:            phase0.Slot(initSlot),
		finalSlot:           phase0.Slot(finalSlot),
		epochTaskChan:       make(chan *EpochTask, 1),
		valTaskChan:         make(chan *ValTask, workerNum),
		blockTaskChan:       make(chan *BlockTask, 1),
		transactionTaskChan: make(chan *TransactionTask, 1),
		validatorWorkerNum:  workerNum,
		cli:                 httpCli,
		dbClient:            idbClient,
		routineClosed:       make(chan struct{}, 1),
		eventsObj:           events.NewEventsObj(ctx, httpCli),
		downloadMode:        downloadMode,
		metrics:             metricsObj,
		PromMetrics:         promethMetrics,
	}

	InitGenesis(analyzer.dbClient, analyzer.cli)

	analyzerMet := analyzer.GetMetrics()
	promethMetrics.AddMeticsModule(analyzerMet)

	return analyzer, nil
}

func (s *ChainAnalyzer) Run() {
	defer s.cancel()
	// Get init time
	s.initTime = time.Now()

	log.Info("Blocks Analyzer initialized at ", s.initTime)

	// Block requester
	var wgDownload sync.WaitGroup

	// Metrics per block
	var wgProcess sync.WaitGroup

	// Validator metrics
	var wgWorkers sync.WaitGroup

	// Transactions per block
	var wgTransaction sync.WaitGroup

	totalTime := int64(0)
	start := time.Now()

	if s.downloadMode == "hybrid" || s.downloadMode == "historical" {
		// Block requester + Task generator
		wgDownload.Add(1)
		go s.runDownloadBlocks(&wgDownload)
	}

	if s.downloadMode == "hybrid" || s.downloadMode == "finalized" {
		// Block requester in finalized slots, not used for now
		wgDownload.Add(1)
		go s.runDownloadBlocksFinalized(&wgDownload)
	}
	wgProcess.Add(1)
	go s.runProcessBlock(&wgProcess)
	wgProcess.Add(1)
	go s.runProcessState(&wgProcess)

	wgTransaction.Add(1)
	go s.runProcessTransactions(&wgTransaction)

	for i := 0; i < s.validatorWorkerNum; i++ {
		// state workers, receiving State and valIdx to measure performance
		wlog := logrus.WithField(
			"worker", i,
		)

		wlog.Tracef("Launching Task Worker")
		wgWorkers.Add(1)
		go s.runWorker(wlog, &wgWorkers)
	}

	s.PromMetrics.Start()

	wgDownload.Wait()
	s.downloadFinished = true

	wgProcess.Wait()
	s.processerFinished = true
	close(s.blockTaskChan)
	close(s.epochTaskChan)

	wgTransaction.Wait()
	close(s.transactionTaskChan)

	wgWorkers.Wait()
	close(s.valTaskChan)

	s.dbClient.Finish()

	totalTime += int64(time.Since(start).Seconds())
	analysisDuration := time.Since(s.initTime).Seconds()
	log.Info("Blocks Analyzer finished in ", analysisDuration)
	s.routineClosed <- struct{}{}
}

func (s *ChainAnalyzer) Close() {
	log.Info("Sudden closed detected, closing StateAnalyzer")
	s.stop = true
	<-s.routineClosed // Wait for services to stop before returning
}

type BlockTask struct {
	Block    spec.AgnosticBlock
	Slot     uint64
	Proposed bool
}

type TransactionTask struct {
	Slot         uint64
	Transactions []*spec.AgnosticTransaction
}

type EpochTask struct {
	NextState spec.AgnosticState
	State     spec.AgnosticState
	PrevState spec.AgnosticState
	Finalized bool
}

type ValTask struct {
	ValIdxs         []phase0.ValidatorIndex
	StateMetricsObj metrics.StateMetrics
	OnlyPrevAtt     bool
	PoolName        string
	Finalized       bool
}
