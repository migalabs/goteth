package spec

import (
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type BLSToExecutionChange struct {
	Slot               phase0.Slot
	Epoch              phase0.Epoch
	ValidatorIndex     phase0.ValidatorIndex
	FromBLSPublicKey   phase0.BLSPubKey
	ToExecutionAddress bellatrix.ExecutionAddress
}

func (f BLSToExecutionChange) Type() ModelType {
	return BLSToExecutionChangeModel
}

func (f BLSToExecutionChange) ToArray() []interface{} {
	rows := []interface{}{
		f.Slot,
		f.Epoch,
		f.ValidatorIndex,
		f.FromBLSPublicKey.String(),
		f.ToExecutionAddress.String(),
	}
	return rows
}
