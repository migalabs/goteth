package analyzer

import (
	"fmt"
	"sort"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s *ChainAnalyzer) AdvanceFinalized(newFinalizedSlot phase0.Slot) {

	// Each FinalizedCheckpointEvent fires a new `go AdvanceFinalized(...)`. If a
	// previous invocation is still running, two goroutines race over the same
	// StateHistory: the new one calls CleanUpTo and evicts entries that the old
	// one is still blocked on inside StateHistory.Wait/BlockHistory.Wait, which
	// then deadlocks holding processerBook slots until the whole pool is leaked.
	// Skip overlapping invocations — the next finalized event will pick up any
	// epochs the skipped one would have processed (the iteration is monotonic
	// over GetKeyList()).
	if !s.advanceFinalizedMu.TryLock() {
		log.Infof("AdvanceFinalized already in progress, skipping invocation for slot %d", newFinalizedSlot)
		return
	}
	defer s.advanceFinalizedMu.Unlock()

	finalizedEpoch := newFinalizedSlot / spec.SlotsPerEpoch

	stateKeys := s.downloadCache.StateHistory.GetKeyList()

	// Sort keys so we process epochs in ascending order. This guarantees
	// that when we reach epoch E, we already know whether blocks in
	// earlier epochs (E-1, E-2) changed and can propagate reprocessing.
	sort.Slice(stateKeys, func(i, j int) bool { return stateKeys[i] < stateKeys[j] })

	advance := false
	epochsWithChangedBlocks := make(map[uint64]bool)
	// Epochs whose state object this invocation replaced, by re-download or by
	// refreshing its blocks. The rows derived from those states are what goes
	// stale, so this, not epochsWithChangedBlocks, is what gets carried.
	epochsWithChangedState := make(map[uint64]bool)

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
			// Refreshing the blocks below rewrites the derived fields of state
			// epoch, so record it here rather than after the state root check:
			// that check can fail and skip the rest of this iteration.
			epochsWithChangedState[epoch] = true
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
		if stateRootChanged {
			epochsWithChangedState[epoch] = true
		}

		needsReprocess := stateRootChanged || blocksChanged
		if epoch >= 1 && epochsWithChangedBlocks[epoch-1] {
			needsReprocess = true
		}
		if epoch >= 2 && epochsWithChangedBlocks[epoch-2] {
			needsReprocess = true
		}
		if s.consumeCarriedEpoch(epoch) {
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

	s.carryStaleDependents(epochsWithChangedState, uint64(finalizedEpoch))

	s.downloadCache.CleanUpTo(newFinalizedSlot)

	if advance {
		log.Infof("checked states until slot %d, epoch %d", newFinalizedSlot, newFinalizedSlot/spec.SlotsPerEpoch)
	}
}

// consumeCarriedEpoch reports whether this epoch was carried over by an earlier
// invocation that rewrote one of its predecessors, and clears the debt.
//
// The debt is cleared whether or not the rest of the iteration succeeds. An
// epoch that keeps failing would otherwise be retried on every finalized event
// for the lifetime of the process.
//
// Callers must hold advanceFinalizedMu.
func (s *ChainAnalyzer) consumeCarriedEpoch(epoch uint64) bool {
	if !s.pendingReprocess[epoch] {
		return false
	}
	delete(s.pendingReprocess, epoch)
	log.Infof("reprocessing epoch %d carried over from an earlier AdvanceFinalized", epoch)
	return true
}

// carryStaleDependents records the epochs whose derived rows this invocation
// made stale but could not rewrite itself.
//
// Replacing the state at epoch E leaves rows belonging to later epochs stale:
// the epoch row for E is written by processing E+1 (ExportToEpoch reads
// CurrentState), and the validator rewards for E+1 and E+2 by processing E+1
// and E+2, both of which read state E. AdvanceFinalized reprocesses those when
// its loop reaches them, but it skips everything at or past the finalized
// boundary, and the flags that would have marked them do not survive the call,
// so the next invocation no longer knows they are stale. Carrying them is what
// makes a later invocation rewrite them (#285).
//
// The caller passes the epochs whose state object it replaced, whether by
// re-downloading it after a state root mismatch or by refreshing its blocks.
// Both rewrite state E, and it is state E the stale rows derive from. Epochs
// reprocessed only because a predecessor changed do not belong here: their own
// state is untouched, so nothing downstream of them went stale.
//
// Only dependents at or past the boundary are carried. Anything below it the
// loop already reprocessed, and carrying it would repeat the work.
//
// Callers must hold advanceFinalizedMu.
func (s *ChainAnalyzer) carryStaleDependents(changed map[uint64]bool, finalizedEpoch uint64) {
	if s.pendingReprocess == nil {
		s.pendingReprocess = make(map[uint64]bool)
	}
	for epoch := range changed {
		for _, dependent := range []uint64{epoch + 1, epoch + 2} {
			if dependent >= finalizedEpoch {
				s.pendingReprocess[dependent] = true
			}
		}
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

	// Epochs whose state the reorg replaced, collected here and rewritten after
	// the walk. See rewriteStateMetrics for why not during it.
	replaced := make([]phase0.Epoch, 0)

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

			// Only note it here. Rewriting the metrics now would read the two
			// preceding states, which sit at lower slots this walk has not
			// reached yet and so are still the pre-reorg copies.
			if newState.StateRoot != oldState.StateRoot {
				replaced = append(replaced, epoch)
			}
		}
		i -= 1
	}

	s.rewriteStateMetrics(replaced)
}

