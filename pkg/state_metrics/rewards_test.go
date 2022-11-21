package reward_metrics

import (
	"math"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth2-state-analyzer/pkg/reward_metrics/fork_state"
	"github.com/stretchr/testify/require"
)

func TestMaxAttestationReward(t *testing.T) {

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

	state := fork_state.ForkStateContentBase{
		Validators: validatorArray,
		Balances:   balancesArray,
		BlockRoots: make([][]byte, 0),
		Epoch:      10,
	}
	state.Setup()

	balancesArray1 := make([]uint64, 0)
	balancesArray1 = append(balancesArray1, 34000000000, 31200000000)
	validator1 = phase0.Validator{
		EffectiveBalance: 32000000000,
		ActivationEpoch:  0,
		ExitEpoch:        10000000000,
	}

	validator2 = phase0.Validator{
		EffectiveBalance: 31000000000,
		ActivationEpoch:  0,
		ExitEpoch:        10000000000,
	}

	validatorArray1 := make([]*phase0.Validator, 0)
	validatorArray1 = append(validatorArray1, &validator1, &validator2)

	statePrev := fork_state.ForkStateContentBase{
		Validators: validatorArray1,
		Balances:   balancesArray1,
		BlockRoots: make([][]byte, 0),
		Epoch:      9,
	}

	attestations := make([]altair.ParticipationFlags, 0)
	attestations = append(attestations, altair.ParticipationFlags(6))
	statePrev.Setup()
	fork_state.ProcessAttestations(&state, attestations)

	stateNext := statePrev
	stateNext.Epoch = 11
	rewardsObj := NewAltairMetrics(
		statePrev,
		state,
		stateNext)

	baseRewardPerInc := uint64(fork_state.EFFECTIVE_BALANCE_INCREMENT * fork_state.BASE_REWARD_FACTOR)
	baseRewardPerInc = baseRewardPerInc / uint64(math.Sqrt(float64(rewardsObj.CurrentState.TotalActiveBalance)))
	require.Equal(t,
		rewardsObj.GetBaseRewardPerInc(rewardsObj.CurrentState.TotalActiveBalance),
		uint64(baseRewardPerInc))

	require.Equal(t,
		rewardsObj.GetBaseReward(0, uint64(validator1.EffectiveBalance), rewardsObj.CurrentState.TotalActiveBalance),
		uint64(baseRewardPerInc*32))

	require.Equal(t,
		rewardsObj.GetBaseReward(1, uint64(validator2.EffectiveBalance), rewardsObj.CurrentState.TotalActiveBalance),
		uint64(baseRewardPerInc*31))

	attReward := 0

	// Source
	reward := uint64(14) * baseRewardPerInc * 31 * uint64(rewardsObj.CurrentState.AttestingBalance[0]/1000000000)
	reward = reward / (uint64(rewardsObj.CurrentState.TotalActiveBalance / 1000000000)) / 64
	attReward += int(reward)

	// Target
	reward = uint64(26) * baseRewardPerInc * 31 * uint64(rewardsObj.CurrentState.AttestingBalance[1]/1000000000)
	reward = reward / (uint64(rewardsObj.CurrentState.TotalActiveBalance / 1000000000)) / 64
	attReward += int(reward)

	// Head
	reward = uint64(14) * baseRewardPerInc * 31 * uint64(rewardsObj.CurrentState.AttestingBalance[2]/1000000000)
	reward = reward / (uint64(rewardsObj.CurrentState.TotalActiveBalance / 1000000000)) / 64
	attReward += int(reward)

	require.Equal(t,
		rewardsObj.GetMaxAttestationReward(1),
		uint64(attReward))

	attReward = 0

	// Source
	reward = uint64(14) * baseRewardPerInc * 32 * uint64(rewardsObj.CurrentState.AttestingBalance[0]/1000000000)
	reward = reward / (uint64(rewardsObj.CurrentState.TotalActiveBalance / 1000000000)) / 64
	attReward += int(reward)

	// Target
	reward = uint64(26) * baseRewardPerInc * 32 * uint64(rewardsObj.CurrentState.AttestingBalance[1]/1000000000)
	reward = reward / (uint64(rewardsObj.CurrentState.TotalActiveBalance / 1000000000)) / 64
	attReward += int(reward)

	// Head
	reward = uint64(14) * baseRewardPerInc * 32 * uint64(rewardsObj.CurrentState.AttestingBalance[2]/1000000000)
	reward = reward / (uint64(rewardsObj.CurrentState.TotalActiveBalance / 1000000000)) / 64
	attReward += int(reward)
	require.Equal(t,
		rewardsObj.GetMaxAttestationReward(0),
		uint64(2590292))

}

func TestMaxSyncCommitteeReward(t *testing.T) {

	// create state
	balancesArray := make([]uint64, 0)
	balancesArray = append(balancesArray, 34000000000, 31200000000)
	validator1Pubkey := "0x8b9cfaf7480d7bb848cc2017b9770bef00f8d9e761bea9ea06a0534c449a98f50b587db18a93c4da8ae805476edee55a"
	validator1 := phase0.Validator{
		EffectiveBalance: 32000000000,
		ActivationEpoch:  0,
		ExitEpoch:        10000000000,
		PublicKey:        phase0.BLSPubKey{},
	}

	copy(validator1.PublicKey[:], validator1Pubkey)

	validator2Pubkey := "0x9b9cfaf7480d7bb848cc2017b9770bef00f8d9e761bea9ea06a0534c449a98f50b587db18a93c4da8ae805476edee55a"
	validator2 := phase0.Validator{
		EffectiveBalance: 31000000000,
		ActivationEpoch:  0,
		ExitEpoch:        10000000000,
		PublicKey:        phase0.BLSPubKey{},
	}
	copy(validator1.PublicKey[:], validator2Pubkey)

	validatorArray := make([]*phase0.Validator, 0)
	validatorArray = append(validatorArray, &validator1, &validator2)
	syncCommittee := altair.SyncCommittee{
		Pubkeys: make([]phase0.BLSPubKey, 0),
	}
	syncCommittee.Pubkeys = append(syncCommittee.Pubkeys, validator1.PublicKey) // validator 1 is in sync committee

	// state creation
	stateNext := fork_state.ForkStateContentBase{
		Validators:    validatorArray,
		Balances:      balancesArray,
		BlockRoots:    make([][]byte, 0),
		Epoch:         11,
		SyncCommittee: syncCommittee,
	}
	stateNext.Setup()

	// create fork metrics
	rewardsObj := NewAltairMetrics(
		stateNext,
		fork_state.ForkStateContentBase{},
		fork_state.ForkStateContentBase{})

	baseRewardPerInc := uint64(fork_state.EFFECTIVE_BALANCE_INCREMENT * fork_state.BASE_REWARD_FACTOR)
	baseRewardPerInc = baseRewardPerInc / uint64(math.Sqrt(float64(rewardsObj.NextState.TotalActiveBalance)))

	participantReward := uint64(rewardsObj.NextState.TotalActiveBalance / 1000000000)
	participantReward = participantReward * baseRewardPerInc
	participantReward = participantReward * 2 / 64 / 32
	participantReward = participantReward / 512
	require.Equal(t,
		rewardsObj.GetMaxSyncComReward(0),
		uint64(participantReward*32))

	require.Equal(t,
		rewardsObj.GetMaxSyncComReward(1),
		uint64(0)) // not in sync committee

	rewardsObj.NextState.MissedBlocks = append(rewardsObj.NextState.MissedBlocks, 1)

	require.Equal(t,
		rewardsObj.GetMaxSyncComReward(0),
		uint64(participantReward*31)) // one missed block

}
