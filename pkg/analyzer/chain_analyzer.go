package analyzer

import (
	"context"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/clientapi"
	"github.com/migalabs/goteth/pkg/config"
	"github.com/migalabs/goteth/pkg/db"
	prom_metrics "github.com/migalabs/goteth/pkg/metrics"
	"github.com/migalabs/goteth/pkg/utils"

	"github.com/migalabs/goteth/pkg/events"
	"github.com/pkg/errors"
)

type ChainAnalyzer struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Slot Range
	initSlot  phase0.Slot
	finalSlot phase0.Slot

	// Channels
	downloadTaskChan chan phase0.Slot

	// Connections
	cli       *clientapi.APIClient
	eventsObj events.Events
	dbClient  *db.PostgresDBService

	// Control Variables
	wgMainRoutine *sync.WaitGroup
	wgDownload    *sync.WaitGroup
	stop          bool
	routineClosed chan struct{}
	downloadMode  string
	metrics       db.DBMetrics
	processerBook *utils.RoutineBook

	queue Queue

	initTime    time.Time
	PromMetrics *prom_metrics.PrometheusMetrics
}

func NewChainAnalyzer(
	pCtx context.Context,
	iConfig config.AnalyzerConfig) (*ChainAnalyzer, error) {

	// gen new ctx from parent
	ctx, cancel := context.WithCancel(pCtx)

	// calculate the list of slots that we will analyze

	if iConfig.DownloadMode == "hybrid" || iConfig.DownloadMode == "historical" {

		if iConfig.FinalSlot <= iConfig.InitSlot {
			return &ChainAnalyzer{}, errors.Errorf("Final Slot cannot be greater than Init Slot")
		}

		log.Infof("generating new Block Analyzer from slots %d:%d", iConfig.InitSlot, iConfig.FinalSlot)
	}

	metricsObj, err := db.NewMetrics(iConfig.Metrics)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read metric.")
	}

	idbClient, err := db.New(ctx, iConfig.DBUrl, db.WithWorkers(iConfig.DbWorkerNum))
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate DB Client.")
	}

	idbClient.Connect()

	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(pCtx,
		iConfig.BnEndpoint,
		clientapi.WithELEndpoint(iConfig.ElEndpoint),
		clientapi.WithDBMetrics(metricsObj))
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate API Client.")
	}

	// generate the central exporting service
	promethMetrics := prom_metrics.NewPrometheusMetrics(ctx, "0.0.0.0", iConfig.PrometheusPort)

	analyzer := &ChainAnalyzer{
		ctx:              ctx,
		cancel:           cancel,
		initSlot:         phase0.Slot(iConfig.InitSlot),
		finalSlot:        phase0.Slot(iConfig.FinalSlot),
		downloadTaskChan: make(chan phase0.Slot),
		cli:              cli,
		dbClient:         idbClient,
		routineClosed:    make(chan struct{}, 1),
		eventsObj:        events.NewEventsObj(ctx, cli),
		downloadMode:     iConfig.DownloadMode,
		metrics:          metricsObj,
		PromMetrics:      promethMetrics,
		queue:            NewQueue(),
		processerBook:    utils.NewRoutineBook(100),
		wgMainRoutine:    &sync.WaitGroup{},
		wgDownload:       &sync.WaitGroup{},
	}

	InitGenesis(analyzer.dbClient, analyzer.cli)

	analyzerMet := analyzer.GetPrometheusMetrics()
	promethMetrics.AddMeticsModule(analyzerMet)

	return analyzer, nil
}

func (s *ChainAnalyzer) Run() {
	defer s.cancel()
	// Get init time
	s.initTime = time.Now()

	log.Info("Blocks Analyzer initialized at ", s.initTime)

	totalTime := int64(0)
	start := time.Now()

	s.wgDownload.Add(1)
	go s.runDownloadBlocks()
	if s.downloadMode == "historical" {
		// Block requester + Task generator
		s.wgMainRoutine.Add(1)
		go s.runHistorical(s.initSlot, s.finalSlot)
	}

	if s.downloadMode == "finalized" {
		// Block requester in finalized slots, not used for now
		s.wgMainRoutine.Add(1)
		go s.runHead()
	}

	s.PromMetrics.Start()

	s.wgMainRoutine.Wait()
	s.stop = true
	log.Infof("main routine finished, waiting for downloader...")

	s.wgDownload.Wait()

	log.Infof("downloader finished, waiting for db client...")

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
