package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ValidatorRewards struct {
	ValidatorIndex       phase0.ValidatorIndex
	Epoch                phase0.Epoch
	ValidatorBalance     phase0.Gwei
	Reward               int64 // it can be negative
	MaxReward            phase0.Gwei
	AttestationReward    phase0.Gwei
	SyncCommitteeReward  phase0.Gwei
	BaseReward           phase0.Gwei
	AttSlot              phase0.Slot
	AttestationIncluded  bool
	InSyncCommittee      bool
	ProposerSlot         phase0.Slot
	ProposerApiReward    phase0.Gwei
	ProposerManualReward phase0.Gwei
	MissingSource        bool
	MissingTarget        bool
	MissingHead          bool
	Status               ValidatorStatus
	InclusionDelay       int
}

func (f ValidatorRewards) Type() ModelType {
	return ValidatorRewardsModel
}

func (f ValidatorRewards) BalanceToEth() float32 {
	return float32(f.ValidatorBalance) / EffectiveBalanceInc
}

func (f ValidatorRewards) ToArray() []interface{} {
	rows := []interface{}{
		f.ValidatorIndex,
		f.Epoch,
		f.BalanceToEth(),
		f.Reward,
		f.MaxReward,
		f.AttestationReward,
		f.SyncCommitteeReward,
		f.BaseReward,
		f.AttSlot,
		f.AttestationIncluded,
		f.InSyncCommittee,
		f.ProposerSlot,
		f.ProposerApiReward,
		f.ProposerManualReward,
		f.MissingSource,
		f.MissingTarget,
		f.MissingHead,
		f.Status,
		f.InclusionDelay,
	}
	return rows
}
