package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ValidatorRewards struct {
	ValidatorIndex                      phase0.ValidatorIndex
	Epoch                               phase0.Epoch
	ValidatorBalance                    phase0.Gwei
	EffectiveBalance                    phase0.Gwei
	WithdrawalPrefix                    byte
	Reward                              int64 // it can be negative
	MaxReward                           phase0.Gwei
	AttestationReward                   phase0.Gwei
	SyncCommitteeReward                 phase0.Gwei
	AttSlot                             phase0.Slot
	AttestationIncluded                 bool
	BaseReward                          phase0.Gwei
	InSyncCommittee                     bool
	SyncCommitteeParticipationsIncluded uint8
	ProposerSlot                        phase0.Slot
	MissingSource                       bool
	MissingTarget                       bool
	MissingHead                         bool
	Status                              ValidatorStatus
	ProposerApiReward                   phase0.Gwei
	ProposerManualReward                phase0.Gwei
	InclusionDelay                      int
}

func (f ValidatorRewards) Type() ModelType {
	return ValidatorRewardsModel
}

func (f ValidatorRewards) BalanceToEth() float32 {
	return float32(f.ValidatorBalance) / EffectiveBalanceInc
}

func (f ValidatorRewards) ToArray() []any {
	rows := []any{
		f.ValidatorIndex,
		f.Epoch,
		f.BalanceToEth(),
		f.EffectiveBalance,
		f.WithdrawalPrefix,
		f.Reward,
		f.MaxReward,
		f.AttestationReward,
		f.SyncCommitteeReward,
		f.BaseReward,
		f.AttSlot,
		f.AttestationIncluded,
		f.InSyncCommittee,
		f.SyncCommitteeParticipationsIncluded,
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
