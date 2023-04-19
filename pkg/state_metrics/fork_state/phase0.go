package fork_state

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

// This Wrapper is meant to include all necessary data from the Phase0 Fork
func NewPhase0State(bstate spec.VersionedBeaconState, iApi *http.Service) ForkStateContentBase {

	balances := make([]phase0.Gwei, 0)

	for _, item := range bstate.Phase0.Balances {
		balances = append(balances, phase0.Gwei(item))
	}

	phase0Obj := ForkStateContentBase{
		Version:          bstate.Version,
		Balances:         balances,
		Validators:       bstate.Phase0.Validators,
		EpochStructs:     NewEpochData(iApi, bstate.Phase0.Slot),
		Epoch:            phase0.Epoch(bstate.Phase0.Slot / utils.SLOTS_PER_EPOCH),
		Slot:             phase0.Slot(bstate.Phase0.Slot),
		BlockRoots:       bstate.Phase0.BlockRoots,
		PrevAttestations: bstate.Phase0.PreviousEpochAttestations,
	}

	phase0Obj.Setup()

	return phase0Obj

}
