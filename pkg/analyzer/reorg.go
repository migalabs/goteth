package analyzer

import (
	"fmt"
	"sort"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s *ChainAnalyzer) AdvanceFinalized(newFinalizedSlot phase0.Slot) {

	finalizedEpoch := newFinalizedSlot / spec.SlotsPerEpoch

	stateKeys := s.downloadCache.StateHistory.GetKeyList()

	// Sort keys so we process epochs in ascending order. This guarantees
	// that when we reach epoch E, we already know whether blocks in
	// earlier epochs (E-1, E-2) changed and can propagate reprocessing.
	sort.Slice(stateKeys, func(i, j int) bool { return stateKeys[i] < stateKeys[j] })

	advance := false
	epochsWithChangedBlocks := make(map[uint64]bool)

	for _, epoch := range stateKeys {
		if epoch >= uint64(finalizedEpoch) {
			continue // only process epochs that are before the given epoch
		}
		advance = true // only set flag if there is something to do

		// --- Step 1: verify block roots FIRST ---
		// Blocks must be checked before state metrics are (re)processed,
		// because state processing reads block data (e.g. isFlagPossible
		// uses prevState.Blocks to decide the head attester reward).
		blocksChanged := false
		for slot := (epoch * spec.SlotsPerEpoch); slot < ((epoch + 1) * spec.SlotsPerEpoch); slot++ {

			cacheBlock, err := s.downloadCache.BlockHistory.Wait(s.ctx, slot)
			if err != nil {
				log.Errorf("context cancelled waiting for block at slot %d: %s", slot, err)
				return
			}
			finalizedBlockRoot := s.cli.RequestBlockRoot(phase0.Slot(cacheBlock.Slot))
			cacheBlockRoot := cacheBlock.Root

			if finalizedBlockRoot != cacheBlockRoot {
				log.Warnf("cache block root: %s\nfinalized block root: %s", cacheBlockRoot, finalizedBlockRoot)
				log.Warnf("block root for block (slot=%d) incorrect, redownload", cacheBlock.Slot)

				s.dbClient.DeleteBlockMetrics(phase0.Slot(slot))
				log.Infof("rewriting metrics for slot %d", slot)
				s.DownloadBlock(phase0.Slot(slot))
				s.ProcessBlock(phase0.Slot(slot))
				blocksChanged = true
			}
		}

		if blocksChanged {
			epochsWithChangedBlocks[epoch] = true
			// Refresh the state's Blocks array so it points to the newly
			// downloaded block objects instead of the stale pre-reorg ones.
			if err := s.downloadCache.RefreshStateBlocks(s.ctx, epoch); err != nil {
				log.Errorf("failed to refresh state blocks for epoch %d: %s", epoch, err)
			}
		}

		// --- Step 2: verify state root ---
		cacheState, err := s.downloadCache.StateHistory.Wait(s.ctx, epoch)
		if err != nil {
			log.Errorf("context cancelled waiting for state at epoch %d: %s", epoch, err)
			return
		}
		finalizedStateRoot, err := s.cli.RequestStateRoot(phase0.Slot(cacheState.Slot))
		if err != nil {
			log.Errorf("could not get state root at slot %d: %s", cacheState.Slot, err)
			continue
		}

		stateRootChanged := finalizedStateRoot != cacheState.StateRoot

		// Determine if state metrics need reprocessing.
		// ProcessStateTransitionMetrics(E) uses three states:
		//   prevState  (E-2) — validator rewards via isFlagPossible
		//   currentState (E-1) — block rewards
		//   nextState  (E)   — proposer duties, epoch metrics
		// If blocks changed in any of those epochs, the derived metrics
		// for epoch E are stale and must be recomputed.
		needsReprocess := stateRootChanged || blocksChanged
		if epoch >= 1 && epochsWithChangedBlocks[epoch-1] {
			needsReprocess = true
		}
		if epoch >= 2 && epochsWithChangedBlocks[epoch-2] {
			needsReprocess = true
		}

		if needsReprocess {
			if stateRootChanged {
				log.Warnf("cache state root: %s\nfinalized state root: %s", cacheState.StateRoot, finalizedStateRoot)
				log.Warnf("state root for state (slot=%d) incorrect, redownloading", cacheState.Slot)

				// Evict the stale state from the in-memory cache and
				// re-download from the beacon node (now finalized).
				// Without this, StateHistory.Wait() returns the same
				// wrong state and the analyzer loops forever.
				stateSlot := phase0.Slot(cacheState.Slot)
				s.downloadCache.StateHistory.Delete(epoch)
				s.DownloadState(stateSlot)
			}

			// ProcessStateTransitionMetrics(E) calls StateHistory.Wait for
			// epochs E, E-1 and E-2. Those dependency states may have been
			// evicted by a previous CleanUpTo call, which would cause Wait
			// to block forever. Re-download any that are missing. (#245)
			s.ensureDependencyStates(epoch)

			s.dbClient.DeleteStateMetrics(phase0.Epoch(epoch))
			log.Infof("rewriting metrics for epoch %d (stateRootChanged=%t, blocksChanged=%t, dep=%t)",
				epoch, stateRootChanged, blocksChanged,
				(epoch >= 1 && epochsWithChangedBlocks[epoch-1]) || (epoch >= 2 && epochsWithChangedBlocks[epoch-2]))
			s.ProcessStateTransitionMetrics(phase0.Epoch(epoch))
		}
	}

	s.downloadCache.CleanUpTo(newFinalizedSlot)

	if advance {
		log.Infof("checked states until slot %d, epoch %d", newFinalizedSlot, newFinalizedSlot/spec.SlotsPerEpoch)
	}
}

