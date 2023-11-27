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
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"

	"github.com/migalabs/goteth/pkg/events"
	"github.com/pkg/errors"
)

type ChainAnalyzer struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Slot Range for historical
	initSlot  phase0.Slot
	finalSlot phase0.Slot

	// Channels
	downloadTaskChan chan phase0.Slot // channel to send download tasks

	// Connections
	cli       *clientapi.APIClient  // client to request data to the CL and EL clients
	eventsObj events.Events         // object to receive signals from beacon node
	dbClient  *db.PostgresDBService // client to communicate with psql

	// Control Variables
	wgMainRoutine *sync.WaitGroup    // wait group for main routine (either historical or head)
	wgDownload    *sync.WaitGroup    // wait group for download routine
	stop          bool               // flag to notify all routine to finish
	routineClosed chan struct{}      // signal that everything was closed succesfully
	downloadMode  string             // whether to download historical blocks (defined by user) or follow chain head
	metrics       db.DBMetrics       // waht metrics to be downloaded / processed
	processerBook *utils.RoutineBook // defines slot to process new metrics into the database, good for monitoring

	downloadCache ChainCache // store the blocks and states downloaded

	initTime    time.Time
	PromMetrics *prom_metrics.PrometheusMetrics // metrics to be stored to prometheus
}

func NewChainAnalyzer(
	pCtx context.Context,
	iConfig config.AnalyzerConfig) (*ChainAnalyzer, error) {

	// gen new ctx from parent
	ctx, cancel := context.WithCancel(pCtx)

	// generate the central exporting service
	promethMetrics := prom_metrics.NewPrometheusMetrics(ctx, "0.0.0.0", iConfig.PrometheusPort)

	// calculate the list of slots that we will analyze

	if iConfig.DownloadMode == "historical" {

		if iConfig.FinalSlot <= iConfig.InitSlot {
			return &ChainAnalyzer{
				ctx:    ctx,
				cancel: cancel,
			}, errors.Errorf("Final Slot cannot be greater than Init Slot")
		}
		iConfig.InitSlot = iConfig.InitSlot / spec.SlotsPerEpoch * spec.SlotsPerEpoch
		iConfig.FinalSlot = iConfig.FinalSlot / spec.SlotsPerEpoch * spec.SlotsPerEpoch
		log.Infof("generating new Block Analyzer from slots %d:%d", iConfig.InitSlot, iConfig.FinalSlot)
	}

	metricsObj, err := db.NewMetrics(iConfig.Metrics)
	if err != nil {
		return &ChainAnalyzer{
			ctx:    ctx,
			cancel: cancel,
		}, errors.Wrap(err, "unable to read metric.")
	}

	idbClient, err := db.New(ctx, iConfig.DBUrl, db.WithWorkers(iConfig.DbWorkerNum))
	if err != nil {
		return &ChainAnalyzer{
			ctx:    ctx,
			cancel: cancel,
		}, errors.Wrap(err, "unable to generate DB Client.")
	}

	idbClient.Connect()

	// generate the httpAPI client
	cli, err := clientapi.NewAPIClient(pCtx,
		iConfig.BnEndpoint,
		clientapi.WithELEndpoint(iConfig.ElEndpoint),
		clientapi.WithDBMetrics(metricsObj),
		clientapi.WithPromMetrics(promethMetrics))
	if err != nil {
		return &ChainAnalyzer{
			ctx:    ctx,
			cancel: cancel,
		}, errors.Wrap(err, "unable to generate API Client.")
	}

	analyzer := &ChainAnalyzer{
		ctx:              ctx,
		cancel:           cancel,
		initSlot:         phase0.Slot(iConfig.InitSlot),
		finalSlot:        phase0.Slot(iConfig.FinalSlot),
		downloadTaskChan: make(chan phase0.Slot, rateLimit), // TODO: define size of buffer depending on performance
		cli:              cli,
		dbClient:         idbClient,
		routineClosed:    make(chan struct{}, 1),
		eventsObj:        events.NewEventsObj(ctx, cli),
		downloadMode:     iConfig.DownloadMode,
		metrics:          metricsObj,
		PromMetrics:      promethMetrics,
		downloadCache:    NewQueue(),
		processerBook:    utils.NewRoutineBook(32, "processer"), // one whole epoch
		wgMainRoutine:    &sync.WaitGroup{},
		wgDownload:       &sync.WaitGroup{},
	}

	InitGenesis(analyzer.dbClient, analyzer.cli)

	analyzerMet := analyzer.GetPrometheusMetrics()
	promethMetrics.AddMeticsModule(analyzerMet)
	promethMetrics.AddMeticsModule(analyzer.processerBook.GetPrometheusMetrics())
	promethMetrics.AddMeticsModule(idbClient.GetPrometheusMetrics())

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
