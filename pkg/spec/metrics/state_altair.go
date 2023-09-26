package metrics

import (
	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

type AltairMetrics struct {
	baseMetrics StateMetricsBase
}

func NewAltairMetrics(nextBstate spec.AgnosticState, bstate spec.AgnosticState, prevBstate spec.AgnosticState) AltairMetrics {

	altairObj := AltairMetrics{}
	altairObj.baseMetrics.CurrentState = bstate
	altairObj.baseMetrics.PrevState = prevBstate
	altairObj.baseMetrics.NextState = nextBstate

	return altairObj
}

func (p AltairMetrics) GetMetricsBase() StateMetricsBase {
	return p.baseMetrics
}

// TODO: to be implemented once we can process each block
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#modified-process_attestation
func (p AltairMetrics) GetProposerApiReward(valIdx phase0.ValidatorIndex) phase0.Gwei {

	reward := phase0.Gwei(0)

	duties := p.baseMetrics.NextState.EpochStructs.ProposerDuties
	// validator will only have duties it is active at this point
	for _, duty := range duties {
		if duty.ValidatorIndex == phase0.ValidatorIndex(valIdx) {
			reward += phase0.Gwei(p.baseMetrics.NextState.Blocks[duty.Slot%spec.SlotsPerEpoch].Reward.Data.Total)
			break
		}
	}
	return phase0.Gwei(reward)

}

// So far we have computed the max sync committee proposer reward for a slot. Since the validator remains in the sync committee for the full epoch, we multiply the reward for the 32 slots in the epoch.
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#sync-aggregate-processing
func (p AltairMetrics) GetMaxSyncComReward(valIdx phase0.ValidatorIndex) phase0.Gwei {

	valPubKey := p.baseMetrics.NextState.Validators[valIdx].PublicKey

	syncCommitteePubKeys := p.baseMetrics.NextState.SyncCommittee
	reward := phase0.Gwei(0)

	for _, item := range syncCommitteePubKeys.Pubkeys {
		if valPubKey == item { // hit, one validator can be multiple times in the same committee
			// at this point we know the validator was inside the sync committee and, therefore, active at that point

			totalActiveInc := p.baseMetrics.NextState.TotalActiveBalance / spec.EffectiveBalanceInc
			totalBaseRewards := p.GetBaseRewardPerInc(p.baseMetrics.NextState.TotalActiveBalance) * totalActiveInc
			maxParticipantRewards := totalBaseRewards * phase0.Gwei(spec.SyncRewardWeight) / phase0.Gwei(spec.WeightDenominator) / spec.SlotsPerEpoch
			participantReward := maxParticipantRewards / phase0.Gwei(spec.SyncCommitteeSize) // this is the participantReward for a single slot

			reward += participantReward * phase0.Gwei(spec.SlotsPerEpoch-len(p.baseMetrics.NextState.MissedBlocks)) // max reward would be 32 perfect slots
		}
	}

	return reward

}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#get_flag_index_deltas
func (p AltairMetrics) GetMaxAttestationReward(valIdx phase0.ValidatorIndex) phase0.Gwei {

	maxFlagsReward := phase0.Gwei(0)
	// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

	if spec.IsActive(*p.baseMetrics.NextState.Validators[valIdx], phase0.Epoch(p.baseMetrics.PrevState.Epoch)) {
		baseReward := p.GetBaseReward(valIdx, p.baseMetrics.CurrentState.Validators[valIdx].EffectiveBalance, p.baseMetrics.CurrentState.TotalActiveBalance)
		// only consider flag Index rewards if the validator was active in the previous epoch

		for i := range p.baseMetrics.CurrentState.AttestingBalance {

			// apply formula
			attestingBalanceInc := p.baseMetrics.CurrentState.AttestingBalance[i] / spec.EffectiveBalanceInc

			flagReward := phase0.Gwei(spec.ParticipatingFlagsWeight[i]) * baseReward * attestingBalanceInc
			flagReward = flagReward / ((phase0.Gwei(p.baseMetrics.CurrentState.TotalActiveBalance / spec.EffectiveBalanceInc)) * phase0.Gwei(spec.WeightDenominator))
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

	baseReward := p.GetBaseReward(valIdx, p.baseMetrics.NextState.Validators[valIdx].EffectiveBalance, p.baseMetrics.NextState.TotalActiveBalance)

	flagIndexMaxReward := p.GetMaxAttestationReward(valIdx)

	syncComMaxReward := p.GetMaxSyncComReward(valIdx)

	inSyncCommitte := false
	if syncComMaxReward > 0 {
		inSyncCommitte = true
	}

	proposerReward := p.GetProposerApiReward(valIdx)

	maxReward := flagIndexMaxReward + syncComMaxReward + proposerReward

	flags := p.baseMetrics.CurrentState.MissingFlags(valIdx)

	result := spec.ValidatorRewards{
		ValidatorIndex:      valIdx,
		Epoch:               p.baseMetrics.NextState.Epoch,
		ValidatorBalance:    p.baseMetrics.NextState.Balances[valIdx],
		Reward:              p.baseMetrics.EpochReward(valIdx) + int64(p.baseMetrics.NextState.Withdrawals[valIdx]),
		MaxReward:           maxReward,
		AttestationReward:   flagIndexMaxReward,
		SyncCommitteeReward: syncComMaxReward,
		AttSlot:             p.baseMetrics.PrevState.EpochStructs.ValidatorAttSlot[valIdx],
		MissingSource:       flags[0],
		MissingTarget:       flags[1],
		MissingHead:         flags[2],
		Status:              p.baseMetrics.NextState.GetValStatus(valIdx),
		BaseReward:          baseReward,
		ProposerReward:      int64(proposerReward),
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
