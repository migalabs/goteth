package fork_state

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/require"
)

func TestState(t *testing.T) {

	balancesArray := make([]uint64, 0)
	balancesArray = append(balancesArray, 34000000000, 31200000000)
	validator1 := phase0.Validator{
		EffectiveBalance: 32000000000,
		ActivationEpoch:  0,
		ExitEpoch:        10000000000,
	}

	validator2 := phase0.Validator{
		EffectiveBalance: 31000000000,
		ActivationEpoch:  0,
		ExitEpoch:        10000000000,
	}

	validatorArray := make([]*phase0.Validator, 0)
	validatorArray = append(validatorArray, &validator1, &validator2)

	state := ForkStateContentBase{
		Validators: validatorArray,
		Balances:   balancesArray,
		BlockRoots: make([][]byte, 0),
	}
	state.Setup()

	require.Equal(t, state.TotalActiveBalance, uint64(validator1.EffectiveBalance+validator2.EffectiveBalance))

	state.Validators = append(state.Validators, &validator1, &validator2)

	state = ForkStateContentBase{
		Validators: validatorArray,
		Balances:   balancesArray,
		BlockRoots: make([][]byte, 0),
	}
	state.Setup()
	attestations := make([]altair.ParticipationFlags, 0)
	attestations = append(attestations, altair.ParticipationFlags(7))

	ProcessAttestations(&state, attestations)

	require.Equal(t, state.AttestingBalance[0], uint64(validator1.EffectiveBalance))
	require.Equal(t, state.AttestingVals[0], true)
	require.Equal(t, state.AttestingVals[1], false)

	attestations = append(attestations, altair.ParticipationFlags(7))
	state = ForkStateContentBase{
		Validators: validatorArray,
		Balances:   balancesArray,
		BlockRoots: make([][]byte, 0),
	}
	state.Setup()
	ProcessAttestations(&state, attestations)

	require.Equal(t, state.AttestingBalance[0], uint64(validator1.EffectiveBalance+validator2.EffectiveBalance))
	require.Equal(t, state.AttestingVals[0], true)
	require.Equal(t, state.AttestingVals[1], true)

	state = ForkStateContentBase{
		Validators: validatorArray,
		Balances:   balancesArray,
		BlockRoots: make([][]byte, 0),
	}
	state.Setup()
	attestations[1] = altair.ParticipationFlags(6)
	ProcessAttestations(&state, attestations)

	// no source attesting
	require.Equal(t, state.AttestingBalance[0], uint64(validator1.EffectiveBalance))
	require.Equal(t, state.AttestingVals[0], true)
	require.Equal(t, state.AttestingVals[1], true)

}
