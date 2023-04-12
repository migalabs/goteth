package state

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"
)

type SummaryMetrics struct {
	AvgReward          float64
	AvgMaxReward       float64
	AvgAttMaxReward    float64
	AvgSyncMaxReward   float64
	AvgBaseReward      float64
	MissingSourceCount uint64
	MissingTargetCount uint64
	MissingHeadCount   uint64

	NumActiveVals      uint64
	NumNonProposerVals uint64
	NumSyncVals        uint64
}

func NewSummaryMetrics() SummaryMetrics {
	return SummaryMetrics{}
}

func (s *SummaryMetrics) AddMetrics(
	maxRewards model.ValidatorRewards,
	stateMetrics state_metrics.StateMetrics,
	valIdx phase0.ValidatorIndex,
	validatorDBRow model.ValidatorRewards) {
	if maxRewards.ProposerSlot == 0 { // there is no proposer at slot 0
		// only do rewards statistics in case the validator is not a proposer
		// right now we cannot measure the max reward for a proposer

		// process batch metrics
		s.AvgReward += float64(stateMetrics.GetMetricsBase().EpochReward(valIdx))
		s.AvgMaxReward += float64(maxRewards.MaxReward)
		s.NumNonProposerVals += 1
	}
	if maxRewards.InSyncCommittee {
		s.NumSyncVals += 1
		s.AvgSyncMaxReward += float64(maxRewards.SyncCommitteeReward)
	}

	s.AvgBaseReward += float64(maxRewards.BaseReward)

	// in case of Phase0 AttestationRewards and InlusionDelay is filled
	// in case of Altair, only FlagIndexReward is filled
	// TODO: we might need to do the same for single validator rewards
	s.AvgAttMaxReward += float64(maxRewards.AttestationReward)

	if fork_state.IsActive(*stateMetrics.GetMetricsBase().NextState.Validators[valIdx],
		phase0.Epoch(stateMetrics.GetMetricsBase().NextState.Epoch)) {
		s.NumActiveVals += 1

		if validatorDBRow.MissingSource {
			s.MissingSourceCount += 1
		}

		if validatorDBRow.MissingTarget {
			s.MissingTargetCount += 1
		}

		if validatorDBRow.MissingHead {
			s.MissingHeadCount += 1
		}
	}
}

func (s *SummaryMetrics) Aggregate() {
	// calculate averages
	s.AvgReward = s.AvgReward / float64(s.NumNonProposerVals)
	s.AvgMaxReward = s.AvgMaxReward / float64(s.NumNonProposerVals)

	s.AvgBaseReward = s.AvgBaseReward / float64(s.NumActiveVals)
	s.AvgAttMaxReward = s.AvgAttMaxReward / float64(s.NumActiveVals)
	s.AvgSyncMaxReward = s.AvgSyncMaxReward / float64(s.NumSyncVals)

	// sanitize in case of division by 0
	if s.NumActiveVals == 0 {
		s.AvgBaseReward = 0
		s.AvgAttMaxReward = 0
	}

	if s.NumNonProposerVals == 0 {
		// al validators are proposers, therefore average rewards cannot be calculated
		// (we still cannot calulate proposer max rewards)
		s.AvgReward = 0
		s.AvgMaxReward = 0
	}

	// avoid division by 0
	if s.NumSyncVals == 0 {
		s.AvgSyncMaxReward = 0
	}
}
