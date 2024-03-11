package metrics

import (
	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
	local_spec "github.com/migalabs/goteth/pkg/spec"
	log "github.com/sirupsen/logrus"
)

type DenebMetrics struct {
	AltairMetrics
}

func NewDenebMetrics(
	nextBstate *spec.AgnosticState,
	bstate *spec.AgnosticState,
	prevBstate *spec.AgnosticState) DenebMetrics {

	denebObj := DenebMetrics{}
	denebObj.baseMetrics.CurrentState = bstate
	denebObj.baseMetrics.PrevState = prevBstate
	denebObj.baseMetrics.NextState = nextBstate
	denebObj.baseMetrics.BlockRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	denebObj.baseMetrics.SlashingRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	denebObj.baseMetrics.InclusionDelays = make(map[phase0.ValidatorIndex]int)
	denebObj.ProcessAttestations()
	prevBstateFilled := prevBstate.StateRoot != phase0.Root{}
	if prevBstateFilled {
		denebObj.ProcessInclusionDelays()
	}
	denebObj.ProcessSyncAggregates()
	denebObj.ProcessSlashings()

	return denebObj
}

func (p DenebMetrics) GetParticipationFlags(attestation phase0.Attestation, includedInBlock spec.AgnosticBlock) [3]bool {
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
	// https://github.com/ethereum/consensus-specs/blob/dev/specs/deneb/beacon-chain.md#modified-get_attestation_participation_flag_indices
	includedInEpoch := phase0.Epoch(includedInBlock.Slot / local_spec.SlotsPerEpoch)
	attestationEpoch := phase0.Epoch(attestation.Data.Slot / local_spec.SlotsPerEpoch)
	targetInclusionOk := includedInEpoch-attestationEpoch <= 1

	if matchingSource && (inclusionDelay <= int(math.Sqrt(local_spec.SlotsPerEpoch))) {
		result[0] = true
	}
	if matchingTarget && targetInclusionOk {
		result[1] = true
	}
	if matchingHead && (inclusionDelay <= local_spec.MinInclusionDelay) {
		result[2] = true
	}

	return result
}

func (p DenebMetrics) isFlagPossible(valIdx phase0.ValidatorIndex, flagIndex int) bool {
	attSlot := p.baseMetrics.PrevState.EpochStructs.ValidatorAttSlot[valIdx]
	maxInclusionDelay := 0

	switch flagIndex { // for every flag there is a max inclusion delay to obtain a reward

	case local_spec.AttSourceFlagIndex: // 5
		maxInclusionDelay = int(math.Sqrt(local_spec.SlotsPerEpoch))

	case local_spec.AttTargetFlagIndex: // until end of next epoch
		remainingSlotsInEpoch := local_spec.SlotsPerEpoch - int(attSlot%local_spec.SlotsPerEpoch)
		maxInclusionDelay = local_spec.SlotsPerEpoch + remainingSlotsInEpoch

	case local_spec.AttHeadFlagIndex: // 1
		maxInclusionDelay = 1
	default:
		log.Fatalf("provided flag index %d is not known", flagIndex)
	}

	// look for any block proposed => the attester could have achieved it
	for slot := attSlot + 1; slot <= (attSlot + phase0.Slot(maxInclusionDelay)); slot++ {
		slotInEpoch := slot % local_spec.SlotsPerEpoch
		block := p.baseMetrics.PrevState.Blocks[slotInEpoch]
		if slot >= phase0.Slot(p.baseMetrics.CurrentState.Epoch*local_spec.SlotsPerEpoch) {
			block = p.baseMetrics.CurrentState.Blocks[slotInEpoch]
		}

		if block.Proposed { // if there was a block proposed inside the inclusion window
			return true
		}
	}
	return false

}
