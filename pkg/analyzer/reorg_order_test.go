package analyzer

import (
	"errors"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var errDeleteFailed = errors.New("delete failed")

// HandleReorg walks slots from the head downwards, so it meets a higher epoch
// before the lower ones that epoch derives from. Rewriting in that order
// computes each epoch from predecessors the walk has not corrected yet, so the
// epochs are collected during the walk and ordered before anything is written.

func TestCollectedEpochsAreRewrittenLowestFirst(t *testing.T) {
	// The order a backward walk produces.
	walked := []phase0.Epoch{102, 101, 100}

	got := ascendingEpochs(walked)

	want := []phase0.Epoch{100, 101, 102}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("processing order %v, want %v", got, want)
		}
	}
}

func TestOrderingLeavesTheCollectedSliceAlone(t *testing.T) {
	// The caller's slice records what the walk found. Sorting it in place would
	// make the record disagree with the walk.
	walked := []phase0.Epoch{102, 101, 100}

	ascendingEpochs(walked)

	for i, want := range []phase0.Epoch{102, 101, 100} {
		if walked[i] != want {
			t.Errorf("the collected slice was reordered: %v", walked)
			break
		}
	}
}

func TestOrderingHandlesNothingToRewrite(t *testing.T) {
	// A reorg that replaced no state at all is the common case: block roots
	// changed, state roots did not.
	if got := ascendingEpochs(nil); len(got) != 0 {
		t.Errorf("got %v, want nothing to rewrite", got)
	}
	if got := ascendingEpochs([]phase0.Epoch{}); len(got) != 0 {
		t.Errorf("got %v, want nothing to rewrite", got)
	}
}

func TestOrderingKeepsEveryEpochItWasGiven(t *testing.T) {
	// Dropping one would leave an epoch deleted-but-not-rewritten, which is the
	// orphaning this branch exists to stop.
	walked := []phase0.Epoch{5, 3, 4, 1, 2}

	got := ascendingEpochs(walked)

	if len(got) != len(walked) {
		t.Fatalf("got %d epochs, want %d: %v", len(got), len(walked), got)
	}
	seen := make(map[phase0.Epoch]bool, len(got))
	for _, e := range got {
		seen[e] = true
	}
	for _, e := range walked {
		if !seen[e] {
			t.Errorf("epoch %d was collected but would never be rewritten", e)
		}
	}
}

func TestOrderingIsAscendingWhateverTheInput(t *testing.T) {
	// A reorg spanning several epochs can meet them in any order once the walk
	// crosses boundaries, so the ordering must not assume the input is reversed.
	for _, input := range [][]phase0.Epoch{
		{1, 2, 3},
		{3, 1, 2},
		{7},
		{9, 9, 8},
	} {
		got := ascendingEpochs(input)
		for i := 1; i < len(got); i++ {
			if got[i-1] > got[i] {
				t.Errorf("input %v produced %v, which is not ascending", input, got)
				break
			}
		}
	}
}

// record captures the sequence of operations rewriteInOrder performs, so the
// order and the pairing can be asserted without a database.
type record struct{ steps []string }

func (r *record) del(e phase0.Epoch) error {
	r.steps = append(r.steps, "delete "+epochStr(e))
	return nil
}

func (r *record) process(e phase0.Epoch) {
	r.steps = append(r.steps, "process "+epochStr(e))
}

func epochStr(e phase0.Epoch) string {
	return string(rune('0' + int(e)))
}

func TestEachEpochIsDeletedImmediatelyBeforeItIsRewritten(t *testing.T) {
	// Deleting everything up front would leave every epoch after the first
	// deleted and not yet rewritten for as long as the rewrites take, which is
	// the orphaning this branch exists to stop.
	var r record

	rewriteInOrder([]phase0.Epoch{3, 1, 2}, r.del, r.process)

	want := []string{
		"delete 1", "process 1",
		"delete 2", "process 2",
		"delete 3", "process 3",
	}
	if len(r.steps) != len(want) {
		t.Fatalf("got %v, want %v", r.steps, want)
	}
	for i := range want {
		if r.steps[i] != want[i] {
			t.Fatalf("got %v, want %v", r.steps, want)
		}
	}
}

func TestARewriteStillHappensWhenTheDeleteFails(t *testing.T) {
	// A failed delete must not skip the rewrite. Skipping would leave the
	// pre-reorg rows in place with nothing to correct them, which is the silent
	// wrong value this whole change is about.
	var processed []phase0.Epoch
	failing := func(phase0.Epoch) error { return errDeleteFailed }

	rewriteInOrder([]phase0.Epoch{4}, failing, func(e phase0.Epoch) {
		processed = append(processed, e)
	})

	if len(processed) != 1 || processed[0] != 4 {
		t.Errorf("epoch 4 was not rewritten after its delete failed: %v", processed)
	}
}

func TestNothingIsTouchedWhenNoStateWasReplaced(t *testing.T) {
	var r record

	rewriteInOrder(nil, r.del, r.process)

	if len(r.steps) != 0 {
		t.Errorf("a reorg that replaced no state still touched the database: %v", r.steps)
	}
}
