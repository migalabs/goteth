package analyzer

import "testing"

// AdvanceFinalized rewrites epoch E, which leaves rows derived from E stale:
// the epoch row for E is written by processing E+1, and the validator rewards
// for E+1 and E+2 by processing E+1 and E+2. Epochs at or past the finalized
// boundary are skipped, and the in-function flags that would have marked them
// do not survive the call. Without carrying that debt, nothing ever rewrites
// them (migalabs/goteth#285).

func TestDependentsBelowTheBoundaryAreNotCarried(t *testing.T) {
	// The loop reached them itself, so there is no debt - and a debt below the
	// boundary could not be paid anyway: CleanUpTo evicts those states at the
	// end of the invocation, so no later loop ever visits them again.
	s := &ChainAnalyzer{}

	s.carryStaleDependents(map[uint64]bool{100: true}, 200)

	if len(s.pendingReprocess) != 0 {
		t.Errorf("carried %v; those epochs were processed in the same invocation", s.pendingReprocess)
	}
}

func TestDependentsAtTheBoundaryAreCarried(t *testing.T) {
	// Blocks changed at 199 while 200 is the first unfinalized epoch, so 200
	// and 201 were skipped and their rows are stale.
	s := &ChainAnalyzer{}

	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	for _, epoch := range []uint64{200, 201} {
		if !s.pendingReprocess[epoch] {
			t.Errorf("epoch %d was not carried; nothing would ever rewrite its rows", epoch)
		}
	}
}

func TestOnlyTheUnreachableDependentIsCarried(t *testing.T) {
	// 199 is below the boundary and was processed; 200 was not.
	s := &ChainAnalyzer{}

	s.carryStaleDependents(map[uint64]bool{198: true}, 200)

	if s.pendingReprocess[199] {
		t.Error("199 is below the boundary and was already processed; carrying it repeats work")
	}
	if !s.pendingReprocess[200] {
		t.Error("200 is past the boundary and its rows are stale; it must be carried")
	}
}

// A carried epoch must be reprocessed once and then forgotten. Leaving the debt
// in place would retry it on every finalized event for the life of the process.
func TestACarriedEpochIsConsumedExactlyOnce(t *testing.T) {
	s := &ChainAnalyzer{}
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	if !s.consumeCarriedEpoch(200) {
		t.Fatal("the carried epoch was not reported as needing reprocessing")
	}
	if s.consumeCarriedEpoch(200) {
		t.Error("the same epoch was reported twice; the debt was never cleared")
	}
}

func TestAnEpochThatWasNeverCarriedIsNotReprocessed(t *testing.T) {
	s := &ChainAnalyzer{}

	if s.consumeCarriedEpoch(200) {
		t.Error("an epoch nothing made stale was queued for reprocessing")
	}
}

// A state root mismatch makes AdvanceFinalized delete and re-download the state
// for that epoch, even when none of its blocks changed. The rows derived from
// that state go stale exactly as they would after a block change, so the carry
// keys on the state having been replaced rather than on the blocks.
func TestAStateReplacedWithoutBlockChangesIsStillCarried(t *testing.T) {
	s := &ChainAnalyzer{}

	// What AdvanceFinalized records for an epoch where only the root differed.
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	for _, epoch := range []uint64{200, 201} {
		if !s.pendingReprocess[epoch] {
			t.Errorf("epoch %d was not carried after the state at 199 was replaced; "+
				"its rows keep pre-reorg values and nothing rewrites them", epoch)
		}
	}
}

// Consuming a carried epoch must not enqueue its own dependents: it is being
// reprocessed because a predecessor changed, not because its own blocks did.
// Propagating from it would cascade forward with no end.
func TestConsumingACarriedEpochDoesNotCascade(t *testing.T) {
	s := &ChainAnalyzer{}
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	s.consumeCarriedEpoch(200)
	s.consumeCarriedEpoch(201)

	// The consumed epochs are not added to epochsWithChangedBlocks, so the
	// next invocation carries nothing on their behalf.
	s.carryStaleDependents(map[uint64]bool{}, 202)

	if len(s.pendingReprocess) != 0 {
		t.Errorf("consuming carried epochs left %v queued", s.pendingReprocess)
	}
}
