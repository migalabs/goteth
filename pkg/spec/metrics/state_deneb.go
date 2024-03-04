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
