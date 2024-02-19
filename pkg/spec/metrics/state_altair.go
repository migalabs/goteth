package metrics

import (
	"log"
	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/migalabs/goteth/pkg/spec"
	local_spec "github.com/migalabs/goteth/pkg/spec"
)

type AltairMetrics struct {
	baseMetrics StateMetricsBase
}

func NewAltairMetrics(
	nextBstate *spec.AgnosticState,
	bstate *spec.AgnosticState,
	prevBstate *spec.AgnosticState) AltairMetrics {

	altairObj := AltairMetrics{}
	altairObj.baseMetrics.CurrentState = bstate
	altairObj.baseMetrics.PrevState = prevBstate
	altairObj.baseMetrics.NextState = nextBstate
	altairObj.baseMetrics.BlockRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	altairObj.baseMetrics.SlashingRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)

	altairObj.ProcessAttestations()
	altairObj.ProcessSyncAggregates()
	altairObj.ProcessSlashings()

	return altairObj
}

func (p AltairMetrics) GetMetricsBase() StateMetricsBase {
	return p.baseMetrics
}

func (p *AltairMetrics) ProcessSlashings() {

	for _, block := range p.GetMetricsBase().NextState.Blocks {
		slashedIdxs := make([]phase0.ValidatorIndex, 0)
		whistleBlowerIdx := block.ProposerIndex // spec always contemplates whistleblower to be the block proposer
		whistleBlowerReward := phase0.Gwei(0)
		proposerReward := phase0.Gwei(0)
		for _, attSlashing := range block.AttesterSlashings {
			slashedIdxs = append(slashedIdxs, spec.SlashingIntersection(attSlashing.Attestation1.AttestingIndices, attSlashing.Attestation2.AttestingIndices)...)
		}
		for _, proposerSlashing := range block.ProposerSlashings {
			slashedIdxs = append(slashedIdxs, proposerSlashing.SignedHeader1.Message.ProposerIndex)
		}

		for _, idx := range slashedIdxs {
			slashedEffBalance := p.baseMetrics.NextState.Validators[idx].EffectiveBalance
			whistleBlowerReward += slashedEffBalance / spec.WhistleBlowerRewardQuotient
			proposerReward += whistleBlowerReward * spec.ProposerWeight / spec.WeightDenominator
		}
		p.baseMetrics.SlashingRewards[block.ProposerIndex] += proposerReward
		p.baseMetrics.SlashingRewards[whistleBlowerIdx] += whistleBlowerReward - proposerReward
	}
}

func (p AltairMetrics) ProcessSyncAggregates() {
	for _, block := range p.baseMetrics.NextState.Blocks {

		totalActiveInc := p.baseMetrics.NextState.TotalActiveBalance / spec.EffectiveBalanceInc
		totalBaseRewards := p.GetBaseRewardPerInc(p.baseMetrics.NextState.TotalActiveBalance) * totalActiveInc
		maxParticipantRewards := totalBaseRewards * phase0.Gwei(spec.SyncRewardWeight) / phase0.Gwei(spec.WeightDenominator) / spec.SlotsPerEpoch
		participantReward := maxParticipantRewards / phase0.Gwei(spec.SyncCommitteeSize) // this is the participantReward for a single slot
		singleProposerSyncReward := phase0.Gwei(participantReward * local_spec.ProposerWeight / (local_spec.WeightDenominator - local_spec.ProposerWeight))
		proposerSyncReward := singleProposerSyncReward * phase0.Gwei(block.SyncAggregate.SyncCommitteeBits.Count())

		if _, ok := p.baseMetrics.BlockRewards[block.ProposerIndex]; ok {
			p.baseMetrics.BlockRewards[block.ProposerIndex] += proposerSyncReward
		} else {
			p.baseMetrics.BlockRewards[block.ProposerIndex] = proposerSyncReward
		}
	}
}

