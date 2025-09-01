package analyzer

import (
    "sync"
    "testing"

    "github.com/attestantio/go-eth2-client/spec/phase0"
    local_spec "github.com/migalabs/goteth/pkg/spec"
)

func TestRewardsAggregationConcurrentAccess(t *testing.T) {
    s := &ChainAnalyzer{
        rewardsAggregationEpochs:      3,
        startEpochAggregation:         10,
        endEpochAggregation:           12,
        validatorsRewardsAggregations: make(map[phase0.ValidatorIndex]*local_spec.ValidatorRewardsAggregation),
    }

    // Prepare concurrent workers aggregating for 5 validators
    const goroutines = 8
    const iterations = 1000
    counts := make(map[phase0.ValidatorIndex]int)
    var mu sync.Mutex

    var wg sync.WaitGroup
    wg.Add(goroutines)
    for g := 0; g < goroutines; g++ {
        go func(id int) {
            defer wg.Done()
            for i := 0; i < iterations; i++ {
                idx := phase0.ValidatorIndex(id % 5)
                vr := local_spec.ValidatorRewards{ValidatorIndex: idx, MaxReward: 1}
                s.testAggregate(vr)
                mu.Lock()
                counts[idx]++
                mu.Unlock()
            }
        }(g)
    }
    wg.Wait()

    // Snapshot at end epoch
    snap := s.testSnapshotIfEpoch(12)
    if snap == nil {
        t.Fatalf("expected snapshot at epoch 12, got nil")
    }
    if len(snap) != 5 {
        t.Fatalf("expected 5 validators in snapshot, got %d", len(snap))
    }
    for idx, c := range counts {
        if agg, ok := snap[idx]; !ok {
            t.Fatalf("missing validator %d in snapshot", idx)
        } else if int(agg.MaxReward) != c {
            t.Fatalf("validator %d: expected MaxReward=%d, got %d", idx, c, agg.MaxReward)
        }
    }
}

func TestFinalizeOnlyOnce(t *testing.T) {
    s := &ChainAnalyzer{
        rewardsAggregationEpochs:      2,
        startEpochAggregation:         20,
        endEpochAggregation:           21,
        validatorsRewardsAggregations: make(map[phase0.ValidatorIndex]*local_spec.ValidatorRewardsAggregation),
    }
    // add some data
    s.testAggregate(local_spec.ValidatorRewards{ValidatorIndex: 1, MaxReward: 5})
    s.testAggregate(local_spec.ValidatorRewards{ValidatorIndex: 2, MaxReward: 7})

    snap1 := s.testSnapshotIfEpoch(21)
    if snap1 == nil || len(snap1) != 2 {
        t.Fatalf("expected non-nil snapshot of size 2, got %+v", snap1)
    }
    // Second call for same epoch should return nil (range rotated)
    snap2 := s.testSnapshotIfEpoch(21)
    if snap2 != nil {
        t.Fatalf("expected nil snapshot on second call, got %+v", snap2)
    }
    // Ensure range rotated
    if s.startEpochAggregation != 22 || s.endEpochAggregation != 23 {
        t.Fatalf("unexpected aggregation range after rotate: start=%d end=%d", s.startEpochAggregation, s.endEpochAggregation)
    }
}

