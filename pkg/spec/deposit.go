package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type Deposit struct {
	Slot                  phase0.Slot
	EpochProcessed        phase0.Epoch
	PublicKey             phase0.BLSPubKey
	WithdrawalCredentials []byte
	Amount                phase0.Gwei
	Signature             phase0.BLSSignature
	Index                 uint8
}

func (f Deposit) Type() ModelType {
	return DepositModel
}

func (f Deposit) ToArray() []any {
	rows := []any{
		f.Slot,
		f.EpochProcessed,
		f.PublicKey,
		f.WithdrawalCredentials,
		f.Amount,
		f.Signature,
		f.Index,
	}
	return rows
}
