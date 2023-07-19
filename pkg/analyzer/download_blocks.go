package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

// This routine is able to download block by block in the slot range
func (s *ChainAnalyzer) runDownloadBlocks(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon Block Requester")
	rootHistory := NewSlotHistory()
	queue := StateQueue{}

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

			// if epoch boundary, download state
			if slot%spec.SlotsPerEpoch == 0 {
				// new epoch
				s.DownloadNewState(&queue, slot-1, false)
			}

		}

	}

	log.Infof("Block Download routine finished")
}

func (s *ChainAnalyzer) runDownloadBlocksFinalized(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon Block Finalized Requester")

	rootHistory := NewSlotHistory()
	// ------ fill from last epoch in database to current head -------

	// obtain current head
	headSlot, headRoot := s.cli.GetFinalizedEndSlotStateRoot()

	// obtain last epoch in database
	nextSlotDownload, err := s.dbClient.ObtainLastSlot()
	if err != nil {
		log.Errorf("could not obtain last slot in database: %s", err)
	}
	if nextSlotDownload == 0 || nextSlotDownload > headSlot {
		nextSlotDownload = headSlot
	}

	nextSlotDownload = nextSlotDownload - (epochsToFinalizedTentative * spec.SlotsPerEpoch) // 2 epochs before
	queue := NewStateQueue(headSlot, headRoot)
	for nextSlotDownload < headSlot {
		log.Infof("filling missing blocks: %d", nextSlotDownload)
		s.DownloadNewBlock(&rootHistory, phase0.Slot(nextSlotDownload))
		if nextSlotDownload%spec.SlotsPerEpoch == 0 {
			// new epoch
			s.DownloadNewState(&queue, nextSlotDownload-1, true)
		}
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
			log.Tracef("received new head signal: %d", headSlot)

			for nextSlotDownload <= headSlot {
				if s.stop {
					log.Info("sudden shutdown detected, block downloader routine")
					return
				}

				s.DownloadNewBlock(&rootHistory, phase0.Slot(nextSlotDownload))

				// if epoch boundary, download state
				if nextSlotDownload%spec.SlotsPerEpoch == 0 {
					// new epoch
					s.DownloadNewState(&queue, nextSlotDownload-1, true)
				}
				nextSlotDownload = nextSlotDownload + 1

			}
		case newFinalCheckpoint := <-s.eventsObj.FinalizedChan:
			s.dbClient.Persist(db.ChepointTypeFromCheckpoint(newFinalCheckpoint))

		case newReorg := <-s.eventsObj.ReorgChan:
			s.dbClient.Persist(db.ReorgTypeFromReorg(newReorg))
			baseSlot := newReorg.Slot - phase0.Slot(newReorg.Depth)
			log.Infof("rewinding to %d", newReorg.Slot-phase0.Slot(newReorg.Depth))

			nextSlotDownload = baseSlot + 1
			s.ReorgRewind(baseSlot, newReorg.Slot)
			queue.ReOrganizeReorg(phase0.Epoch(nextSlotDownload / spec.SlotsPerEpoch))

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

func (s ChainAnalyzer) DownloadNewBlock(history *SlotRootHistory, slot phase0.Slot) {

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
	log.Tracef("sending a new task for slot %d", slot)
	s.blockTaskChan <- blockTask

	// store transactions if it has been enabled
	if s.metrics.Transactions {
		transactions, err := s.cli.RequestTransactionDetails(newBlock)
		if err == nil {
			transactionTask := &TransactionTask{
				Slot:         uint64(slot),
				Transactions: transactions,
			}
			log.Tracef("sending a new tx task for slot %d", slot)
			s.transactionTaskChan <- transactionTask
		}
	}

	<-ticker.C
	// check if the min Request time has been completed (to avoid spaming the API)
}

func (s *ChainAnalyzer) ReorgRewind(baseSlot phase0.Slot, slot phase0.Slot) {

	log.Infof("deleting block data from %d (included) onwards", baseSlot+1)
	s.dbClient.Persist(db.BlockDropType(baseSlot + 1))
	s.dbClient.Persist(db.TransactionDropType(baseSlot + 1))
	s.dbClient.Persist(db.WithdrawalDropType(baseSlot + 1))

	baseEpoch := phase0.Epoch((baseSlot + 1) / spec.SlotsPerEpoch)
	reorgEpoch := phase0.Epoch(slot / spec.SlotsPerEpoch)
	if slot%spec.SlotsPerEpoch == 31 || // end of epoch
		baseEpoch != reorgEpoch { // the reorg crosses and epoch boundary
		epoch := baseEpoch - 1
		log.Infof("deleting epoch data from %d (included) onwards", epoch)
		s.dbClient.Persist(db.EpochDropType(epoch))
		s.dbClient.Persist(db.ProposerDutiesDropType(epoch))
		s.dbClient.Persist(db.ValidatorRewardsDropType(epoch + 1)) // validator rewards are always written at epoch+1
	}

}
