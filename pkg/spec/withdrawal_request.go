package spec

import (
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type WithdrawalRequestResult uint8

const (
	WithdrawalRequestResultSuccess WithdrawalRequestResult = iota
	WithdrawalRequestResultQueueFull
	WithdrawalRequestResultValidatorNotFound
	WithdrawalRequestResultInvalidCredentials
	WithdrawalRequestResultValidatorNotActive
	WithdrawalRequestResultExitAlreadyInitiated
	WithdrawalRequestResultValidatorNotOldEnough
	WithdrawalRequestResultPendingWithdrawalExists
	WithdrawalRequestResultInsufficientBalance
	WithdrawalRequestResultValidatorNotCompounding
	WithdrawalRequestResultNoExcessBalance
)

type WithdrawalRequest struct {
	Slot            phase0.Slot
	Index           uint64
	SourceAddress   bellatrix.ExecutionAddress
	ValidatorPubkey phase0.BLSPubKey
	Amount          phase0.Gwei
	Result          WithdrawalRequestResult
}

func (f WithdrawalRequest) Type() ModelType {
	return WithdrawalRequestModel
}

func (f WithdrawalRequest) ToArray() []interface{} {
	rows := []interface{}{
		f.Slot,
		f.Index,
		f.SourceAddress,
		f.ValidatorPubkey,
		f.Amount,
		f.Result,
	}
	return rows
}
