package metrics

import (
	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

type AltairMetrics struct {
	baseMetrics StateMetricsBase
}

func NewAltairMetrics(
	firstState spec.AgnosticState,
	secondState spec.AgnosticState,
	thirdState spec.AgnosticState,
	fourthState spec.AgnosticState) AltairMetrics {

	altairObj := AltairMetrics{}
	altairObj.baseMetrics.FirstState = firstState
	altairObj.baseMetrics.SecondState = secondState
	altairObj.baseMetrics.ThirdState = thirdState
	altairObj.baseMetrics.FourthState = fourthState

	return altairObj
}

func (p AltairMetrics) GetMetricsBase() StateMetricsBase {
	return p.baseMetrics
}

// TODO: to be implemented once we can process each block
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#modified-process_attestation
func (p AltairMetrics) GetMaxProposerAttReward(valIdx phase0.ValidatorIndex) (phase0.Gwei, phase0.Slot) {

	proposerSlot := phase0.Slot(0)
	reward := phase0.Gwei(0)
	duties := p.baseMetrics.ThirdState.EpochStructs.ProposerDuties
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

	valPubKey := p.baseMetrics.ThirdState.Validators[valIdx].PublicKey

	syncCommitteePubKeys := p.baseMetrics.ThirdState.SyncCommittee

	for _, item := range syncCommitteePubKeys.Pubkeys {
		if valPubKey == item {
			inCommittee = true
		}
	}

	if !inCommittee {
		return 0
	}

	// at this point we know the validator was inside the sync committee and, therefore, active at that point

	totalActiveInc := p.baseMetrics.ThirdState.TotalActiveBalance / spec.EffectiveBalanceInc
	totalBaseRewards := p.GetBaseRewardPerInc(p.baseMetrics.ThirdState.TotalActiveBalance) * totalActiveInc
	maxParticipantRewards := totalBaseRewards * phase0.Gwei(spec.SyncRewardWeight) / phase0.Gwei(spec.WeightDenominator) / spec.SlotsPerEpoch
	participantReward := maxParticipantRewards / phase0.Gwei(spec.SyncCommitteeSize) // this is the participantReward for a single slot

	return participantReward * phase0.Gwei(spec.SlotsPerEpoch-len(p.baseMetrics.ThirdState.GetMissingBlocks())) // max reward would be 32 perfect slots

}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#get_flag_index_deltas
func (p AltairMetrics) GetMaxAttestationReward(valIdx phase0.ValidatorIndex) phase0.Gwei {

	maxFlagsReward := phase0.Gwei(0)
	// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

	if spec.IsActive(*p.baseMetrics.ThirdState.Validators[valIdx], phase0.Epoch(p.baseMetrics.FirstState.Epoch)) {
		baseReward := p.GetBaseReward(valIdx, p.baseMetrics.SecondState.Validators[valIdx].EffectiveBalance, p.baseMetrics.SecondState.TotalActiveBalance)
		// only consider flag Index rewards if the validator was active in the previous epoch

		for i := range p.baseMetrics.SecondState.AttestingBalance {

			// apply formula
			attestingBalanceInc := p.baseMetrics.SecondState.AttestingBalance[i] / spec.EffectiveBalanceInc

			flagReward := phase0.Gwei(spec.ParticipatingFlagsWeight[i]) * baseReward * attestingBalanceInc
			flagReward = flagReward / ((phase0.Gwei(p.baseMetrics.SecondState.TotalActiveBalance / spec.EffectiveBalanceInc)) * phase0.Gwei(spec.WeightDenominator))
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

func (p AltairMetrics) GetMaxReward(valIdx phase0.ValidatorIndex) (spec.ValidatorRewards, error) {

	baseReward := p.GetBaseReward(valIdx, p.baseMetrics.ThirdState.Validators[valIdx].EffectiveBalance, p.baseMetrics.ThirdState.TotalActiveBalance)

	flagIndexMaxReward := p.GetMaxAttestationReward(valIdx)

	syncComMaxReward := p.GetMaxSyncComReward(valIdx)

	inSyncCommitte := false
	if syncComMaxReward > 0 {
		inSyncCommitte = true
	}

	_, proposerSlot := p.GetMaxProposerAttReward(
		valIdx)

	maxReward := flagIndexMaxReward + syncComMaxReward

	flags := p.baseMetrics.SecondState.MissingFlags(valIdx)

	result := spec.ValidatorRewards{
		ValidatorIndex:      valIdx,
		Epoch:               p.baseMetrics.ThirdState.Epoch,
		ValidatorBalance:    p.baseMetrics.ThirdState.Balances[valIdx],
		Reward:              p.baseMetrics.EpochReward(valIdx),
		MaxReward:           maxReward,
		AttestationReward:   flagIndexMaxReward,
		SyncCommitteeReward: syncComMaxReward,
		AttSlot:             p.baseMetrics.FirstState.EpochStructs.ValidatorAttSlot[valIdx],
		MissingSource:       flags[0],
		MissingTarget:       flags[1],
		MissingHead:         flags[2],
		Status:              p.baseMetrics.ThirdState.GetValStatus(valIdx),
		BaseReward:          baseReward,
		ProposerSlot:        proposerSlot,
		InSyncCommittee:     inSyncCommitte,
	}
	return result, nil

}

func (p AltairMetrics) GetBaseReward(valIdx phase0.ValidatorIndex, effectiveBalance phase0.Gwei, totalEffectiveBalance phase0.Gwei) phase0.Gwei {
	effectiveBalanceInc := effectiveBalance / spec.EffectiveBalanceInc
	return p.GetBaseRewardPerInc(totalEffectiveBalance) * effectiveBalanceInc
}

func (p AltairMetrics) GetBaseRewardPerInc(totalEffectiveBalance phase0.Gwei) phase0.Gwei {

	var baseReward phase0.Gwei

	sqrt := uint64(math.Sqrt(float64(totalEffectiveBalance)))

	num := spec.EffectiveBalanceInc * spec.BaseRewardFactor
	baseReward = phase0.Gwei(uint64(num) / sqrt)

	return baseReward
}
