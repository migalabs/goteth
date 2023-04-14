package blocks

import (
	"context"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/drivers/postgresql"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/events"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
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
	VALIDATOR_SET_SIZE = 500000
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

	downloadMode string
	// Control Variables
	finishDownload bool
	routineClosed  chan struct{}
	eventsObj      events.Events

	initTime time.Time
}

func NewBlockAnalyzer(
	pCtx context.Context,
	httpCli *clientapi.APIClient,
	initSlot uint64,
	finalSlot uint64,
	idbUrl string,
	workerNum int,
	dbWorkerNum int,
	downloadMode string) (*BlockAnalyzer, error) {
	log.Infof("generating new Block Analzyer from slots %d:%d", initSlot, finalSlot)
	// gen new ctx from parent
	ctx, cancel := context.WithCancel(pCtx)

	// calculate the list of slots that we will analyze
	slotRanges := make([]uint64, 0)

	if downloadMode == "hybrid" || downloadMode == "historical" {

		epochRange := uint64(0)

		// start two epochs before and end two epochs after
		for i := initSlot; i <= finalSlot; i += 1 {
			slotRanges = append(slotRanges, i)
			epochRange++
		}
		log.Debug("slotRanges are:", slotRanges)
	}
	i_dbClient, err := postgresql.ConnectToDB(ctx, idbUrl, maxWorkers*VALIDATOR_SET_SIZE, dbWorkerNum)
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
		eventsObj:          events.NewEventsObj(ctx, httpCli),
		downloadMode:       downloadMode,
	}, nil
}

func (s *BlockAnalyzer) Run() {
	defer s.cancel()
	// Get init time
	s.initTime = time.Now()

	log.Info("Blocks Analyzer initialized at ", s.initTime)

	// Block requester
	var wgDownload sync.WaitGroup
	downloadFinishedFlag := false

	// Metrics per block
	var wgProcess sync.WaitGroup
	// processFinishedFlag := false

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
	go s.runProcessBlock(&wgProcess, &downloadFinishedFlag)

	wgDownload.Wait()
	downloadFinishedFlag = true
	log.Info("Beacon Blocks Downloads finished")

	wgProcess.Wait()
	// processFinishedFlag = true
	log.Info("Beacon Blocks Processing finished")

	s.dbClient.DoneTasks()
	<-s.dbClient.FinishSignalChan

	totalTime += int64(time.Since(start).Seconds())
	analysisDuration := time.Since(s.initTime).Seconds()
	log.Info("Blocks Analyzer finished in ", analysisDuration)
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
	Block    model.ForkBlockContentBase
	Slot     uint64
	Proposed bool
}

func (s BlockAnalyzer) CreateMissingBlock(slot phase0.Slot) model.ForkBlockContentBase {
	duties, err := s.cli.Api.ProposerDuties(s.ctx, phase0.Epoch(slot/32), []phase0.ValidatorIndex{})
	proposerValIdx := phase0.ValidatorIndex(0)
	if err != nil {
		log.Errorf("could not request proposer duty: %s", err)
	} else {
		for _, duty := range duties {
			if duty.Slot == phase0.Slot(slot) {
				proposerValIdx = duty.ValidatorIndex
			}
		}
	}

	return model.ForkBlockContentBase{
		Slot:              slot,
		ProposerIndex:     proposerValIdx,
		Graffiti:          [32]byte{},
		Proposed:          false,
		Attestations:      make([]*phase0.Attestation, 0),
		Deposits:          make([]*phase0.Deposit, 0),
		ProposerSlashings: make([]*phase0.ProposerSlashing, 0),
		AttesterSlashings: make([]*phase0.AttesterSlashing, 0),
		VoluntaryExits:    make([]*phase0.SignedVoluntaryExit, 0),
		SyncAggregate: &altair.SyncAggregate{
			SyncCommitteeBits:      bitfield.NewBitvector512(),
			SyncCommitteeSignature: phase0.BLSSignature{}},
		ExecutionPayload: model.ForkBlockPayloadBase{
			FeeRecipient:  bellatrix.ExecutionAddress{},
			GasLimit:      0,
			GasUsed:       0,
			Timestamp:     0,
			BaseFeePerGas: [32]byte{},
			BlockHash:     phase0.Hash32{},
			Transactions:  make([]bellatrix.Transaction, 0),
		},
	}
}
