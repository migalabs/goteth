package custom_spec

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/prysmaticlabs/go-bitfield"
)

type BellatrixSpec struct {
	BState     spec.VersionedBeaconState
	Committees map[string]bitfield.Bitlist
}

func NewBellatrixSpec(bstate *spec.VersionedBeaconState) BellatrixSpec {
	bellatrixObj := BellatrixSpec{
		BState:     *bstate,
		Committees: make(map[string]bitfield.Bitlist),
	}
	bellatrixObj.CalculatePreviousEpochAttestations()

	return bellatrixObj
}

func (p BellatrixSpec) ObtainCurrentSlot() uint64 {
	return p.BState.Bellatrix.Slot
}

func (p BellatrixSpec) ObtainCurrentEpoch() uint64 {
	return uint64(p.ObtainCurrentSlot() / 32)
}

func (p BellatrixSpec) CalculatePreviousEpochAttestations() {
}

func (p BellatrixSpec) ObtainPreviousEpochAttestations() uint64 {
	return 0
}

func (p BellatrixSpec) ObtainPreviousEpochValNum() uint64 {
	return 0
}

func (p BellatrixSpec) ObtainBalance(valIdx uint64) (uint64, error) {
	if uint64(len(p.BState.Bellatrix.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.BState.Bellatrix.Slot)
		return 0, err
	}
	balance := p.BState.Bellatrix.Balances[valIdx]

	return balance, nil
}
