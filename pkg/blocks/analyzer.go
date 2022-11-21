package blocks

import (
	"context"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/cortze/eth2-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	modName = "Blocks"
	log     = logrus.WithField(
		"module", modName,
	)
	maxWorkers         = 50
	minReqTime         = 1 * time.Second / 10
	MAX_VAL_BATCH_SIZE = 20000
	VAL_LEN            = 400000
	SLOT_SECONDS       = 12
	EPOCH_SLOTS        = 32
)

type BlockAnalyzer struct {
	ctx        context.Context
	cancel     context.CancelFunc
	InitSlot   uint64
	FinalSlot  uint64
	SlotRanges []uint64

	validatorWorkerNum int
	BlockTaskChan      chan *BlockTask

	cli      *clientapi.APIClient
	dbClient *postgresql.PostgresDBService

	// Control Variables
	finishDownload bool
	routineClosed  chan struct{}

	initTime time.Time
}

func NewBlockAnalyzer(
	pCtx context.Context,
	httpCli *clientapi.APIClient,
	initSlot uint64,
	finalSlot uint64,
	idbUrl string,
	workerNum int,
	dbWorkerNum int) (*BlockAnalyzer, error) {
	log.Infof("generating new Block Analzyer from slots %d:%d", initSlot, finalSlot)
	// gen new ctx from parent
	ctx, cancel := context.WithCancel(pCtx)

	// calculate the list of slots that we will analyze
	slotRanges := make([]uint64, 0)
	epochRange := uint64(0)

	// start two epochs before and end two epochs after
	for i := initSlot; i <= finalSlot; i += 1 {
		slotRanges = append(slotRanges, i)
		epochRange++
	}
	log.Debug("slotRanges are:", slotRanges)

	i_dbClient, err := postgresql.ConnectToDB(ctx, idbUrl, maxWorkers, dbWorkerNum)
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate DB Client.")
	}

	return &BlockAnalyzer{
		ctx:                ctx,
		cancel:             cancel,
		InitSlot:           initSlot,
		FinalSlot:          finalSlot,
		SlotRanges:         slotRanges,
		BlockTaskChan:      make(chan *BlockTask, 10),
		cli:                httpCli,
		dbClient:           i_dbClient,
		validatorWorkerNum: workerNum,
		routineClosed:      make(chan struct{}),
	}, nil
}

func (s *BlockAnalyzer) Run() {
	defer s.cancel()
	// Get init time
	s.initTime = time.Now()

	log.Info("State Analyzer initialized at ", s.initTime)

	// Block requester
	var wgDownload sync.WaitGroup
	downloadFinishedFlag := false

	// Metrics per block
	var wgProcess sync.WaitGroup
	// processFinishedFlag := false

	totalTime := int64(0)
	start := time.Now()

	// Block requester + Task generator
	wgDownload.Add(1)
	go s.runDownloadBlocks(&wgDownload)

	// Block requester in finalized slots, not used for now
	// wgDownload.Add(1)
	// go s.runDownloadBlocksFinalized(&wgDownload)

	wgProcess.Add(1)
	go s.runProcessBlock(&wgProcess, &downloadFinishedFlag)

	wgDownload.Wait()
	downloadFinishedFlag = true
	log.Info("Beacon State Downloads finished")

	wgProcess.Wait()
	// processFinishedFlag = true
	log.Info("Beacon State Processing finished")

	s.dbClient.DoneTasks()
	<-s.dbClient.FinishSignalChan

	totalTime += int64(time.Since(start).Seconds())
	analysisDuration := time.Since(s.initTime).Seconds()
	log.Info("State Analyzer finished in ", analysisDuration)
	if s.finishDownload {
		s.routineClosed <- struct{}{}
	}
}

func (s *BlockAnalyzer) Close() {
	log.Info("Sudden closed detected, closing StateAnalyzer")
	s.finishDownload = true
	<-s.routineClosed
	s.cancel()
}

//
type BlockTask struct {
	Block spec.VersionedSignedBeaconBlock
	Slot  uint64
}
