package metrics

import (
	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/migalabs/goteth/pkg/spec"
)

type AltairMetrics struct {
	Phase0Metrics
	MaxSyncCommitteeRewards map[phase0.ValidatorIndex]phase0.Gwei // rewards from participating in the sync committee
}

func NewAltairMetrics(
	nextState *spec.AgnosticState,
	currentState *spec.AgnosticState,
	prevState *spec.AgnosticState) AltairMetrics {

	altairObj := AltairMetrics{}

	altairObj.InitBundle(nextState, currentState, prevState)
	altairObj.PreProcessBundle()

	return altairObj

}

func (p *AltairMetrics) InitBundle(nextState *spec.AgnosticState,
	currentState *spec.AgnosticState,
	prevState *spec.AgnosticState) {
	p.baseMetrics.NextState = nextState
	p.baseMetrics.CurrentState = currentState
	p.baseMetrics.PrevState = prevState
	p.baseMetrics.MaxBlockRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	p.baseMetrics.MaxSlashingRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	p.baseMetrics.InclusionDelays = make([]int, len(p.baseMetrics.NextState.Validators))
	p.baseMetrics.MaxAttesterRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	p.MaxSyncCommitteeRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
}

func (p *AltairMetrics) PreProcessBundle() {

	if !p.baseMetrics.PrevState.EmptyStateRoot() {
		// block rewards
		p.ProcessAttestations()
		if !p.baseMetrics.CurrentState.EmptyStateRoot() {
			p.ProcessSlashings()
			p.ProcessSyncAggregates()

			p.GetMaxFlagIndexDeltas()
			p.ProcessInclusionDelays()
			p.GetMaxSyncComReward()
		}
	}
}

func (p AltairMetrics) GetMetricsBase() StateMetricsBase {
	return p.baseMetrics
}

func (p AltairMetrics) ProcessSyncAggregates() {
	nextState := p.baseMetrics.NextState
	for _, block := range nextState.Blocks {

		totalActiveInc := nextState.TotalActiveBalance / spec.EffectiveBalanceInc
		totalBaseRewards := p.GetBaseRewardPerInc(nextState.TotalActiveBalance) * totalActiveInc
		maxParticipantRewards := totalBaseRewards * phase0.Gwei(spec.SyncRewardWeight) / phase0.Gwei(spec.WeightDenominator) / spec.SlotsPerEpoch
		participantReward := maxParticipantRewards / phase0.Gwei(spec.SyncCommitteeSize) // this is the participantReward for a single slot
		singleProposerSyncReward := phase0.Gwei(participantReward * spec.ProposerWeight / (spec.WeightDenominator - spec.ProposerWeight))
		proposerSyncReward := singleProposerSyncReward * phase0.Gwei(block.SyncAggregate.SyncCommitteeBits.Count())

		p.baseMetrics.MaxBlockRewards[block.ProposerIndex] += proposerSyncReward
		block.ManualReward += proposerSyncReward
	}
}

