package metrics

import (
	"bytes"
	"math"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/migalabs/goteth/pkg/spec"
)

type Phase0Metrics struct {
	baseMetrics StateMetricsBase
}

func NewPhase0Metrics(nextBstate *spec.AgnosticState, currentState *spec.AgnosticState, prevState *spec.AgnosticState) Phase0Metrics {

	phase0Obj := Phase0Metrics{}
	phase0Obj.baseMetrics.NextState = nextBstate
	phase0Obj.baseMetrics.CurrentState = currentState
	phase0Obj.baseMetrics.PrevState = prevState

	phase0Obj.CalculateAttestingVals()
	return phase0Obj

}

func (p Phase0Metrics) GetMetricsBase() StateMetricsBase {
	return p.baseMetrics
}

// Processes attestations and fills several structs
func (p *Phase0Metrics) CalculateAttestingVals() {

	for _, item := range p.baseMetrics.CurrentState.PrevAttestations {

		slot := item.Data.Slot            // Block that is being attested, not included
		committeeIndex := item.Data.Index // committee in the attested slot
		inclusionSlot := slot + item.InclusionDelay

		validatorIDs := p.baseMetrics.PrevState.EpochStructs.GetValList(slot, committeeIndex) // Beacon Committee

		attestingIndices := item.AggregationBits.BitIndices() // we only get the 1s, meaning the validator voted

		for _, index := range attestingIndices {
			attestingValIdx := validatorIDs[index]

			p.baseMetrics.CurrentState.AttestingVals[attestingValIdx] = true

			// add correct flags and balances
			if p.IsCorrectSource() && p.baseMetrics.CurrentState.CorrectFlags[altair.TimelySourceFlagIndex][attestingValIdx] == 0 {
				p.baseMetrics.CurrentState.CorrectFlags[altair.TimelySourceFlagIndex][attestingValIdx] += 1
				p.baseMetrics.CurrentState.AttestingBalance[altair.TimelySourceFlagIndex] += p.baseMetrics.CurrentState.Validators[attestingValIdx].EffectiveBalance
			}

			if p.IsCorrectTarget(*item) && p.baseMetrics.CurrentState.CorrectFlags[altair.TimelyTargetFlagIndex][attestingValIdx] == 0 {
				p.baseMetrics.CurrentState.CorrectFlags[altair.TimelyTargetFlagIndex][attestingValIdx] += 1
				p.baseMetrics.CurrentState.AttestingBalance[altair.TimelyTargetFlagIndex] += p.baseMetrics.CurrentState.Validators[attestingValIdx].EffectiveBalance
			}

			if p.IsCorrectHead(*item) && p.baseMetrics.CurrentState.CorrectFlags[altair.TimelyHeadFlagIndex][attestingValIdx] == 0 {
				p.baseMetrics.CurrentState.CorrectFlags[altair.TimelyHeadFlagIndex][attestingValIdx] += 1
				p.baseMetrics.CurrentState.AttestingBalance[altair.TimelyHeadFlagIndex] += p.baseMetrics.CurrentState.Validators[attestingValIdx].EffectiveBalance
			}

			// we also organize which validator attested when, and when was the attestation included
			if val, ok := p.baseMetrics.CurrentState.ValAttestationInclusion[attestingValIdx]; ok {
				// it already existed
				val.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.baseMetrics.CurrentState.ValAttestationInclusion[attestingValIdx] = val
			} else {

				// it did not exist
				newAtt := spec.ValVote{
					ValId: uint64(attestingValIdx),
				}
				newAtt.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.baseMetrics.CurrentState.ValAttestationInclusion[attestingValIdx] = newAtt

			}
		}

	}
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#components-of-attestation-deltas
func (p Phase0Metrics) GetMaxProposerReward(valIdx phase0.ValidatorIndex, baseReward phase0.Gwei) (phase0.Gwei, phase0.Slot) {

	isProposer := false
	proposerSlot := phase0.Slot(0)
	duties := append(p.baseMetrics.CurrentState.EpochStructs.ProposerDuties, p.baseMetrics.PrevState.EpochStructs.ProposerDuties...)
	// there will be no duties if the validator is not active
	for _, duty := range duties {
		if duty.ValidatorIndex == phase0.ValidatorIndex(valIdx) {
			isProposer = true
			proposerSlot = duty.Slot
			break
		}
	}

	if isProposer {
		votesIncluded := 0
		for _, valAttestation := range p.baseMetrics.CurrentState.ValAttestationInclusion {
			for _, item := range valAttestation.InclusionSlot {
				if item == uint64(proposerSlot) {
					// the block the attestation was included is the same as the slot the val proposed a block
					// therefore, proposer included this attestation
					votesIncluded += 1
				}
			}
		}
		if votesIncluded > 0 {
			return phase0.Gwei(baseReward/spec.ProposerRewardQuotient) * phase0.Gwei(votesIncluded), proposerSlot
		}

	}

	return 0, 0
}

// TODO: review formulas
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#rewards-and-penalties-1
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#components-of-attestation-deltas
func (p Phase0Metrics) GetMaxReward(valIdx phase0.ValidatorIndex) (spec.ValidatorRewards, error) {

	if p.baseMetrics.CurrentState.Epoch == 0 { // No rewards are applied at genesis
		return spec.ValidatorRewards{}, nil
	}

	if valIdx >= phase0.ValidatorIndex(len(p.baseMetrics.NextState.Validators)) || !spec.IsActive(*p.baseMetrics.NextState.Validators[valIdx], phase0.Epoch(p.baseMetrics.PrevState.Epoch)) {
		return spec.ValidatorRewards{}, nil
	}
	// only consider attestations rewards in case the validator was active in the previous epoch
	baseReward := p.GetBaseReward(p.baseMetrics.CurrentState.Validators[valIdx].EffectiveBalance)
	voteReward := phase0.Gwei(0)
	proposerReward := phase0.Gwei(0)
	proposerSlot := phase0.Slot(0)
	maxReward := phase0.Gwei(0)
	inclusionDelayReward := phase0.Gwei(0)

	for i := range p.baseMetrics.CurrentState.CorrectFlags {

		previousAttestedBalance := p.baseMetrics.CurrentState.AttestingBalance[i]

		// participationRate per flag ==> previousAttestBalance / TotalActiveBalance
		singleReward := baseReward * (previousAttestedBalance / spec.EffectiveBalanceInc)

		// for each flag, we add baseReward * participationRate
		voteReward += singleReward / (p.baseMetrics.CurrentState.TotalActiveBalance / spec.EffectiveBalanceInc)
	}

	proposerReward = baseReward / spec.ProposerRewardQuotient
	// only add it when there was an attestation (correct source)
	inclusionDelayReward = baseReward - proposerReward

	_, proposerSlot = p.GetMaxProposerReward(valIdx, baseReward)
	maxReward = voteReward + inclusionDelayReward + proposerReward

	result := spec.ValidatorRewards{
		ValidatorIndex:      valIdx,
		Epoch:               p.baseMetrics.NextState.Epoch,
		ValidatorBalance:    p.baseMetrics.CurrentState.Balances[valIdx],
		Reward:              p.baseMetrics.EpochReward(valIdx),
		MaxReward:           maxReward,
		AttestationReward:   voteReward + inclusionDelayReward,
		SyncCommitteeReward: 0,
		AttSlot:             p.baseMetrics.CurrentState.EpochStructs.ValidatorAttSlot[valIdx],
		MissingSource:       false,
		MissingTarget:       false,
		MissingHead:         false,
		Status:              p.baseMetrics.NextState.GetValStatus(valIdx),
		BaseReward:          baseReward,
		ProposerSlot:        proposerSlot,
		ProposerReward:      int64(proposerReward),
		InSyncCommittee:     false,
	}
	return result, nil
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectSource() bool {
	epoch := phase0.Epoch(p.baseMetrics.CurrentState.Slot / spec.SlotsPerEpoch)
	if epoch == p.baseMetrics.CurrentState.Epoch || epoch == p.baseMetrics.PrevState.Epoch {
		return true
	}
	return false
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectTarget(attestation phase0.PendingAttestation) bool {
	target := attestation.Data.Target.Root

	slot := p.baseMetrics.PrevState.Slot / spec.SlotsPerEpoch
	slot = slot * spec.SlotsPerEpoch
	expected := p.baseMetrics.PrevState.BlockRoots[slot%spec.SlotsPerHistoricalRoot]

	res := bytes.Compare(target[:], expected[:])

	return res == 0 // if 0, then block roots are the same
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectHead(attestation phase0.PendingAttestation) bool {
	head := attestation.Data.BeaconBlockRoot

	index := attestation.Data.Slot % spec.SlotsPerHistoricalRoot
	expected := p.baseMetrics.CurrentState.BlockRoots[index]

	res := bytes.Compare(head[:], expected[:])
	return res == 0 // if 0, then block roots are the same
}

// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helpers
func (p Phase0Metrics) GetBaseReward(valEffectiveBalance phase0.Gwei) phase0.Gwei {

	var baseReward phase0.Gwei

	sqrt := math.Sqrt(float64(p.baseMetrics.CurrentState.TotalActiveBalance))

	denom := spec.BaseRewardPerEpoch * sqrt

	num := (valEffectiveBalance * spec.BaseRewardFactor)
	baseReward = phase0.Gwei(num) / phase0.Gwei(denom)

	return baseReward
}
