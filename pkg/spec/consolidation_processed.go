package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ConsolidationProcessed struct {
	Epoch              phase0.Epoch
	Index              uint64
	SourceIndex        phase0.ValidatorIndex
	TargetIndex        phase0.ValidatorIndex
	ConsolidatedAmount phase0.Gwei
	Valid              bool
}

func (f ConsolidationProcessed) Type() ModelType {
	return ConsolidationRequestModel
}

func (f ConsolidationProcessed) ToArray() []interface{} {
	rows := []interface{}{
		f.Epoch,
		f.Index,
		f.SourceIndex,
		f.TargetIndex,
		f.ConsolidatedAmount,
	}
	return rows
}
