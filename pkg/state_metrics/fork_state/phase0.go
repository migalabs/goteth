package fork_state

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
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

	balances := make([]phase0.Gwei, 0)

	for _, item := range bstate.Phase0.Balances {
		balances = append(balances, phase0.Gwei(item))
	}

	phase0Obj := ForkStateContentBase{
		Version:          bstate.Version,
		Balances:         balances,
		Validators:       bstate.Phase0.Validators,
		EpochStructs:     NewEpochData(iApi, bstate.Phase0.Slot),
		Epoch:            phase0.Epoch(bstate.Phase0.Slot / SLOTS_PER_EPOCH),
		Slot:             phase0.Slot(bstate.Phase0.Slot),
		BlockRoots:       bstate.Phase0.BlockRoots,
		PrevAttestations: bstate.Phase0.PreviousEpochAttestations,
	}

	phase0Obj.Setup()

	return phase0Obj

}
