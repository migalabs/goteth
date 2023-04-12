package model

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
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
	Status              int
}

func (f ValidatorRewards) InsertOp() bool {
	return true
}

func (f ValidatorRewards) DropOp() bool {
	return false
}

func (f ValidatorRewards) BalanceToEth() float32 {
	return float32(f.ValidatorBalance) / utils.EFFECTIVE_BALANCE_INCREMENT
}
