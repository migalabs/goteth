package fork_state

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// This Wrapper is meant to include all necessary data from the Bellatrix Fork
func NewBellatrixState(bstate spec.VersionedBeaconState, iApi *http.Service) ForkStateContentBase {

	bellatrixObj := ForkStateContentBase{
		Version:       bstate.Version,
		Balances:      bstate.Bellatrix.Balances,
		Validators:    bstate.Bellatrix.Validators,
		EpochStructs:  NewEpochData(iApi, uint64(bstate.Bellatrix.Slot)),
		Epoch:         phase0.Epoch(bstate.Bellatrix.Slot / SLOTS_PER_EPOCH),
		Slot:          bstate.Bellatrix.Slot,
		BlockRoots:    RootToByte(bstate.Bellatrix.BlockRoots),
		SyncCommittee: *bstate.Bellatrix.CurrentSyncCommittee,
	}

	bellatrixObj.Setup()

	ProcessAttestations(&bellatrixObj, bstate.Bellatrix.PreviousEpochParticipation)

	return bellatrixObj
}
