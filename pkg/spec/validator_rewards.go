package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ValidatorRewards struct {
	ValidatorIndex      phase0.ValidatorIndex
	Epoch               phase0.Epoch
	ValidatorBalance    phase0.Gwei
	Reward              int64 // it can be negative
	MaxReward           phase0.Gwei
	AttestationReward   phase0.Gwei
	SyncCommitteeReward phase0.Gwei
	BaseReward          phase0.Gwei
	AttSlot             phase0.Slot
	InSyncCommittee     bool
	ProposerSlot        phase0.Slot
	MissingSource       bool
	MissingTarget       bool
	MissingHead         bool
	Status              ValidatorStatus
}

func (f ValidatorRewards) Type() ModelType {
	return ValidatorRewardsModel
}

func (f ValidatorRewards) BalanceToEth() float32 {
	return float32(f.ValidatorBalance) / EffectiveBalanceInc
}
