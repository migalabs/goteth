package fork_metrics

import (
	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth2-state-analyzer/pkg/fork_metrics/fork_state"
)

var ( // spec weight constants
	TIMELY_SOURCE_WEIGHT       = 14
	TIMELY_TARGET_WEIGHT       = 26
	TIMELY_HEAD_WEIGHT         = 14
	PARTICIPATING_FLAGS_WEIGHT = []int{TIMELY_SOURCE_WEIGHT, TIMELY_TARGET_WEIGHT, TIMELY_HEAD_WEIGHT}
	SYNC_REWARD_WEIGHT         = 2
	PROPOSER_WEIGHT            = 8
	WEIGHT_DENOMINATOR         = 64
	SYNC_COMMITTEE_SIZE        = 512
)

type AltairMetrics struct {
	StateMetricsBase
}

func NewAltairSpec(nextBstate fork_state.ForkStateContentBase, bstate fork_state.ForkStateContentBase, prevBstate fork_state.ForkStateContentBase) AltairMetrics {

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
func (p AltairMetrics) GetMaxProposerAttReward(valIdx uint64) (float64, int64) {

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

	return float64(reward), int64(proposerSlot)

}

// TODO: to be implemented once we can process each block
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#sync-aggregate-processing
func (p AltairMetrics) GetMaxProposerSyncReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	return 0

}

// So far we have computed the max sync committee proposer reward for a slot. Since the validator remains in the sync committee for the full epoch, we multiply the reward for the 32 slots in the epoch.
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#sync-aggregate-processing
func (p AltairMetrics) GetMaxSyncComReward(valIdx uint64) float64 {

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
	totalBaseRewards := p.GetBaseRewardPerInc() * float64(totalActiveInc)
	maxParticipantRewards := totalBaseRewards * float64(SYNC_REWARD_WEIGHT) / float64(WEIGHT_DENOMINATOR) / fork_state.SLOTS_PER_EPOCH
	participantReward := maxParticipantRewards / float64(SYNC_COMMITTEE_SIZE) // this is the participantReward for a single slot

	return participantReward * fork_state.SLOTS_PER_EPOCH

}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#get_flag_index_deltas
func (p AltairMetrics) GetMaxAttestationReward(valIdx uint64, baseReward float64) float64 {

	maxFlagsReward := float64(0)
	// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

	if fork_state.IsActive(*p.NextState.Validators[valIdx], phase0.Epoch(p.PrevState.Epoch)) {
		// only consider flag Index rewards if the validator was active in the previous epoch

		for i := range p.CurrentState.AttestingBalance {

			// apply formula
			attestingBalanceInc := p.CurrentState.AttestingBalance[i] / fork_state.EFFECTIVE_BALANCE_INCREMENT

			flagReward := float64(PARTICIPATING_FLAGS_WEIGHT[i]) * baseReward * float64(attestingBalanceInc)
			flagReward = flagReward / ((float64(p.CurrentState.TotalActiveBalance / fork_state.EFFECTIVE_BALANCE_INCREMENT)) * float64(WEIGHT_DENOMINATOR))
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

	baseReward := p.GetBaseReward(valIdx)

	flagIndexMaxReward := p.GetMaxAttestationReward(valIdx, baseReward)

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
		SyncCommittee:   0,
		MaxReward:       maxReward,
		BaseReward:      baseReward,
		ProposerSlot:    proposerSlot,
		InSyncCommittee: inSyncCommitte,
	}
	return result, nil

}

func (p AltairMetrics) GetBaseReward(valIdx uint64) float64 {
	effectiveBalanceInc := p.CurrentState.Validators[valIdx].EffectiveBalance / fork_state.EFFECTIVE_BALANCE_INCREMENT
	return p.GetBaseRewardPerInc() * float64(effectiveBalanceInc)
}

func (p AltairMetrics) GetBaseRewardPerInc() float64 {

	var baseReward float64

	sqrt := uint64(math.Sqrt(float64(p.CurrentState.TotalActiveBalance)))

	num := fork_state.EFFECTIVE_BALANCE_INCREMENT * fork_state.BASE_REWARD_FACTOR
	baseReward = num / float64(sqrt)

	return baseReward
}
