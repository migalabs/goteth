package state_metrics

import (
	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"
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
func (p AltairMetrics) GetMaxProposerAttReward(valIdx phase0.ValidatorIndex) (phase0.Gwei, phase0.Slot) {

	proposerSlot := phase0.Slot(0)
	reward := phase0.Gwei(0)
	duties := p.NextState.EpochStructs.ProposerDuties
	// validator will only have duties it is active at this point
	for _, duty := range duties {
		if duty.ValidatorIndex == phase0.ValidatorIndex(valIdx) {
			proposerSlot = duty.Slot
			break
		}
	}

	return reward, proposerSlot

}

// TODO: to be implemented once we can process each block
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#sync-aggregate-processing
func (p AltairMetrics) GetMaxProposerSyncReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) int64 {

	return 0

}

// So far we have computed the max sync committee proposer reward for a slot. Since the validator remains in the sync committee for the full epoch, we multiply the reward for the 32 slots in the epoch.
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#sync-aggregate-processing
func (p AltairMetrics) GetMaxSyncComReward(valIdx phase0.ValidatorIndex) phase0.Gwei {

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
	totalBaseRewards := p.GetBaseRewardPerInc(p.NextState.TotalActiveBalance) * totalActiveInc
	maxParticipantRewards := totalBaseRewards * phase0.Gwei(fork_state.SYNC_REWARD_WEIGHT) / phase0.Gwei(fork_state.WEIGHT_DENOMINATOR) / fork_state.SLOTS_PER_EPOCH
	participantReward := maxParticipantRewards / phase0.Gwei(fork_state.SYNC_COMMITTEE_SIZE) // this is the participantReward for a single slot

	return participantReward * phase0.Gwei(fork_state.SLOTS_PER_EPOCH-len(p.NextState.MissedBlocks)) // max reward would be 32 perfect slots

}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#get_flag_index_deltas
func (p AltairMetrics) GetMaxAttestationReward(valIdx phase0.ValidatorIndex) phase0.Gwei {

	maxFlagsReward := phase0.Gwei(0)
	// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

	if fork_state.IsActive(*p.NextState.Validators[valIdx], phase0.Epoch(p.PrevState.Epoch)) {
		baseReward := p.GetBaseReward(valIdx, p.CurrentState.Validators[valIdx].EffectiveBalance, p.CurrentState.TotalActiveBalance)
		// only consider flag Index rewards if the validator was active in the previous epoch

		for i := range p.CurrentState.AttestingBalance {

			// apply formula
			attestingBalanceInc := p.CurrentState.AttestingBalance[i] / fork_state.EFFECTIVE_BALANCE_INCREMENT

			flagReward := phase0.Gwei(fork_state.PARTICIPATING_FLAGS_WEIGHT[i]) * baseReward * attestingBalanceInc
			flagReward = flagReward / ((phase0.Gwei(p.CurrentState.TotalActiveBalance / fork_state.EFFECTIVE_BALANCE_INCREMENT)) * phase0.Gwei(fork_state.WEIGHT_DENOMINATOR))
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

func (p AltairMetrics) GetMaxReward(valIdx phase0.ValidatorIndex) (model.ValidatorRewards, error) {

	baseReward := p.GetBaseReward(valIdx, p.NextState.Validators[valIdx].EffectiveBalance, p.NextState.TotalActiveBalance)

	flagIndexMaxReward := p.GetMaxAttestationReward(valIdx)

	syncComMaxReward := p.GetMaxSyncComReward(valIdx)

	inSyncCommitte := false
	if syncComMaxReward > 0 {
		inSyncCommitte = true
	}

	_, proposerSlot := p.GetMaxProposerAttReward(
		valIdx)

	maxReward := flagIndexMaxReward + syncComMaxReward

	result := model.ValidatorRewards{
		ValidatorIndex:      valIdx,
		Epoch:               p.NextState.Epoch,
		ValidatorBalance:    p.CurrentState.Balances[valIdx],
		Reward:              p.EpochReward(valIdx),
		MaxReward:           maxReward,
		AttestationReward:   flagIndexMaxReward,
		SyncCommitteeReward: syncComMaxReward,
		AttSlot:             0,
		MissingSource:       false,
		MissingTarget:       false,
		MissingHead:         false,
		Status:              0,
		BaseReward:          baseReward,
		ProposerSlot:        proposerSlot,
		InSyncCommittee:     inSyncCommitte,
	}
	return result, nil

}

func (p AltairMetrics) GetBaseReward(valIdx phase0.ValidatorIndex, effectiveBalance phase0.Gwei, totalEffectiveBalance phase0.Gwei) phase0.Gwei {
	effectiveBalanceInc := effectiveBalance / fork_state.EFFECTIVE_BALANCE_INCREMENT
	return p.GetBaseRewardPerInc(totalEffectiveBalance) * effectiveBalanceInc
}

func (p AltairMetrics) GetBaseRewardPerInc(totalEffectiveBalance phase0.Gwei) phase0.Gwei {

	var baseReward phase0.Gwei

	sqrt := uint64(math.Sqrt(float64(totalEffectiveBalance)))

	num := fork_state.EFFECTIVE_BALANCE_INCREMENT * fork_state.BASE_REWARD_FACTOR
	baseReward = phase0.Gwei(uint64(num) / sqrt)

	return baseReward
}
