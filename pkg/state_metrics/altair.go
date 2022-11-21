package state_metrics

import (
	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth2-state-analyzer/pkg/state_metrics/fork_state"
)

type AltairMetrics struct {
	StateMetricsBase
}

func NewAltairMetrics(nextBstate fork_state.ForkStateContentBase, bstate fork_state.ForkStateContentBase, prevBstate fork_state.ForkStateContentBase) AltairMetrics {

	altairObj := AltairMetrics{}
	altairObj.CurrentState = bstate
	altairObj.PrevState = prevBstate
	altairObj.NextState = nextBstate

	return altairObj
}

func (p AltairMetrics) GetMetricsBase() StateMetricsBase {
	return p.StateMetricsBase
}

// TODO: to be implemented once we can process each block
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#modified-process_attestation
func (p AltairMetrics) GetMaxProposerAttReward(valIdx uint64) (uint64, int64) {

	proposerSlot := -1
	reward := 0
	duties := p.NextState.EpochStructs.ProposerDuties
	// validator will only have duties it is active at this point
	for _, duty := range duties {
		if duty.ValidatorIndex == phase0.ValidatorIndex(valIdx) {
			proposerSlot = int(duty.Slot)
			break
		}
	}

	return uint64(reward), int64(proposerSlot)

}

// TODO: to be implemented once we can process each block
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#sync-aggregate-processing
func (p AltairMetrics) GetMaxProposerSyncReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) int64 {

	return 0

}

// So far we have computed the max sync committee proposer reward for a slot. Since the validator remains in the sync committee for the full epoch, we multiply the reward for the 32 slots in the epoch.
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#sync-aggregate-processing
func (p AltairMetrics) GetMaxSyncComReward(valIdx uint64) uint64 {

	inCommittee := false

	valPubKey := p.NextState.Validators[valIdx].PublicKey

	syncCommitteePubKeys := p.NextState.SyncCommittee

	for _, item := range syncCommitteePubKeys.Pubkeys {
		if valPubKey == item {
			inCommittee = true
		}
	}

	if !inCommittee {
		return 0
	}

	// at this point we know the validator was inside the sync committee and, therefore, active at that point

	totalActiveInc := p.NextState.TotalActiveBalance / fork_state.EFFECTIVE_BALANCE_INCREMENT
	totalBaseRewards := p.GetBaseRewardPerInc(p.NextState.TotalActiveBalance) * uint64(totalActiveInc)
	maxParticipantRewards := totalBaseRewards * uint64(fork_state.SYNC_REWARD_WEIGHT) / uint64(fork_state.WEIGHT_DENOMINATOR) / fork_state.SLOTS_PER_EPOCH
	participantReward := maxParticipantRewards / uint64(fork_state.SYNC_COMMITTEE_SIZE) // this is the participantReward for a single slot

	return participantReward * uint64(fork_state.SLOTS_PER_EPOCH-len(p.NextState.MissedBlocks)) // max reward would be 32 perfect slots

}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#get_flag_index_deltas
func (p AltairMetrics) GetMaxAttestationReward(valIdx uint64) uint64 {

	maxFlagsReward := uint64(0)
	// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

	if fork_state.IsActive(*p.NextState.Validators[valIdx], phase0.Epoch(p.PrevState.Epoch)) {
		baseReward := p.GetBaseReward(valIdx, uint64(p.CurrentState.Validators[valIdx].EffectiveBalance), p.CurrentState.TotalActiveBalance)
		// only consider flag Index rewards if the validator was active in the previous epoch

		for i := range p.CurrentState.AttestingBalance {

			// apply formula
			attestingBalanceInc := p.CurrentState.AttestingBalance[i] / fork_state.EFFECTIVE_BALANCE_INCREMENT

			flagReward := uint64(fork_state.PARTICIPATING_FLAGS_WEIGHT[i]) * baseReward * uint64(attestingBalanceInc)
			flagReward = flagReward / ((uint64(p.CurrentState.TotalActiveBalance / fork_state.EFFECTIVE_BALANCE_INCREMENT)) * uint64(fork_state.WEIGHT_DENOMINATOR))
			maxFlagsReward += flagReward
		}
	}

	return maxFlagsReward
}

// This method returns the Max Reward the validator could gain
// Keep in mind we are calculating rewards at the last slot of the current epoch
// The max reward we calculate now, will be seen in the next epoch, but we will do this at the last slot of it.
// Therefore we consider:
// Attestations from last epoch (we see them in this epoch), balance change will take effect in the first slot of next epoch
// Sync Committee attestations from next epoch: balance change is added on the fly
// Proposer Rewards from next epoch: balance change is added on the fly

func (p AltairMetrics) GetMaxReward(valIdx uint64) (ValidatorSepRewards, error) {

	baseReward := p.GetBaseReward(valIdx, uint64(p.NextState.Validators[valIdx].EffectiveBalance), p.NextState.TotalActiveBalance)

	flagIndexMaxReward := p.GetMaxAttestationReward(valIdx)

	syncComMaxReward := p.GetMaxSyncComReward(valIdx)

	inSyncCommitte := false
	if syncComMaxReward > 0 {
		inSyncCommitte = true
	}

	_, proposerSlot := p.GetMaxProposerAttReward(
		valIdx)

	maxReward := flagIndexMaxReward + syncComMaxReward

	result := ValidatorSepRewards{
		Attestation:     0,
		InclusionDelay:  0,
		FlagIndex:       flagIndexMaxReward,
		SyncCommittee:   syncComMaxReward,
		MaxReward:       maxReward,
		BaseReward:      baseReward,
		ProposerSlot:    proposerSlot,
		InSyncCommittee: inSyncCommitte,
	}
	return result, nil

}

func (p AltairMetrics) GetBaseReward(valIdx uint64, effectiveBalance uint64, totalEffectiveBalance uint64) uint64 {
	effectiveBalanceInc := effectiveBalance / fork_state.EFFECTIVE_BALANCE_INCREMENT
	return p.GetBaseRewardPerInc(totalEffectiveBalance) * uint64(effectiveBalanceInc)
}

func (p AltairMetrics) GetBaseRewardPerInc(totalEffectiveBalance uint64) uint64 {

	var baseReward uint64

	sqrt := uint64(math.Sqrt(float64(totalEffectiveBalance)))

	num := fork_state.EFFECTIVE_BALANCE_INCREMENT * fork_state.BASE_REWARD_FACTOR
	baseReward = uint64(num) / uint64(sqrt)

	return baseReward
}
