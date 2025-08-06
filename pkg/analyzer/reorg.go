package analyzer

import (
	"fmt"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
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
		cacheState := s.downloadCache.StateHistory.Wait(epoch)
		finalizedStateRoot := s.cli.RequestStateRoot(phase0.Slot(cacheState.Slot))
		cacheStateRoot := cacheState.StateRoot

		if finalizedStateRoot != cacheStateRoot { // no match, reorg happened
			log.Warnf("cache state root: %s\nfinalized block root: %s", cacheStateRoot, finalizedStateRoot)
			log.Warnf("state root for state (slot=%d) incorrect, redownload", cacheState.Slot)

			s.dbClient.DeleteStateMetrics(phase0.Epoch(epoch))
			log.Infof("rewriting metrics for epoch %d", epoch)
			// write epoch metrics
			s.ProcessStateTransitionMetrics(phase0.Epoch(epoch))
		}

		// loop over slots in the epoch
		for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {

			// Retrieve stored root and redownload root once finalized
			cacheBlock := s.downloadCache.BlockHistory.Wait(slot)
			finalizedBlockRoot := s.cli.RequestBlockRoot(phase0.Slot(cacheBlock.Slot))
			cacheBlockRoot := cacheBlock.Root

			if finalizedBlockRoot != cacheBlockRoot {
				log.Warnf("cache block root: %s\nfinalized block root: %s", cacheBlockRoot, finalizedBlockRoot)
				log.Warnf("block root for block (slot=%d) incorrect, redownload", cacheBlock.Slot)

				s.dbClient.DeleteBlockMetrics(phase0.Slot(slot))
				log.Infof("rewriting metrics for slot %d", slot)
				// write slot metrics
				s.ProcessBlock(phase0.Slot(slot))
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

	reorgedSlots := uint64(0)

	cacheHeadBlock := s.downloadCache.GetHeadBlock()
	i := cacheHeadBlock.Slot

	// CRITICAL FIX: Track epochs that need state metric recalculation
	epochsToRecalculate := make(map[phase0.Epoch]bool)

	for reorgedSlots <= depth { // for every slot in the reorg

		block := s.downloadCache.BlockHistory.Wait(SlotTo[uint64](i)) // first check that it was already in the cache
		if i < reorgSlot && block.Proposed {
			reorgedSlots += 1 // only count as reorged slot if there was a block porposed and we are not at the reorg slot
		}
		s.processerBook.WaitUntilInactive(fmt.Sprintf("%s%d", slotProcesserTag, i)) // wait until has been processed
		oldBlock := *block

		s.DownloadBlock(i) // -> inserts into the queue and replaces old block
		newBlock := s.downloadCache.BlockHistory.Wait(SlotTo[uint64](i))

		if newBlock.Root != oldBlock.Root { // only rewrite if stateroots are different
			if block.Proposed { // keep orphans -> if previous block was proposed and roots have changed
				s.dbClient.PersistOrphans([]spec.AgnosticBlock{oldBlock})
			}
			s.dbClient.DeleteBlockMetrics(i)
			log.Infof("rewriting metrics for slot %d", i)
			// write slot metrics
			s.ProcessBlock(i)
		} else {
			log.Infof("reorg slot %d: block roots are the same", i)
		}

		if (i+1)%spec.SlotsPerEpoch == 0 { // then we are at the end of the epoch, rewrite state
			epoch := phase0.Epoch(i / spec.SlotsPerEpoch)

			state := s.downloadCache.StateHistory.Wait(EpochTo[uint64](epoch))           // first check that it was already in the cache
			s.processerBook.WaitUntilInactive(fmt.Sprintf("%s%d", epochProcesserTag, i)) // wait until has been processed
			oldState := *state
			s.DownloadState(i) // -> inserts into the queue and replaces old block
			newState := s.downloadCache.StateHistory.Wait(EpochTo[uint64](epoch))

			if newState.StateRoot != oldState.StateRoot {
				// CRITICAL FIX: Mark this epoch and related epochs for recalculation
				epochsToRecalculate[epoch] = true
				if epoch >= 1 {
					epochsToRecalculate[epoch-1] = true // prevState dependency
				}
				if epoch >= 2 {
					epochsToRecalculate[epoch-2] = true // might affect rewards calculation
				}
				epochsToRecalculate[epoch+1] = true // nextState dependency
				epochsToRecalculate[epoch+2] = true // might affect rewards calculation

				log.Infof("state root changed for epoch %d, marking related epochs for recalculation", epoch)
			}
		}
		i -= 1
	}

	// CRITICAL FIX: Recalculate state metrics for all affected epochs atomically
	// This ensures consistent state transitions across epoch boundaries
	for epoch := range epochsToRecalculate {
		log.Infof("recalculating state metrics for epoch %d due to reorg", epoch)
		s.dbClient.DeleteStateMetrics(epoch)
		s.ProcessStateTransitionMetrics(epoch)
	}

}
