package fork_state

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
)

const (
	MAX_EFFECTIVE_INCREMENTS    = 32
	BASE_REWARD_FACTOR          = 64
	BASE_REWARD_PER_EPOCH       = 4
	EFFECTIVE_BALANCE_INCREMENT = 1000000000
	SLOTS_PER_EPOCH             = 32
	SHUFFLE_ROUND_COUNT         = uint64(90)
	PROPOSER_REWARD_QUOTIENT    = 8
	GENESIS_EPOCH               = 0
	SLOTS_PER_HISTORICAL_ROOT   = 8192
)

// This Wrapper is meant to include all necessary data from the Phase0 Fork
func NewPhase0State(bstate spec.VersionedBeaconState, iApi *http.Service) ForkStateContentBase {

	phase0Obj := ForkStateContentBase{
		Version:          bstate.Version,
		Balances:         bstate.Phase0.Balances,
		Validators:       bstate.Phase0.Validators,
		EpochStructs:     NewEpochData(iApi, bstate.Phase0.Slot),
		Epoch:            utils.GetEpochFromSlot(bstate.Phase0.Slot),
		Slot:             bstate.Phase0.Slot,
		BlockRoots:       bstate.Phase0.BlockRoots,
		PrevAttestations: bstate.Phase0.PreviousEpochAttestations,
	}

	phase0Obj.Setup()

	return phase0Obj

}
