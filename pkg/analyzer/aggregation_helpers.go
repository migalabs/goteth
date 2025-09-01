package analyzer

import (
    "github.com/attestantio/go-eth2-client/spec/phase0"
    local_spec "github.com/migalabs/goteth/pkg/spec"
)

// testAggregate aggregates a single validator reward under lock.
// It is intended for testing concurrency safety of the aggregation map.
func (s *ChainAnalyzer) testAggregate(val local_spec.ValidatorRewards) {
    s.rewardsAggMu.Lock()
    if _, ok := s.validatorsRewardsAggregations[val.ValidatorIndex]; !ok {
        s.validatorsRewardsAggregations[val.ValidatorIndex] = local_spec.NewValidatorRewardsAggregation(val.ValidatorIndex, s.startEpochAggregation, s.endEpochAggregation)
    }
    s.validatorsRewardsAggregations[val.ValidatorIndex].Aggregate(val)
    s.rewardsAggMu.Unlock()
}

// testSnapshotIfEpoch snapshots and rotates the aggregation when the given epoch matches endEpochAggregation.
// Intended for tests, mirrors the production logic.
func (s *ChainAnalyzer) testSnapshotIfEpoch(epoch phase0.Epoch) map[phase0.ValidatorIndex]*local_spec.ValidatorRewardsAggregation {
    if s.rewardsAggregationEpochs <= 1 || epoch != s.endEpochAggregation {
        return nil
    }
    s.rewardsAggMu.Lock()
    snapshot := s.validatorsRewardsAggregations
    s.validatorsRewardsAggregations = make(map[phase0.ValidatorIndex]*local_spec.ValidatorRewardsAggregation)
    s.startEpochAggregation = s.endEpochAggregation + 1
    s.endEpochAggregation = s.endEpochAggregation + phase0.Epoch(s.rewardsAggregationEpochs)
    s.rewardsAggMu.Unlock()
    return snapshot
}

