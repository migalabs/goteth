package spec

import (
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ConsolidationRequest struct {
	Slot          phase0.Slot
	SourceAddress bellatrix.ExecutionAddress
	SourcePubkey  phase0.BLSPubKey
	TargetPubkey  phase0.BLSPubKey
}

func (f ConsolidationRequest) Type() ModelType {
	return ConsolidationRequestModel
}

func (f ConsolidationRequest) ToArray() []interface{} {
	rows := []interface{}{
		f.Slot,
		f.SourceAddress,
		f.SourcePubkey,
		f.TargetPubkey,
	}
	return rows
}
