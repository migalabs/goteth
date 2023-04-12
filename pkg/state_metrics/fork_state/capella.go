package fork_state

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// This Wrapper is meant to include all necessary data from the Capella Fork
func NewCapellaState(bstate spec.VersionedBeaconState, iApi *http.Service) ForkStateContentBase {

	capellaObj := ForkStateContentBase{
		Version:       bstate.Version,
		Balances:      bstate.Capella.Balances,
		Validators:    bstate.Capella.Validators,
		EpochStructs:  NewEpochData(iApi, uint64(bstate.Capella.Slot)),
		Epoch:         phase0.Epoch(bstate.Capella.Slot / SLOTS_PER_EPOCH),
		Slot:          bstate.Capella.Slot,
		BlockRoots:    RootToByte(bstate.Capella.BlockRoots),
		SyncCommittee: *bstate.Capella.CurrentSyncCommittee,
	}

	capellaObj.Setup()

	ProcessAttestations(&capellaObj, bstate.Capella.PreviousEpochParticipation)

	return capellaObj
}
