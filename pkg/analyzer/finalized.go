package analyzer

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s *ChainAnalyzer) AdvanceFinalized(newFinalizedSlot phase0.Slot) {

	finalizedEpoch := newFinalizedSlot / spec.SlotsPerEpoch

	rewriteSlots := make([]phase0.Slot, 0)
	rewriteEpochs := make([]phase0.Epoch, 0)

	stateKeys := s.queue.StateHistory.GetKeyList()

	for _, epoch := range stateKeys {
		if epoch >= uint64(finalizedEpoch) {
			continue // only process epochs that are before the given epoch
		}

		// Retrieve stored root and redownload root once finalized
		queueState := s.queue.StateHistory.Wait(epoch)
		finalizedStateRoot := s.cli.RequestStateRoot(phase0.Slot(queueState.Slot))
		historyStateRoot := queueState.StateRoot

		if finalizedStateRoot != historyStateRoot { // no match, reorg happened
			log.Warnf("state root for state (slot=%d) incorrect, redownload", queueState.Slot)
			// need to redownload the epoch
			s.queue.StateHistory.Delete(epoch)
			s.DownloadState(queueState.Slot) // -> inserts into the queue

			// keep track of the rewrite metrics
			rewriteEpochs = append(rewriteEpochs, phase0.Epoch(epoch))
		}

		// loop over slots in the epoch
		for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {

			// Retrieve stored root and redownload root once finalized
			queueBlock := s.queue.BlockHistory.Wait(slot)
			finalizedBlockRoot := s.cli.RequestStateRoot(phase0.Slot(queueBlock.Slot))
			historyBlockRoot := queueBlock.StateRoot

			if finalizedBlockRoot != historyBlockRoot {
				log.Warnf("state root for block (slot=%d) incorrect, redownload", queueBlock.Slot)
				// need to redownload the epoch
				s.queue.BlockHistory.Delete(slot)
				s.DownloadBlock(phase0.Slot(slot)) // -> inserts into the queue

				// keep track of the rewrite metrics
				rewriteSlots = append(rewriteSlots, phase0.Slot(slot))

				// keep orphans
				if queueBlock.Proposed {
					s.dbClient.Persist(db.OrphanBlock(queueBlock))
				}
			}

		}

	}

	// Time to process rewrites

	for _, slot := range rewriteSlots {
		// delete slot metrics
		s.dbClient.DeleteBlockMetrics(slot)
		// write slot metrics
		s.ProcessBlock(slot)

	}

	for _, epoch := range rewriteEpochs {
		// delete epoch metrics
		s.dbClient.DeleteStateMetrics(epoch)
		// write epoch metrics
		s.ProcessStateTransitionMetrics(epoch - 1)
		s.ProcessStateTransitionMetrics(epoch)
		s.ProcessStateTransitionMetrics(epoch + 1)
		s.ProcessStateTransitionMetrics(epoch + 2)
	}

	// Delete from History

	for _, epoch := range stateKeys {
		if epoch >= uint64(finalizedEpoch) {
			continue // only process epochs that are before the finalized
		}
		s.queue.StateHistory.Delete(epoch)
		// loop over slots in the epoch
		for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {
			s.queue.BlockHistory.Delete(slot)
		}
	}

	log.Infof("checked states until slot %d, epoch %d", newFinalizedSlot, newFinalizedSlot/spec.SlotsPerEpoch)
}
