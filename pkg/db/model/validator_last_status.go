package model

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

type ValidatorLastStatus struct {
	ValIdx         phase0.ValidatorIndex
	Epoch          phase0.Epoch
	CurrentBalance phase0.Gwei
	CurrentStatus  int
}

func (f ValidatorLastStatus) InsertOp() bool {
	return true
}

func (f ValidatorLastStatus) DropOp() bool {
	return false
}

func (f ValidatorLastStatus) BalanceToEth() float32 {
	return float32(f.CurrentBalance) / utils.EFFECTIVE_BALANCE_INCREMENT
}
