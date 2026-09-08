package analyzer

import "testing"

// AdvanceFinalized rewrites epoch E, which leaves rows derived from E stale:
// the epoch row for E is written by processing E+1, and the validator rewards
// for E+1 and E+2 by processing E+1 and E+2. Epochs at or past the finalized
// boundary are skipped, and the in-function flags that would have marked them
// do not survive the call. Without carrying that debt, nothing ever rewrites
// them (migalabs/goteth#285).

func TestDependentsBelowTheBoundaryAreNotCarried(t *testing.T) {
	// The loop reached them itself, so there is no debt to record. Note the
	// second reason this once had - that a debt below the boundary could never
	// be paid, because CleanUpTo evicts those states - no longer holds:
	// epochsToVisit re-adds an evicted debt and ensureDependencyStates
	// re-downloads it. Not carrying these is now purely about not repeating
	// work already done in this invocation.
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

// A carried epoch that was actually rewritten must be forgotten. Leaving the
// debt in place would retry it on every finalized event for the life of the
// process.
func TestACarriedEpochIsForgottenOnceItIsWritten(t *testing.T) {
	s := &ChainAnalyzer{}
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	if !s.carriedEpoch(200) {
		t.Fatal("the carried epoch was not reported as needing reprocessing")
	}
	s.settleReprocess(200, true)

	if s.carriedEpoch(200) {
		t.Error("the same epoch was reported again after being written; the debt was never cleared")
	}
}

// The failure this exists for: the debt used to be cleared when the epoch was
// reached, before anything was known to have been written. A rewrite that wrote
// nothing then left the rows deleted and no debt to retry, so the hole was
// permanent (migalabs/goteth#291).
func TestAnEpochThatWroteNothingKeepsItsDebt(t *testing.T) {
	s := &ChainAnalyzer{}
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	if !s.carriedEpoch(200) {
		t.Fatal("the carried epoch was not reported as needing reprocessing")
	}
	s.settleReprocess(200, false)

	if !s.carriedEpoch(200) {
		t.Error("the debt was cleared even though nothing was written; " +
			"the rows are deleted and nothing will ever rewrite them")
	}
}

// Reading the debt must not clear it, or the caller cannot report an outcome.
func TestReadingTheDebtDoesNotClearIt(t *testing.T) {
	s := &ChainAnalyzer{}
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	if !s.carriedEpoch(200) || !s.carriedEpoch(200) {
		t.Error("reading the debt cleared it, so a failed rewrite could not carry it forward")
	}
}

// An unpayable debt must not be retried forever: the epoch would be
// re-downloaded and re-deleted on every finalized event for the life of the
// process. It is abandoned loudly instead, leaving a hole, which is detectable.
func TestAnUnpayableDebtIsAbandonedAfterABoundedNumberOfAttempts(t *testing.T) {
	s := &ChainAnalyzer{}
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	for attempt := 1; attempt < maxReprocessAttempts; attempt++ {
		s.settleReprocess(200, false)
		if !s.carriedEpoch(200) {
			t.Fatalf("the debt was dropped after %d attempts, before the bound", attempt)
		}
	}
	s.settleReprocess(200, false)

	if s.carriedEpoch(200) {
		t.Errorf("the debt survived %d attempts; it would be retried on every "+
			"finalized event forever", maxReprocessAttempts)
	}
}

// A retry that succeeds must reset the count, or an epoch that fails
// intermittently exhausts its budget across unrelated incidents.
func TestASuccessfulRetryResetsTheAttemptCount(t *testing.T) {
	s := &ChainAnalyzer{}
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	s.settleReprocess(200, false)
	s.settleReprocess(200, true)

	if s.reprocessAttempts[200] != 0 {
		t.Errorf("attempt count survived a successful write: %d", s.reprocessAttempts[200])
	}
}

func TestAnEpochThatWasNeverCarriedIsNotReprocessed(t *testing.T) {
	s := &ChainAnalyzer{}

	if s.carriedEpoch(200) {
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
func TestSettlingACarriedEpochDoesNotCascade(t *testing.T) {
	s := &ChainAnalyzer{}
	s.carryStaleDependents(map[uint64]bool{199: true}, 200)

	s.settleReprocess(200, true)
	s.settleReprocess(201, true)

	// The consumed epochs are not added to epochsWithChangedBlocks, so the
	// next invocation carries nothing on their behalf.
	s.carryStaleDependents(map[uint64]bool{}, 202)

	if len(s.pendingReprocess) != 0 {
		t.Errorf("settling carried epochs left %v queued", s.pendingReprocess)
	}
}
