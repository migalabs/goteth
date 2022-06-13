package custom_spec

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/prysmaticlabs/go-bitfield"
)

type AltairSpec struct {
	BState     spec.VersionedBeaconState
	Committees map[string]bitfield.Bitlist
}

func NewAltairSpec(bstate *spec.VersionedBeaconState) AltairSpec {
	altairObj := AltairSpec{
		BState:     *bstate,
		Committees: make(map[string]bitfield.Bitlist),
	}
	altairObj.CalculatePreviousEpochAttestations()

	return altairObj
}

func (p AltairSpec) ObtainCurrentSlot() uint64 {
	return p.BState.Altair.Slot
}

func (p AltairSpec) ObtainCurrentEpoch() uint64 {
	return uint64(p.ObtainCurrentSlot() / 32)
}

func (p AltairSpec) CalculatePreviousEpochAttestations() {
}

func (p AltairSpec) ObtainPreviousEpochAttestations() uint64 {
	return 0
}

func (p AltairSpec) ObtainPreviousEpochValNum() uint64 {
	return 0
}

func (p AltairSpec) ObtainBalance(valIdx uint64) (uint64, error) {
	if uint64(len(p.BState.Altair.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.BState.Altair.Slot)
		return 0, err
	}
	balance := p.BState.Altair.Balances[valIdx]

	return balance, nil
}
