package model

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var (
	ValidatorRewardsModelOps = map[string]bool{
		INSERT_OP: true,
		DROP_OP:   false,
	}
)

type ValidatorRewards struct {
	ValidatorIndex      phase0.ValidatorIndex
	Epoch               phase0.Epoch
	ValidatorBalance    phase0.Gwei
	Reward              phase0.Gwei
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
	Status              int
}

func (f ValidatorRewards) InsertOp() bool {
	return true
}

func (f ValidatorRewards) DropOp() bool {
	return false
}
