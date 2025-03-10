package metrics

import (
	"math"

	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

type ElectraMetrics struct {
	DenebMetrics
}

func NewElectraMetrics(
	nextState *spec.AgnosticState,
	currentState *spec.AgnosticState,
	prevState *spec.AgnosticState) ElectraMetrics {

	electraObj := ElectraMetrics{}

	electraObj.InitBundle(nextState, currentState, prevState)
	electraObj.PreProcessBundle()

	return electraObj
}

func (p *ElectraMetrics) InitBundle(nextState *spec.AgnosticState,
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

func (p *ElectraMetrics) PreProcessBundle() {

	if !p.baseMetrics.PrevState.EmptyStateRoot() && !p.baseMetrics.CurrentState.EmptyStateRoot() {
		// block rewards
		p.ProcessAttestations()
		p.ProcessSlashings()
		p.ProcessSyncAggregates()

		p.GetMaxFlagIndexDeltas()
		p.ProcessInclusionDelays()
		p.GetMaxSyncComReward()
	}
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/electra/beacon-chain.md#modified-process_attestation
func (p ElectraMetrics) ProcessAttestations() {

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

		for _, attestation := range block.ElectraAttestations {

			attReward := phase0.Gwei(0)
			slot := attestation.Data.Slot
			epochParticipation := nextEpochParticipation
			if slotInEpoch(slot, p.baseMetrics.CurrentState.Epoch) {
				epochParticipation = currentEpochParticipation
			}

			if slot < phase0.Slot(p.baseMetrics.CurrentState.Epoch)*spec.SlotsPerEpoch {
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

				if slotInEpoch(slot, p.baseMetrics.CurrentState.Epoch) {
					p.baseMetrics.CurrentState.ValidatorAttestationIncluded[valIdx] = true
				}

				// we are only counting rewards at NextState
				attesterBaseReward := p.GetBaseReward(valIdx, p.baseMetrics.NextState.Validators[valIdx].EffectiveBalance, p.baseMetrics.NextState.TotalActiveBalance)

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
			if block.Slot >= phase0.Slot(p.baseMetrics.NextState.Epoch)*spec.SlotsPerEpoch {
				denominator := phase0.Gwei((spec.WeightDenominator - spec.ProposerWeight) * spec.WeightDenominator / spec.ProposerWeight)
				attReward = attReward / denominator

				p.baseMetrics.MaxBlockRewards[block.ProposerIndex] += attReward
				block.ManualReward += attReward
			}

		}

	}
}

func (p *ElectraMetrics) ProcessInclusionDelays() {
	for _, block := range append(p.baseMetrics.PrevState.Blocks, p.baseMetrics.CurrentState.Blocks...) {
		// we assume the blocks are in order asc
		for _, attestation := range block.ElectraAttestations {
			attSlot := attestation.Data.Slot
			// Calculate inclusion delays only for attestations corresponding to slots from the previous epoch
			attSlotNotInPrevEpoch := attSlot < phase0.Slot(p.baseMetrics.PrevState.Epoch)*spec.SlotsPerEpoch || attSlot >= phase0.Slot(p.baseMetrics.CurrentState.Epoch)*spec.SlotsPerEpoch
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

// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#get_flag_index_deltas
func (p ElectraMetrics) GetMaxFlagIndexDeltas() {

	for valIdx, validator := range p.baseMetrics.NextState.Validators {
		maxFlagsReward := phase0.Gwei(0)
		// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

		if spec.IsActive(*validator, phase0.Epoch(p.baseMetrics.PrevState.Epoch)) {
			baseReward := p.GetBaseReward(phase0.ValidatorIndex(valIdx), p.baseMetrics.CurrentState.Validators[valIdx].EffectiveBalance, p.baseMetrics.CurrentState.TotalActiveBalance)
			// only consider flag Index rewards if the validator was active in the previous epoch

			for i := range p.baseMetrics.CurrentState.AttestingBalance {

				if !p.isFlagPossible(phase0.ValidatorIndex(valIdx), i) { // consider if the attester could have achieved the flag (inclusion delay wise)
					continue
				}
				// apply formula
				attestingBalanceInc := p.baseMetrics.CurrentState.AttestingBalance[i] / spec.EffectiveBalanceInc

				flagReward := phase0.Gwei(spec.ParticipatingFlagsWeight[i]) * baseReward * attestingBalanceInc
				flagReward = flagReward / ((phase0.Gwei(p.baseMetrics.CurrentState.TotalActiveBalance / spec.EffectiveBalanceInc)) * phase0.Gwei(spec.WeightDenominator))
				maxFlagsReward += flagReward
			}
		}

		p.baseMetrics.MaxAttesterRewards[phase0.ValidatorIndex(valIdx)] += maxFlagsReward
	}
}

func (p ElectraMetrics) GetInclusionDelay(attestation electra.Attestation, includedInBlock spec.AgnosticBlock) int {
	return int(includedInBlock.Slot - attestation.Data.Slot)
}

func (p ElectraMetrics) getParticipationFlags(attestation electra.Attestation, includedInBlock spec.AgnosticBlock) [3]bool {
	var result [3]bool

	justifiedCheckpoint, err := p.GetJustifiedRootfromSlot(attestation.Data.Slot)
	if err != nil {
		log.Fatalf("error getting justified checkpoint: %s", err)
	}

	inclusionDelay := p.GetInclusionDelay(attestation, includedInBlock)

	targetRoot := p.baseMetrics.NextState.GetBlockRoot(attestation.Data.Target.Epoch)
	headRoot := p.baseMetrics.NextState.GetBlockRootAtSlot(attestation.Data.Slot)

	matchingSource := attestation.Data.Source.Root == justifiedCheckpoint
	matchingTarget := matchingSource && targetRoot == attestation.Data.Target.Root
	matchingHead := matchingTarget && attestation.Data.BeaconBlockRoot == headRoot

	// the attestation must be included maximum in the next epoch
	// the worst case scenario is an attestation to the slot 31, which gives a max inclusion delay of 32
	// the best case scenario is an attestation to the slot 0, which gives a max inclusion delay of 64
	// https://github.com/ethereum/consensus-specs/blob/dev/specs/electra/beacon-chain.md#modified-get_attestation_participation_flag_indices
	includedInEpoch := phase0.Epoch(includedInBlock.Slot / spec.SlotsPerEpoch)
	attestationEpoch := phase0.Epoch(attestation.Data.Slot / spec.SlotsPerEpoch)
	targetInclusionOk := includedInEpoch-attestationEpoch <= 1

	if matchingSource && (inclusionDelay <= int(math.Sqrt(spec.SlotsPerEpoch))) {
		result[0] = true
	}
	if matchingTarget && targetInclusionOk {
		result[1] = true
	}
	if matchingHead && (inclusionDelay <= spec.MinInclusionDelay) {
		result[2] = true
	}

	return result
}

func (p ElectraMetrics) isFlagPossible(valIdx phase0.ValidatorIndex, flagIndex int) bool {
	attSlot := p.baseMetrics.PrevState.EpochStructs.ValidatorAttSlot[valIdx]
	maxInclusionDelay := 0

	switch flagIndex { // for every flag there is a max inclusion delay to obtain a reward

	case spec.AttSourceFlagIndex: // 5
		maxInclusionDelay = int(math.Sqrt(spec.SlotsPerEpoch))

	case spec.AttTargetFlagIndex: // until end of next epoch
		remainingSlotsInEpoch := spec.SlotsPerEpoch - int(attSlot%spec.SlotsPerEpoch)
		maxInclusionDelay = spec.SlotsPerEpoch + remainingSlotsInEpoch

	case spec.AttHeadFlagIndex: // 1
		maxInclusionDelay = 1
	default:
		log.Fatalf("provided flag index %d is not known", flagIndex)
	}

	// look for any block proposed => the attester could have achieved it
	for slot := attSlot + 1; slot <= (attSlot + phase0.Slot(maxInclusionDelay)); slot++ {
		slotInEpoch := slot % spec.SlotsPerEpoch
		block := p.baseMetrics.PrevState.Blocks[slotInEpoch]
		if slot >= phase0.Slot(p.baseMetrics.CurrentState.Epoch*spec.SlotsPerEpoch) {
			block = p.baseMetrics.CurrentState.Blocks[slotInEpoch]
		}

		if block.Proposed { // if there was a block proposed inside the inclusion window
			return true
		}
	}
	return false

}

func (p ElectraMetrics) maxInclusionDelay(valIdx phase0.ValidatorIndex) int {

	// check attestationSlot in prev epoch

	slot := p.baseMetrics.PrevState.EpochStructs.ValidatorAttSlot[valIdx]

	slotsUntilEpochEnd := spec.SlotsPerEpoch - (slot % spec.SlotsPerEpoch) - 1

	return spec.SlotsPerEpoch + int(slotsUntilEpochEnd)
}
