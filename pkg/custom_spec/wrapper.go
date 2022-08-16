package custom_spec

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type ForkStateContent struct {
	BState             spec.VersionedBeaconState
	PrevBState         spec.VersionedBeaconState
	PrevEpochStructs   EpochData
	EpochStructs       EpochData
	Api                *http.Service
	TotalActiveBalance uint64
	MissingFlags       []uint64
}

func (p *ForkStateContent) InitializeArrays(arrayLen uint64) {
	p.MissingFlags = make([]uint64, 3)

}