func (p *AltairMetrics) ProcessInclusionDelays() {
	prevState := p.baseMetrics.PrevState
	currentState := p.baseMetrics.CurrentState
	for _, block := range append(prevState.Blocks, currentState.Blocks...) {
		// we assume the blocks are in order asc
		for _, attestation := range block.Attestations {
			attSlot := attestation.Data.Slot
			// Calculate inclusion delays only for attestations corresponding to slots from the previous epoch
			attSlotNotInPrevEpoch := attSlot < phase0.Slot(prevState.Epoch)*spec.SlotsPerEpoch || attSlot >= phase0.Slot(currentState.Epoch)*spec.SlotsPerEpoch
			if attSlotNotInPrevEpoch {
				continue
			}
			inclusionDelay := p.GetInclusionDelay(*attestation, *block)
			committeIndex := attestation.Data.Index

			attestingIndices := attestation.AggregationBits.BitIndices()

			for _, idx := range attestingIndices {
				valIdx, err := p.GetValidatorFromCommitteeIndex(attSlot, committeIndex, idx)
				if err != nil {
					log.Fatalf("error processing attestations at block %d: %s", block.Slot, err)
				}

				if p.baseMetrics.InclusionDelays[valIdx] == 0 {
					p.baseMetrics.InclusionDelays[valIdx] = inclusionDelay
				}
			}
		}
	}

	for valIdx, inclusionDelay := range p.baseMetrics.InclusionDelays {
		if inclusionDelay == 0 {
			p.baseMetrics.InclusionDelays[valIdx] = p.maxInclusionDelay(phase0.ValidatorIndex(valIdx)) + 1
		}
	}
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#modified-process_attestation
func (p AltairMetrics) ProcessAttestations() {
	nextState := p.baseMetrics.NextState
	currentState := p.baseMetrics.CurrentState

	if currentState.Blocks == nil { // only process attestations when CurrentState available
		return
	}

	currentEpochParticipation := make([][]bool, len(currentState.Validators))
	nextEpochParticipation := make([][]bool, len(nextState.Validators))

	blockList := currentState.Blocks
	blockList = append(
		blockList,
		nextState.Blocks...)

	for _, block := range blockList {

		for _, attestation := range block.Attestations {

			attReward := phase0.Gwei(0)
			slot := attestation.Data.Slot
			epochParticipation := nextEpochParticipation
			if slotInEpoch(slot, currentState.Epoch) {
				epochParticipation = currentEpochParticipation
			}

			if slot < phase0.Slot(currentState.Epoch)*spec.SlotsPerEpoch {
				continue
			}

			participationFlags := p.getParticipationFlags(*attestation, *block)

			committeIndex := attestation.Data.Index

			attestingIndices := attestation.AggregationBits.BitIndices()

			for _, idx := range attestingIndices {
				block.VotesIncluded += 1

				valIdx, err := p.GetValidatorFromCommitteeIndex(slot, committeIndex, idx)
				if err != nil {
					log.Fatalf("error processing attestations at block %d: %s", block.Slot, err)
				}
				if epochParticipation[valIdx] == nil {
					epochParticipation[valIdx] = make([]bool, len(spec.ParticipatingFlagsWeight))
				}

				if slotInEpoch(slot, currentState.Epoch) {
					currentState.ValidatorAttestationIncluded[valIdx] = true
				}

				// we are only counting rewards at NextState
				attesterBaseReward := p.GetBaseReward(valIdx, nextState.Validators[valIdx].EffectiveBalance, nextState.TotalActiveBalance)

				new := false
				if participationFlags[spec.AttSourceFlagIndex] && !epochParticipation[valIdx][spec.AttSourceFlagIndex] { // source
					attReward += attesterBaseReward * spec.TimelySourceWeight
					epochParticipation[valIdx][spec.AttSourceFlagIndex] = true
					new = true
				}
				if participationFlags[spec.AttTargetFlagIndex] && !epochParticipation[valIdx][spec.AttTargetFlagIndex] { // target
					attReward += attesterBaseReward * spec.TimelyTargetWeight
					epochParticipation[valIdx][spec.AttTargetFlagIndex] = true
					new = true
				}
				if participationFlags[spec.AttHeadFlagIndex] && !epochParticipation[valIdx][spec.AttHeadFlagIndex] { // head
					attReward += attesterBaseReward * spec.TimelyHeadWeight
					epochParticipation[valIdx][spec.AttHeadFlagIndex] = true
					new = true
				}
				if new {
					block.NewVotesIncluded += 1
				}
			}

			// only process rewards for blocks in NextState
			if block.Slot >= phase0.Slot(nextState.Epoch)*spec.SlotsPerEpoch {
				denominator := phase0.Gwei((spec.WeightDenominator - spec.ProposerWeight) * spec.WeightDenominator / spec.ProposerWeight)
				attReward = attReward / denominator

				p.baseMetrics.MaxBlockRewards[block.ProposerIndex] += attReward
				block.ManualReward += attReward
			}

		}

	}
}

// So far we have computed the max sync committee proposer reward for a slot. Since the validator remains in the sync committee for the full epoch, we multiply the reward for the 32 slots in the epoch.
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#sync-aggregate-processing
func (p AltairMetrics) GetMaxSyncComReward() {

	nextState := p.baseMetrics.NextState
	for _, valPubkey := range nextState.SyncCommittee.Pubkeys {

		for valIdx, validator := range nextState.Validators {

			if valPubkey == validator.PublicKey { // hit, one validator can be multiple times in the same committee
				// at this point we know the validator was inside the sync committee and, therefore, active at that point

				reward := phase0.Gwei(0)
				totalActiveInc := nextState.TotalActiveBalance / spec.EffectiveBalanceInc
				totalBaseRewards := p.GetBaseRewardPerInc(nextState.TotalActiveBalance) * totalActiveInc
				maxParticipantRewards := totalBaseRewards * phase0.Gwei(spec.SyncRewardWeight) / phase0.Gwei(spec.WeightDenominator) / spec.SlotsPerEpoch
				participantReward := maxParticipantRewards / phase0.Gwei(spec.SyncCommitteeSize) // this is the participantReward for a single slot

				reward += participantReward * phase0.Gwei(spec.SlotsPerEpoch-len(nextState.MissedBlocks)) // max reward would be 32 perfect slots
				p.MaxSyncCommitteeRewards[phase0.ValidatorIndex(valIdx)] += reward
			}
		}

	}

}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#get_flag_index_deltas
func (p AltairMetrics) GetMaxFlagIndexDeltas() {
	nextState := p.baseMetrics.NextState
	currentState := p.baseMetrics.CurrentState
	prevState := p.baseMetrics.PrevState
	for valIdx, validator := range nextState.Validators {
		maxFlagsReward := phase0.Gwei(0)
		// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

		if spec.IsActive(*validator, phase0.Epoch(prevState.Epoch)) {
			baseReward := p.GetBaseReward(phase0.ValidatorIndex(valIdx), currentState.Validators[valIdx].EffectiveBalance, currentState.TotalActiveBalance)
			// only consider flag Index rewards if the validator was active in the previous epoch

			for i := range currentState.AttestingBalance {

				if !p.isFlagPossible(phase0.ValidatorIndex(valIdx), i) { // consider if the attester could have achieved the flag (inclusion delay wise)
					continue
				}
				// apply formula
				attestingBalanceInc := currentState.AttestingBalance[i] / spec.EffectiveBalanceInc

				flagReward := phase0.Gwei(spec.ParticipatingFlagsWeight[i]) * baseReward * attestingBalanceInc
				flagReward = flagReward / ((phase0.Gwei(currentState.TotalActiveBalance / spec.EffectiveBalanceInc)) * phase0.Gwei(spec.WeightDenominator))
				maxFlagsReward += flagReward
			}
		}

		p.baseMetrics.MaxAttesterRewards[phase0.ValidatorIndex(valIdx)] += maxFlagsReward
	}
}

// This method returns the Max Reward the validator could gain
// Keep in mind we are calculating rewards at the last slot of the current epoch
// The max reward we calculate now, will be seen in the next epoch, but we will do this at the last slot of it.
// Therefore we consider:
// Attestations from last epoch (we see them in this epoch), balance change will take effect in the first slot of next epoch
// Sync Committee attestations from next epoch: balance change is added on the fly
// Proposer Rewards from next epoch: balance change is added on the fly

func (p AltairMetrics) GetMaxReward(valIdx phase0.ValidatorIndex) (spec.ValidatorRewards, error) {
	nextState := p.baseMetrics.NextState
	prevState := p.baseMetrics.PrevState
	currentState := p.baseMetrics.CurrentState
	flagIndexMaxReward := p.baseMetrics.MaxAttesterRewards[valIdx]
	syncComMaxReward := p.MaxSyncCommitteeRewards[valIdx]
	inSyncCommitte := syncComMaxReward > 0

	proposerReward := phase0.Gwei(0)
	proposerApiReward := phase0.Gwei(0)
	proposerManualReward := phase0.Gwei(0)

	for _, block := range nextState.Blocks {
		if block.Proposed && block.ProposerIndex == valIdx {
			proposerApiReward += phase0.Gwei(block.Reward.Data.Total)
			proposerManualReward += phase0.Gwei(block.ManualReward)
		}
	}
	proposerReward = proposerManualReward
	if proposerApiReward > 0 {
		proposerReward = proposerApiReward // if API rewards, always prioritize api
	}

	maxReward := flagIndexMaxReward + syncComMaxReward + proposerReward
	flags := currentState.MissingFlags(valIdx)
	baseReward := p.GetBaseReward(valIdx, nextState.Validators[valIdx].EffectiveBalance, nextState.TotalActiveBalance)

	attestationIncluded := false
	if int(valIdx) < len(p.baseMetrics.PrevState.ValidatorAttestationIncluded) {
		attestationIncluded = p.baseMetrics.PrevState.ValidatorAttestationIncluded[valIdx]
	}

	result := spec.ValidatorRewards{
		ValidatorIndex:       valIdx,
		Epoch:                nextState.Epoch,
		ValidatorBalance:     nextState.Balances[valIdx],
		WithdrawalPrefix:     nextState.Validators[valIdx].WithdrawalCredentials[0],
		Reward:               p.baseMetrics.EpochReward(valIdx),
		MaxReward:            maxReward,
		AttestationReward:    flagIndexMaxReward,
		SyncCommitteeReward:  syncComMaxReward,
		AttSlot:              prevState.EpochStructs.ValidatorAttSlot[valIdx],
		AttestationIncluded:  attestationIncluded,
		MissingSource:        flags[spec.AttSourceFlagIndex],
		MissingTarget:        flags[spec.AttTargetFlagIndex],
		MissingHead:          flags[spec.AttHeadFlagIndex],
		Status:               currentState.GetValStatus(valIdx),
		BaseReward:           baseReward,
		ProposerApiReward:    proposerApiReward,
		ProposerManualReward: proposerManualReward,
		InSyncCommittee:      inSyncCommitte,
		InclusionDelay:       p.baseMetrics.InclusionDelays[valIdx],
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

func (p AltairMetrics) GetInclusionDelay(attestation phase0.Attestation, includedInBlock spec.AgnosticBlock) int {
	return int(includedInBlock.Slot - attestation.Data.Slot)
}

func (p AltairMetrics) getParticipationFlags(attestation phase0.Attestation, includedInBlock spec.AgnosticBlock) [3]bool {
	nextState := p.baseMetrics.NextState

	var result [3]bool

	justifiedCheckpoint, err := p.GetJustifiedRootfromSlot(attestation.Data.Slot)
	if err != nil {
		log.Fatalf("error getting justified checkpoint: %s", err)
	}

	inclusionDelay := p.GetInclusionDelay(attestation, includedInBlock)

	targetRoot := nextState.GetBlockRoot(attestation.Data.Target.Epoch)
	headRoot := nextState.GetBlockRootAtSlot(attestation.Data.Slot)

	matchingSource := attestation.Data.Source.Root == justifiedCheckpoint
	matchingTarget := matchingSource && targetRoot == attestation.Data.Target.Root
	matchingHead := matchingTarget && attestation.Data.BeaconBlockRoot == headRoot

	if matchingSource && (inclusionDelay <= int(math.Sqrt(spec.SlotsPerEpoch))) {
		result[spec.AttSourceFlagIndex] = true
	}
	if matchingTarget && (inclusionDelay <= spec.SlotsPerEpoch) {
		result[spec.AttTargetFlagIndex] = true
	}
	if matchingHead && (inclusionDelay <= spec.MinInclusionDelay) {
		result[spec.AttHeadFlagIndex] = true
	}

	return result
}

func (p AltairMetrics) isFlagPossible(valIdx phase0.ValidatorIndex, flagIndex int) bool {
	prevState := p.baseMetrics.PrevState
	currentState := p.baseMetrics.CurrentState
	attSlot := prevState.EpochStructs.ValidatorAttSlot[valIdx]
	maxInclusionDelay := 0

	switch flagIndex { // for every flag there is a max inclusion delay to obtain a reward
	case spec.AttSourceFlagIndex: // 5
		maxInclusionDelay = int(math.Sqrt(spec.SlotsPerEpoch))
	case spec.AttTargetFlagIndex: // 32
		maxInclusionDelay = spec.SlotsPerEpoch
	case spec.AttHeadFlagIndex: // 1
		maxInclusionDelay = spec.MinInclusionDelay
	default:
		log.Fatalf("provided flag index %d is not known", flagIndex)
	}

	// look for any block proposed => the attester could have achieved it
	for slot := attSlot + 1; slot <= (attSlot + phase0.Slot(maxInclusionDelay)); slot++ {
		slotInEpoch := slot % spec.SlotsPerEpoch
		block := prevState.Blocks[slotInEpoch]
		if slot >= phase0.Slot(currentState.Epoch*spec.SlotsPerEpoch) {
			block = currentState.Blocks[slotInEpoch]
		}

		if block.Proposed { // if there was a block proposed inside the inclusion window
			return true
		}
	}
	return false

}

func (p AltairMetrics) maxInclusionDelay(_ phase0.ValidatorIndex) int {
	return spec.SlotsPerEpoch
}