// rewriteStateMetrics rewrites the epoch metrics for every state the reorg
// replaced, once every replacement has been downloaded and in ascending epoch
// order.
//
// Both conditions matter, for different reasons.
//
// The walk above runs backwards, so it reaches a higher epoch before the lower
// ones it derives from. ProcessStateTransitionMetrics(E) reads the states for
// E, E-1 and E-2 out of the cache, so rewriting E during the walk computes it
// from whichever of those the walk has not replaced yet: the stale pre-reorg
// copies. Deferring every rewrite until the walk has finished means each one
// reads states that have all been brought up to date.
//
// Ascending order is the second half. ProcessStateTransitionMetrics already
// waits for epoch-1 to finish before starting, because ProcessAttestations
// writes ManualReward onto block objects that the following epoch reads back
// (goteth#242). Running E before E-1 satisfies that barrier trivially - E-1 has
// not started - and then lets E-1 mutate blocks E has already read.
func (s *ChainAnalyzer) rewriteStateMetrics(epochs []phase0.Epoch) {
	rewriteInOrder(epochs, s.dbClient.DeleteStateMetrics, s.ProcessStateTransitionMetrics)
}

// rewriteInOrder deletes and rewrites each epoch's metrics, lowest first, and
// deletes an epoch's rows immediately before rewriting them rather than
// clearing everything up front. Taking the two apart would leave every epoch
// after the first deleted and not yet written, which is the orphaning this
// branch exists to stop, for as long as the rewrites take.
//
// The delete and the process are passed in so the order can be asserted without
// a database or a beacon node.
func rewriteInOrder(epochs []phase0.Epoch, del func(phase0.Epoch) error, process func(phase0.Epoch)) {
	for _, epoch := range ascendingEpochs(epochs) {
		if err := del(epoch); err != nil {
			log.Errorf("could not delete metrics for epoch %d before rewriting them: %s", epoch, err)
		}
		log.Infof("rewriting metrics for epoch %d", epoch)
		process(epoch)
	}
}

// ascendingEpochs returns the epochs in the order they must be processed. The
// reorg walk collects them from the head downwards, which is the reverse of the
// order each epoch's inputs become correct in.
//
// It copies rather than sorting in place: the caller's slice records what the
// walk found, and reordering that under it would make the two disagree.
func ascendingEpochs(epochs []phase0.Epoch) []phase0.Epoch {
	ordered := make([]phase0.Epoch, len(epochs))
	copy(ordered, epochs)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered
}
