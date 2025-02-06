package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type Deposit struct {
	Slot                  phase0.Slot
	PublicKey             phase0.BLSPubKey
	WithdrawalCredentials []byte
	Amount                phase0.Gwei
	Signature             phase0.BLSSignature
	Index                 uint8
}

func (f Deposit) Type() ModelType {
	return DepositModel
}

func (f Deposit) ToArray() []interface{} {
	rows := []interface{}{
		f.Slot,
		f.PublicKey,
		f.WithdrawalCredentials,
		f.Amount,
		f.Signature,
		f.Index,
	}
	return rows
}