// ensureDependencyStates checks that the states required by
// ProcessStateTransitionMetrics (epoch, epoch-1, epoch-2) are present in the
// in-memory cache. Any missing state is re-downloaded from the beacon node,
// along with its epoch's blocks (needed by AddNewState).
func (s *ChainAnalyzer) ensureDependencyStates(epoch uint64) {
	depEpochs := []uint64{epoch}
	if epoch >= 1 {
		depEpochs = append(depEpochs, epoch-1)
	}
	if epoch >= 2 {
		depEpochs = append(depEpochs, epoch-2)
	}

	initEpoch := uint64(s.initSlot / spec.SlotsPerEpoch)
	for _, dep := range depEpochs {
		if dep < initEpoch {
			continue
		}
		if s.downloadCache.StateHistory.Available(dep) {
			continue
		}
		log.Infof("dependency state for epoch %d missing from cache, re-downloading", dep)
		// Ensure blocks exist first — AddNewState calls BlockHistory.Wait
		// for every slot in the epoch.
		for slot := dep * spec.SlotsPerEpoch; slot < (dep+1)*spec.SlotsPerEpoch; slot++ {
			if !s.downloadCache.BlockHistory.Available(slot) {
				s.DownloadBlock(phase0.Slot(slot))
			}
		}
		depSlot := phase0.Slot((dep+1)*spec.SlotsPerEpoch - 1)
		s.DownloadState(depSlot)
	}
}

func (s *ChainAnalyzer) HandleReorg(newReorg v1.ChainReorgEvent) {
	depth := newReorg.Depth
	reorgSlot := newReorg.Slot

	reorgedSlots := uint64(0)

	cacheHeadBlock := s.downloadCache.GetHeadBlock()
	i := cacheHeadBlock.Slot

	for reorgedSlots <= depth { // for every slot in the reorg

		block, err := s.downloadCache.BlockHistory.Wait(s.ctx, SlotTo[uint64](i)) // first check that it was already in the cache
		if err != nil {
			log.Errorf("context cancelled waiting for block at slot %d: %s", i, err)
			return
		}
		if i < reorgSlot && block.Proposed {
			reorgedSlots += 1 // only count as reorged slot if there was a block porposed and we are not at the reorg slot
		}
		s.processerBook.WaitUntilInactive(fmt.Sprintf("%s%d", slotProcesserTag, i)) // wait until has been processed
		oldBlock := *block

		s.DownloadBlock(i) // -> inserts into the queue and replaces old block
		newBlock, err := s.downloadCache.BlockHistory.Wait(s.ctx, SlotTo[uint64](i))
		if err != nil {
			log.Errorf("context cancelled waiting for block at slot %d: %s", i, err)
			return
		}

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

			state, err := s.downloadCache.StateHistory.Wait(s.ctx, EpochTo[uint64](epoch)) // first check that it was already in the cache
			if err != nil {
				log.Errorf("context cancelled waiting for state at epoch %d: %s", epoch, err)
				return
			}
			s.processerBook.WaitUntilInactive(fmt.Sprintf("%s%d", epochProcesserTag, i)) // wait until has been processed
			oldState := *state
			s.DownloadState(i) // -> inserts into the queue and replaces old block
			newState, err := s.downloadCache.StateHistory.Wait(s.ctx, EpochTo[uint64](epoch))
			if err != nil {
				log.Errorf("context cancelled waiting for state at epoch %d: %s", epoch, err)
				return
			}

			if newState.StateRoot != oldState.StateRoot {
				s.dbClient.DeleteStateMetrics(epoch)
				log.Infof("rewriting metrics for epoch %d", epoch)
				// write epoch metrics
				s.ProcessStateTransitionMetrics(epoch)
			}
		}
		i -= 1
	}

}
