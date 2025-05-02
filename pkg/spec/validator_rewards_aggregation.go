package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ValidatorRewardsAggregation struct {
	ValidatorIndex                      phase0.ValidatorIndex
	StartEpoch                          phase0.Epoch
	EndEpoch                            phase0.Epoch // Inclusive
	Reward                              int64        // it can be negative
	MaxReward                           phase0.Gwei
	MaxAttestationReward                phase0.Gwei
	MaxSyncCommitteeReward              phase0.Gwei
	BaseReward                          phase0.Gwei
	InSyncCommitteeCount                uint16
	SyncCommitteeParticipationsIncluded uint16
	AttestationsIncluded                uint16
	MissingSourceCount                  uint16
	MissingTargetCount                  uint16
	MissingHeadCount                    uint16
	ProposerApiReward                   phase0.Gwei
	ProposerManualReward                phase0.Gwei
	InclusionDelaySum                   uint32
}

func NewValidatorRewardsAggregation(validatorIndex phase0.ValidatorIndex, startEpoch phase0.Epoch, endEpoch phase0.Epoch) *ValidatorRewardsAggregation {
	return &ValidatorRewardsAggregation{
		ValidatorIndex: validatorIndex,
		StartEpoch:     startEpoch,
		EndEpoch:       endEpoch,
	}
}

func (f ValidatorRewardsAggregation) Type() ModelType {
	return ValidatorRewardsAggregationModel
}

func (f ValidatorRewardsAggregation) ToArray() []any {
	rows := []any{
		f.ValidatorIndex,
		f.StartEpoch,
		f.EndEpoch,
		f.Reward,
		f.MaxReward,
		f.MaxAttestationReward,
		f.MaxSyncCommitteeReward,
		f.BaseReward,
		f.InSyncCommitteeCount,
		f.SyncCommitteeParticipationsIncluded,
		f.AttestationsIncluded,
		f.MissingSourceCount,
		f.MissingTargetCount,
		f.MissingHeadCount,
		f.ProposerApiReward,
		f.ProposerManualReward,
		f.InclusionDelaySum,
	}
	return rows
}

func (f *ValidatorRewardsAggregation) Aggregate(valRewards ValidatorRewards) {
	f.Reward += valRewards.Reward
	f.MaxReward += valRewards.MaxReward
	f.MaxAttestationReward += valRewards.AttestationReward
	f.MaxSyncCommitteeReward += valRewards.SyncCommitteeReward
	f.BaseReward += valRewards.BaseReward
	if valRewards.InSyncCommittee {
		f.InSyncCommitteeCount++
	}
	f.SyncCommitteeParticipationsIncluded += uint16(valRewards.SyncCommitteeParticipationsIncluded)
	if valRewards.AttestationIncluded {
		f.AttestationsIncluded++
	}
	if valRewards.MissingSource {
		f.MissingSourceCount++
	}
	if valRewards.MissingTarget {
		f.MissingTargetCount++
	}
	if valRewards.MissingHead {
		f.MissingHeadCount++
	}
	f.ProposerApiReward += valRewards.ProposerApiReward
	f.ProposerManualReward += valRewards.ProposerManualReward
	f.InclusionDelaySum += uint32(valRewards.InclusionDelay)
}
