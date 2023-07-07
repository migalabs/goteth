package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

// This routine is able to download block by block in the slot range
func (s *BlockAnalyzer) runDownloadBlocks(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon Block Requester")
	rootHistory := NewSlotHistory()

loop:
	// loop over the list of slots that we need to analyze
	for slot := s.initSlot; slot < s.finalSlot; slot += 1 {

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing block requester routine")
			break loop

		default:
			if s.stop {
				log.Info("sudden shutdown detected, block downloader routine")
				break loop
			}

			log.Infof("requesting Beacon Block from endpoint: slot %d", slot)
			s.DownloadNewBlock(&rootHistory, phase0.Slot(slot))

		}

	}

	log.Infof("Block Download routine finished")
}

func (s *BlockAnalyzer) runDownloadBlocksFinalized(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon Block Finalized Requester")

	rootHistory := NewSlotHistory()
	// ------ fill from last epoch in database to current head -------

	// obtain current head
	headSlot, _ := s.cli.GetFinalizedEndSlotStateRoot()

	// obtain last epoch in database
	nextSlotDownload, err := s.dbClient.ObtainLastSlot()
	if err != nil {
		log.Errorf("could not obtain last slot in database: %s", err)
	}
	if nextSlotDownload == 0 {
		nextSlotDownload = headSlot
	}

	for nextSlotDownload < headSlot {
		log.Infof("filling missing blocks: %d", nextSlotDownload)
		s.DownloadNewBlock(&rootHistory, phase0.Slot(nextSlotDownload))
		nextSlotDownload = nextSlotDownload + 1
		if s.stop {
			log.Info("sudden shutdown detected, block downloader routine")
			return
		}
	}

	// -----------------------------------------------------------------------------------
	s.eventsObj.SubscribeToHeadEvents()
	s.eventsObj.SubscribeToFinalizedCheckpointEvents()
	s.eventsObj.SubscribeToReorgsEvents()
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
	// loop over the list of slots that we need to analyze

	for {
		select {

		case headSlot := <-s.eventsObj.HeadChan: // wait for new head event
			// make the block query
			log.Infof("received new head signal: %d", headSlot)

			for nextSlotDownload <= headSlot {
				log.Infof("downloading block: %d", nextSlotDownload)
				s.DownloadNewBlock(&rootHistory, phase0.Slot(nextSlotDownload))
				nextSlotDownload = nextSlotDownload + 1
				if s.stop {
					log.Info("sudden shutdown detected, block downloader routine")
					return
				}
			}
		// case newFinalCheckpoint := <-s.eventsObj.FinalizedChan:
		// 	// slot must be the last slot previous to the finalized epoch

		case newReorg := <-s.eventsObj.ReorgChan:
			s.dbClient.Persist(db.ReorgTypeFromReorg(newReorg))
			log.Infof("rewinding to %d", newReorg.Slot-phase0.Slot(newReorg.Depth))

			nextSlotDownload = newReorg.Slot - phase0.Slot(newReorg.Depth) + 1
			s.ReorgRewind(nextSlotDownload)

		case <-s.ctx.Done():
			log.Info("context has died, closing block requester routine")
			return

		case <-ticker.C:
			if s.stop {
				log.Info("sudden shutdown detected, block downloader routine")
				return
			}
		}

	}
}

func (s BlockAnalyzer) DownloadNewBlock(history *SlotRootHistory, slot phase0.Slot) {

	ticker := time.NewTicker(minBlockReqTime)
	newBlock, proposed, err := s.cli.RequestBeaconBlock(slot)
	if err != nil {
		log.Panicf("block error at slot %d: %s", slot, err)
	}
	history.AddRoot(slot, newBlock.StateRoot)

	// send task to be processed
	blockTask := &BlockTask{
		Block:    newBlock,
		Slot:     uint64(slot),
		Proposed: proposed,
	}
	log.Debugf("sending a new task for slot %d", slot)
	s.blockTaskChan <- blockTask

	// store transactions if it has been enabled
	if s.enableTransactions {
		transactions, err := s.cli.RequestTransactionDetails(newBlock)
		if err == nil {
			transactionTask := &TransactionTask{
				Slot:         uint64(slot),
				Transactions: transactions,
			}
			log.Debugf("sending a new tx task for slot %d", slot)
			s.transactionTaskChan <- transactionTask
		}
	}

	<-ticker.C
	// check if the min Request time has been completed (to avoid spaming the API)
}

func (s *BlockAnalyzer) ReorgRewind(slot phase0.Slot) {

	log.Infof("deleting block data from %d (included) onwards", slot)
	s.dbClient.Persist(db.BlockDropType(slot))
	s.dbClient.Persist(db.TransactionDropType(slot))
	s.dbClient.Persist(db.WithdrawalDropType(slot))

}
