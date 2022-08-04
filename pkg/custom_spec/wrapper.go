package custom_spec

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type ForkStateWrapper struct {
	BState             spec.VersionedBeaconState
	PrevBState         spec.VersionedBeaconState
	PrevEpochStructs   EpochData
	EpochStructs       EpochData
	Api                *http.Service
	TotalActiveBalance uint64
}
