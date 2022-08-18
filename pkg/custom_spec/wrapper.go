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
	CorrectFlags       [][]bool
	MissedBlocks       []uint64 // array that stores the slot number where there was a missing block
}

func (p *ForkStateContent) InitializeArrays(arrayLen uint64) {
	p.CorrectFlags = make([][]bool, 3)
	p.MissedBlocks = make([]uint64, 0)

	for i := range p.CorrectFlags {
		p.CorrectFlags[i] = make([]bool, arrayLen)
	}
}
