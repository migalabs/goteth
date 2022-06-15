package custom_spec

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
	"github.com/prysmaticlabs/go-bitfield"
)

type BellatrixSpec struct {
	BState      spec.VersionedBeaconState
	Committees  map[string]bitfield.Bitlist
	DoubleVotes uint64
}

func NewBellatrixSpec(bstate *spec.VersionedBeaconState) BellatrixSpec {
	bellatrixObj := BellatrixSpec{
		BState:      *bstate,
		Committees:  make(map[string]bitfield.Bitlist),
		DoubleVotes: 0,
	}
	bellatrixObj.PreviousEpochAttestations()

	return bellatrixObj
}

func (p BellatrixSpec) CurrentSlot() uint64 {
	return p.BState.Bellatrix.Slot
}

func (p BellatrixSpec) CurrentEpoch() uint64 {
	return uint64(p.CurrentSlot() / 32)
}

func (p BellatrixSpec) PreviousEpochAttestations() uint64 {
	numOfAttestations := 0
	attestationsPerVal := p.BState.Bellatrix.PreviousEpochParticipation

	for _, item := range attestationsPerVal {
		// Here we have one item per validator
		// This is an 8-bit string, where the bit 0 is the source
		// If it is set, we consider there was a vote from this validator
		if utils.IsBitSet(uint8(item), 0) {
			numOfAttestations++
		}
	}

	return uint64(numOfAttestations)
}

func (p BellatrixSpec) PreviousEpochMissedAttestations() uint64 {
	numOfMissedAttestations := 0
	attestationsPerVal := p.BState.Bellatrix.PreviousEpochParticipation

	for _, item := range attestationsPerVal {
		// Here we have one item per validator
		if !utils.IsBitSet(uint8(item), 0) {
			numOfMissedAttestations++
		}
	}

	return uint64(numOfMissedAttestations)

}

func (p BellatrixSpec) PreviousEpochValNum() uint64 {
	vals := p.BState.Bellatrix.Validators
	totalAttestingVals := 0

	for _, item := range vals {
		// validator must be either active, exiting or slashed
		if item.ActivationEligibilityEpoch < phase0.Epoch(p.CurrentEpoch()) &&
			item.ExitEpoch > phase0.Epoch(p.CurrentEpoch()) {
			totalAttestingVals += 1
		}
	}
	return uint64(totalAttestingVals)
}

func (p BellatrixSpec) GetDoubleVotes() uint64 {
	return p.DoubleVotes
}

func (p BellatrixSpec) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.BState.Bellatrix.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.BState.Bellatrix.Slot)
		return 0, err
	}
	balance := p.BState.Bellatrix.Balances[valIdx]

	return balance, nil
}
