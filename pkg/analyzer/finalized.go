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
			log.Warnf("state root for state (slot=%d) incorrect, redownload", queueState.Slot)
			// need to redownload the epoch
			//s.downloadCache.StateHistory.Delete(epoch)
			s.DownloadState(queueState.Slot) // -> inserts into the queue

			// keep track of the rewrite metrics
			rewriteEpochs = append(rewriteEpochs, phase0.Epoch(epoch))
		}

		// loop over slots in the epoch
		for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {

			// Retrieve stored root and redownload root once finalized
			queueBlock := s.downloadCache.BlockHistory.Wait(slot)
			finalizedBlockRoot := s.cli.RequestStateRoot(phase0.Slot(queueBlock.Slot))
			historyBlockRoot := queueBlock.StateRoot

			if finalizedBlockRoot != historyBlockRoot {
				log.Warnf("state root for block (slot=%d) incorrect, redownload", queueBlock.Slot)
				// need to redownload the epoch
				//s.downloadCache.BlockHistory.Delete(slot)
				s.DownloadBlock(phase0.Slot(slot)) // -> inserts into the queue

				// keep track of the rewrite metrics
				rewriteSlots = append(rewriteSlots, phase0.Slot(slot))

				// keep orphans
				if queueBlock.Proposed {
					s.dbClient.Persist(db.OrphanBlock(*queueBlock))
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

	s.downloadCache.CleanUpTo(newFinalizedSlot)

	if advance {
		log.Infof("checked states until slot %d, epoch %d", newFinalizedSlot, newFinalizedSlot/spec.SlotsPerEpoch)

	}
}
