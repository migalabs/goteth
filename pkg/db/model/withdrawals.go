package model

import (
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type Withdrawal struct {
	Slot           phase0.Slot
	Index          capella.WithdrawalIndex
	ValidatorIndex phase0.ValidatorIndex
	Address        bellatrix.ExecutionAddress
	Amount         phase0.Gwei
}

func (f Withdrawal) Type() ModelType {
	return WithdrawalModel
}
