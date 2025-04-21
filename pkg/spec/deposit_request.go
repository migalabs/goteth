package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type DepositRequest struct {
	Slot                  phase0.Slot
	Pubkey                phase0.BLSPubKey
	WithdrawalCredentials []byte
	Amount                phase0.Gwei
	Signature             phase0.BLSSignature
	Index                 uint64
}

func (f DepositRequest) Type() ModelType {
	return DepositRequestModel
}

func (f DepositRequest) ToArray() []any {
	rows := []any{
		f.Slot,
		f.Index,
		f.Pubkey,
		f.WithdrawalCredentials,
		f.Amount,
		f.Signature,
	}
	return rows
}
