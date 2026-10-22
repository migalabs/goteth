package analyzer

import (
	"sort"
	"testing"
)

// The loop iterates what GetKeyList returns. A carried epoch whose state was
// evicted by CleanUpTo is not in that list, so before this it was never
// visited again and its debt could not be paid, no matter how many finalized
// events went by (migalabs/goteth#291).

func inCache(epochs ...uint64) func(uint64) bool {
	present := make(map[uint64]bool, len(epochs))
	for _, e := range epochs {
		present[e] = true
	}
	return func(e uint64) bool { return present[e] }
}

func sorted(in []uint64) []uint64 {
	out := append([]uint64(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func TestAnEvictedCarriedEpochIsVisitedAgain(t *testing.T) {
	got := epochsToVisit([]uint64{300, 301}, map[uint64]bool{200: true}, inCache(300, 301))

	want := []uint64{200, 300, 301}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v; the evicted debt at 200 would never be revisited", sorted(got), want)
	}
	for i, e := range sorted(got) {
		if e != want[i] {
			t.Errorf("got %v, want %v", sorted(got), want)
			break
		}
	}
}

func TestACarriedEpochStillInTheCacheIsNotAddedTwice(t *testing.T) {
	// It is already in the key list; adding it again would process it twice in
	// one invocation, which is how a processerBook slot leaks (#292).
	got := epochsToVisit([]uint64{200, 201}, map[uint64]bool{200: true}, inCache(200, 201))

	if len(got) != 2 {
		t.Errorf("got %v; epoch 200 is in the cache already and was added a second time", sorted(got))
	}
}

func TestNoDebtMeansNothingIsAdded(t *testing.T) {
	got := epochsToVisit([]uint64{200, 201}, nil, inCache(200, 201))

	if len(got) != 2 {
		t.Errorf("got %v, want just the cached epochs", sorted(got))
	}
}

func TestADebtWithAnEmptyCacheIsStillVisited(t *testing.T) {
	// Every state evicted, one debt outstanding: the invocation exists purely
	// to pay it, and an empty key list must not mean an empty loop.
	got := epochsToVisit(nil, map[uint64]bool{200: true}, inCache())

	if len(got) != 1 || got[0] != 200 {
		t.Errorf("got %v, want [200]", got)
	}
}
