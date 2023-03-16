package fork_state

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

// This Wrapper is meant to include all necessary data from the Capella Fork
func NewCapellaState(bstate spec.VersionedBeaconState, iApi *http.Service) ForkStateContentBase {

	capellaObj := ForkStateContentBase{
		Version:       bstate.Version,
		Balances:      GweiToUint64(bstate.Capella.Balances),
		Validators:    bstate.Capella.Validators,
		EpochStructs:  NewEpochData(iApi, uint64(bstate.Capella.Slot)),
		Epoch:         utils.GetEpochFromSlot(uint64(bstate.Capella.Slot)),
		Slot:          uint64(bstate.Capella.Slot),
		BlockRoots:    RootToByte(bstate.Capella.BlockRoots),
		SyncCommittee: *bstate.Capella.CurrentSyncCommittee,
	}

	capellaObj.Setup()

	ProcessAttestations(&capellaObj, bstate.Capella.PreviousEpochParticipation)

	return capellaObj
}
