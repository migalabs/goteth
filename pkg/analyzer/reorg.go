package analyzer

import (
	"fmt"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s *ChainAnalyzer) AdvanceFinalized(newFinalizedSlot phase0.Slot) {

	finalizedEpoch := newFinalizedSlot / spec.SlotsPerEpoch

	stateKeys := s.downloadCache.StateHistory.GetKeyList()

	advance := false

	for _, epoch := range stateKeys {
		if epoch >= uint64(finalizedEpoch) {
			continue // only process epochs that are before the given epoch
		}
		advance = true // only set flag if there is something to do

		// Retrieve stored root and redownload root once finalized
		queueState := s.downloadCache.StateHistory.Wait(epoch)
		finalizedStateRoot := s.cli.RequestStateRoot(phase0.Slot(queueState.Slot))
		historyStateRoot := queueState.StateRoot

		if finalizedStateRoot != historyStateRoot { // no match, reorg happened
			log.Fatalf("state root for state (slot=%d) incorrect, redownload", queueState.Slot)
		}

		// loop over slots in the epoch
		for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {

			// Retrieve stored root and redownload root once finalized
			queueBlock := s.downloadCache.BlockHistory.Wait(slot)
			finalizedBlockRoot := s.cli.RequestStateRoot(phase0.Slot(queueBlock.Slot))
			historyBlockRoot := queueBlock.StateRoot

			if finalizedBlockRoot != historyBlockRoot {
				log.Fatalf("state root for block (slot=%d) incorrect, redownload", queueBlock.Slot)
			}
		}
	}

	s.downloadCache.CleanUpTo(newFinalizedSlot)

	if advance {
		log.Infof("checked states until slot %d, epoch %d", newFinalizedSlot, newFinalizedSlot/spec.SlotsPerEpoch)

	}
}

func (s *ChainAnalyzer) HandleReorg(newReorg v1.ChainReorgEvent) {
	depth := newReorg.Depth
	reorgSlot := newReorg.Slot
	fromSlot := reorgSlot - phase0.Slot(depth)
	for i := fromSlot; i < reorgSlot; i++ { // for every slot in the reorg
		block := s.downloadCache.BlockHistory.Wait(uint64(i)) // first check that it was already in the cache
		// keep orphans
		if block.Proposed {
			s.dbClient.Persist(db.OrphanBlock(*block))
		}
		s.DownloadBlock(i) // -> inserts into the queue and replaces old block

		s.processerBook.WaitUntilInactive(fmt.Sprintf("%s%d", slotProcesserTag, i)) // wait until has been processed

		s.dbClient.DeleteBlockMetrics(i)
		// write slot metrics
		s.ProcessBlock(i)

		if (i+1)%spec.SlotsPerEpoch == 0 { // then we are at the end of the epoch, rewrite state
			epoch := phase0.Epoch(i % spec.SlotsPerEpoch)
			s.downloadCache.StateHistory.Available(uint64(i)) // first check that it was already in the cache
			s.DownloadState(i)                                // -> inserts into the queue and replaces old block

			s.processerBook.WaitUntilInactive(fmt.Sprintf("%s%d", epochProcesserTag, i)) // wait until has been processed

			s.dbClient.DeleteStateMetrics(epoch)
			// write slot metrics
			s.ProcessStateTransitionMetrics(epoch)
		}

	}
}
