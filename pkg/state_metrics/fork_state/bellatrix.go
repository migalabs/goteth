package fork_state

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

// This Wrapper is meant to include all necessary data from the Bellatrix Fork
func NewBellatrixState(bstate spec.VersionedBeaconState, iApi *http.Service) ForkStateContentBase {

	bellatrixObj := ForkStateContentBase{
		Version:       bstate.Version,
		Balances:      GweiToUint64(bstate.Bellatrix.Balances),
		Validators:    bstate.Bellatrix.Validators,
		EpochStructs:  NewEpochData(iApi, uint64(bstate.Bellatrix.Slot)),
		Epoch:         utils.GetEpochFromSlot(uint64(bstate.Bellatrix.Slot)),
		Slot:          uint64(bstate.Bellatrix.Slot),
		BlockRoots:    RootToByte(bstate.Bellatrix.BlockRoots),
		SyncCommittee: *bstate.Bellatrix.CurrentSyncCommittee,
	}

	bellatrixObj.Setup()

	ProcessAttestations(&bellatrixObj, bstate.Bellatrix.PreviousEpochParticipation)

	return bellatrixObj
}