// TODO: to be implemented once we can process each block
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#modified-process_attestation
func (p AltairMetrics) ProcessAttestations() {

	if p.baseMetrics.CurrentState.Blocks == nil { // only process attestations when CurrentState available
		return
	}

	currentEpochParticipation := make([][]bool, len(p.baseMetrics.CurrentState.Validators))
	nextEpochParticipation := make([][]bool, len(p.baseMetrics.NextState.Validators))

	blockList := p.baseMetrics.CurrentState.Blocks
	blockList = append(
		blockList,
		p.baseMetrics.NextState.Blocks...)

	for _, block := range blockList {

		newVotes := 0
		for _, attestation := range block.Attestations {

			attReward := phase0.Gwei(0)
			slot := attestation.Data.Slot
			epochParticipation := currentEpochParticipation
			if slot >= phase0.Slot(p.baseMetrics.NextState.Epoch)*spec.SlotsPerEpoch &&
				slot < phase0.Slot(p.baseMetrics.NextState.Epoch+1)*spec.SlotsPerEpoch {
				epochParticipation = nextEpochParticipation
			}

			if slot < phase0.Slot(p.baseMetrics.CurrentState.Epoch)*local_spec.SlotsPerEpoch {
				continue
			}

			participationFlags := p.GetParticipationFlags(*attestation, block)

			committeIndex := attestation.Data.Index

			attestingIndices := attestation.AggregationBits.BitIndices()

			for _, idx := range attestingIndices {
				valIdx, err := p.GetValidatorFromCommitteeIndex(slot, committeIndex, idx)
				if err != nil {
					log.Fatalf("error processing attestations at block %d: %s", block.Slot, err)
				}
				if epochParticipation[valIdx] == nil {
					epochParticipation[valIdx] = make([]bool, len(local_spec.ParticipatingFlagsWeight))
				}

				// we are only counting rewards at NextState
				attesterBaseReward := p.GetBaseReward(valIdx, p.baseMetrics.NextState.Validators[valIdx].EffectiveBalance, p.baseMetrics.NextState.TotalActiveBalance)

				new := false
				if participationFlags[0] && !epochParticipation[valIdx][0] { // source
					attReward += attesterBaseReward * spec.TimelySourceWeight
					epochParticipation[valIdx][0] = true
					new = true
				}
				if participationFlags[1] && !epochParticipation[valIdx][1] { // target
					attReward += attesterBaseReward * spec.TimelyTargetWeight
					epochParticipation[valIdx][1] = true
					new = true
				}
				if participationFlags[2] && !epochParticipation[valIdx][2] { // head
					attReward += attesterBaseReward * spec.TimelyHeadWeight
					epochParticipation[valIdx][2] = true
					new = true
				}
				if new {
					newVotes++
				}
			}

			// only process rewards for blocks in NextState
			if block.Slot >= phase0.Slot(p.baseMetrics.NextState.Epoch)*local_spec.SlotsPerEpoch {
				denominator := phase0.Gwei((spec.WeightDenominator - spec.ProposerWeight) * spec.WeightDenominator / spec.ProposerWeight)
				attReward = attReward / denominator
				if _, ok := p.baseMetrics.BlockRewards[block.ProposerIndex]; ok {
					p.baseMetrics.BlockRewards[block.ProposerIndex] += attReward
				} else {
					p.baseMetrics.BlockRewards[block.ProposerIndex] = attReward
				}
			}

		}

	}
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

	proposerReward := phase0.Gwei(0)
	proposerApiReward := phase0.Gwei(0)
	proposerManualReward := phase0.Gwei(0)

	for _, block := range p.baseMetrics.NextState.Blocks {
		if block.Proposed && block.ProposerIndex == valIdx {
			proposerApiReward += phase0.Gwei(block.Reward.Data.Total)
		}
	}

	if reward, ok := p.baseMetrics.BlockRewards[valIdx]; ok {
		proposerManualReward += reward
	}

	if reward, ok := p.baseMetrics.SlashingRewards[valIdx]; ok {
		proposerManualReward += reward
	}

	proposerReward = proposerManualReward
	if proposerApiReward > 0 {
		proposerReward = proposerApiReward // if API rewards, always prioritize api
	}

	maxReward := flagIndexMaxReward + syncComMaxReward + proposerReward

	flags := p.baseMetrics.CurrentState.MissingFlags(valIdx)

	result := spec.ValidatorRewards{
		ValidatorIndex:       valIdx,
		Epoch:                p.baseMetrics.NextState.Epoch,
		ValidatorBalance:     p.baseMetrics.NextState.Balances[valIdx],
		Reward:               p.baseMetrics.EpochReward(valIdx) + int64(p.baseMetrics.NextState.Withdrawals[valIdx]),
		MaxReward:            maxReward,
		AttestationReward:    flagIndexMaxReward,
		SyncCommitteeReward:  syncComMaxReward,
		AttSlot:              p.baseMetrics.PrevState.EpochStructs.ValidatorAttSlot[valIdx],
		MissingSource:        flags[0],
		MissingTarget:        flags[1],
		MissingHead:          flags[2],
		Status:               p.baseMetrics.NextState.GetValStatus(valIdx),
		BaseReward:           baseReward,
		ProposerApiReward:    int64(proposerApiReward),
		ProposerManualReward: int64(proposerManualReward),
		InSyncCommittee:      inSyncCommitte,
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

func (p AltairMetrics) GetParticipationFlags(attestation phase0.Attestation, includedInBlock spec.AgnosticBlock) [3]bool {
	var result [3]bool

	justifiedCheckpoint, err := p.GetJustifiedRootfromSlot(attestation.Data.Slot)
	if err != nil {
		log.Fatalf("error getting justified checkpoint: %s", err)
	}

	inclusionDelay := int(includedInBlock.Slot - attestation.Data.Slot)

	targetRoot := p.baseMetrics.NextState.GetBlockRoot(attestation.Data.Target.Epoch)
	headRoot := p.baseMetrics.NextState.GetBlockRootAtSlot(attestation.Data.Slot)

	matchingSource := attestation.Data.Source.Root == justifiedCheckpoint
	matchingTarget := matchingSource && targetRoot == attestation.Data.Target.Root
	matchingHead := matchingTarget && attestation.Data.BeaconBlockRoot == headRoot

	if matchingSource && (inclusionDelay <= int(math.Sqrt(local_spec.SlotsPerEpoch))) {
		result[0] = true
	}
	if matchingTarget && (inclusionDelay <= local_spec.SlotsPerEpoch) {
		result[1] = true
	}
	if matchingHead && (inclusionDelay <= local_spec.MinInclusionDelay) {
		result[2] = true
	}

	return result
}
