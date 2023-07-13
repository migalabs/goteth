package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ValidatorLastStatus struct {
	ValIdx          phase0.ValidatorIndex
	Epoch           phase0.Epoch
	CurrentBalance  phase0.Gwei
	CurrentStatus   ValidatorStatus
	Slashed         bool
	ActivationEpoch phase0.Epoch
	WithdrawalEpoch phase0.Epoch
	ExitEpoch       phase0.Epoch
	PublicKey       phase0.BLSPubKey
}

func (f ValidatorLastStatus) Type() ModelType {
	return ValidatorLastStatusModel
}

func (f ValidatorLastStatus) BalanceToEth() float32 {
	return float32(f.CurrentBalance) / EffectiveBalanceInc
}
