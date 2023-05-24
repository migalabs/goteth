package metrics

import (
	"bytes"
	"math"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

type Phase0Metrics struct {
	baseMetrics StateMetricsBase
}

func NewPhase0Metrics(
	firstState spec.AgnosticState,
	secondState spec.AgnosticState,
	thirdState spec.AgnosticState,
	fourthState spec.AgnosticState) Phase0Metrics {

	phase0Obj := Phase0Metrics{}
	phase0Obj.baseMetrics.FirstState = firstState
	phase0Obj.baseMetrics.SecondState = secondState
	phase0Obj.baseMetrics.ThirdState = thirdState
	phase0Obj.baseMetrics.FourthState = fourthState

	phase0Obj.CalculateAttestingVals()
	return phase0Obj

}

func (p Phase0Metrics) GetMetricsBase() StateMetricsBase {
	return p.baseMetrics
}

// Processes attestations and fills several structs
func (p *Phase0Metrics) CalculateAttestingVals() {

	for _, item := range p.baseMetrics.SecondState.PrevAttestations {

		slot := item.Data.Slot            // Block that is being attested, not included
		committeeIndex := item.Data.Index // committee in the attested slot
		inclusionSlot := slot + item.InclusionDelay

		validatorIDs := p.baseMetrics.FirstState.EpochStructs.GetValList(uint64(slot), uint64(committeeIndex)) // Beacon Committee

		attestingIndices := item.AggregationBits.BitIndices() // we only get the 1s, meaning the validator voted

		for _, index := range attestingIndices {
			attestingValIdx := validatorIDs[index]

			p.baseMetrics.SecondState.AttestingVals[attestingValIdx] = true

			// add correct flags and balances
			if p.IsCorrectSource(*item) && p.baseMetrics.SecondState.CorrectFlags[altair.TimelySourceFlagIndex][attestingValIdx] == 0 {
				p.baseMetrics.SecondState.CorrectFlags[altair.TimelySourceFlagIndex][attestingValIdx] += 1
				p.baseMetrics.SecondState.AttestingBalance[altair.TimelySourceFlagIndex] += p.baseMetrics.SecondState.Validators[attestingValIdx].EffectiveBalance
			}

			if p.IsCorrectTarget(*item) && p.baseMetrics.SecondState.CorrectFlags[altair.TimelyTargetFlagIndex][attestingValIdx] == 0 {
				p.baseMetrics.SecondState.CorrectFlags[altair.TimelyTargetFlagIndex][attestingValIdx] += 1
				p.baseMetrics.SecondState.AttestingBalance[altair.TimelyTargetFlagIndex] += p.baseMetrics.SecondState.Validators[attestingValIdx].EffectiveBalance
			}

			if p.IsCorrectHead(*item) && p.baseMetrics.SecondState.CorrectFlags[altair.TimelyHeadFlagIndex][attestingValIdx] == 0 {
				p.baseMetrics.SecondState.CorrectFlags[altair.TimelyHeadFlagIndex][attestingValIdx] += 1
				p.baseMetrics.SecondState.AttestingBalance[altair.TimelyHeadFlagIndex] += p.baseMetrics.SecondState.Validators[attestingValIdx].EffectiveBalance
			}

			// we also organize which validator attested when, and when was the attestation included
			if val, ok := p.baseMetrics.SecondState.ValAttestationInclusion[attestingValIdx]; ok {
				// it already existed
				val.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.baseMetrics.SecondState.ValAttestationInclusion[attestingValIdx] = val
			} else {

				// it did not exist
				newAtt := spec.ValVote{
					ValId: uint64(attestingValIdx),
				}
				newAtt.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.baseMetrics.SecondState.ValAttestationInclusion[attestingValIdx] = newAtt
				p.baseMetrics.SecondState.NumAttestingVals++

			}
		}

	}

}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#components-of-attestation-deltas
func (p Phase0Metrics) GetMaxProposerReward(valIdx phase0.ValidatorIndex, baseReward phase0.Gwei) (phase0.Gwei, phase0.Slot) {

	isProposer := false
	proposerSlot := phase0.Slot(0)
	duties := append(p.baseMetrics.FirstState.EpochStructs.ProposerDuties, p.baseMetrics.SecondState.EpochStructs.ProposerDuties...)
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
		for _, valAttestation := range p.baseMetrics.SecondState.ValAttestationInclusion {
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

	if p.baseMetrics.ThirdState.Epoch == 0 { // No rewards are applied at genesis
		return spec.ValidatorRewards{}, nil
	}

	if valIdx >= phase0.ValidatorIndex(len(p.baseMetrics.ThirdState.Validators)) || !spec.IsActive(*p.baseMetrics.ThirdState.Validators[valIdx], phase0.Epoch(p.baseMetrics.FirstState.Epoch)) {
		return spec.ValidatorRewards{}, nil
	}
	// only consider attestations rewards in case the validator was active in the previous epoch
	baseReward := p.GetBaseReward(p.baseMetrics.SecondState.Validators[valIdx].EffectiveBalance)
	proposerReward := phase0.Gwei(0)
	proposerSlot := phase0.Slot(0)
	maxReward := phase0.Gwei(0)
	voteReward := phase0.Gwei(0)
	inclusionDelayReward := phase0.Gwei(0)

	for i := range p.baseMetrics.SecondState.CorrectFlags {

		previousAttestedBalance := p.baseMetrics.SecondState.AttestingBalance[i]

		// participationRate per flag ==> previousAttestBalance / TotalActiveBalance
		singleReward := baseReward * phase0.Gwei(previousAttestedBalance/spec.EffectiveBalanceInc)

		singleReward = singleReward / phase0.Gwei(p.baseMetrics.SecondState.TotalActiveBalance/spec.EffectiveBalanceInc)

		// for each flag, we add baseReward * participationRate
		voteReward += singleReward
	}

	proposerReward = phase0.Gwei(baseReward / spec.ProposerRewardQuotient)
	// only add it when there was an attestation (correct source)
	inclusionDelayReward = baseReward - proposerReward

	_, proposerSlot = p.GetMaxProposerReward(valIdx, baseReward)
	maxReward = voteReward + inclusionDelayReward
	// TODO: broken max reward in Phase0
	result := spec.ValidatorRewards{
		ValidatorIndex:      valIdx,
		Epoch:               p.baseMetrics.ThirdState.Epoch,
		ValidatorBalance:    p.baseMetrics.ThirdState.Balances[valIdx],
		Reward:              p.baseMetrics.EpochReward(valIdx),
		MaxReward:           maxReward,
		AttestationReward:   voteReward + inclusionDelayReward,
		SyncCommitteeReward: 0,
		AttSlot:             p.baseMetrics.FirstState.EpochStructs.ValidatorAttSlot[valIdx],
		MissingSource:       p.baseMetrics.SecondState.MissingFlags(valIdx)[altair.TimelySourceFlagIndex],
		MissingTarget:       p.baseMetrics.SecondState.MissingFlags(valIdx)[altair.TimelyTargetFlagIndex],
		MissingHead:         p.baseMetrics.SecondState.MissingFlags(valIdx)[altair.TimelyHeadFlagIndex],
		Status:              0,
		BaseReward:          baseReward,
		ProposerSlot:        proposerSlot,
		InSyncCommittee:     false,
	}
	return result, nil
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectSource(attestation phase0.PendingAttestation) bool {
	epoch := phase0.Epoch(attestation.Data.Slot / spec.SlotsPerEpoch)
	if epoch == phase0.Epoch(p.baseMetrics.SecondState.Epoch) || epoch == phase0.Epoch(p.baseMetrics.FirstState.Epoch) {
		return true
	}
	return false
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectTarget(attestation phase0.PendingAttestation) bool {
	if !p.IsCorrectSource(attestation) { // needs to be matching source first of all
		return false
	}
	target := attestation.Data.Target.Root

	slot := phase0.Slot((p.baseMetrics.FirstState.Epoch) * spec.SlotsPerEpoch)

	expected := p.baseMetrics.FirstState.BlockRoots[slot%spec.SlotsPerHistoricalRoot]

	res := bytes.Compare(target[:], expected)

	return res == 0 // if 0, then block roots are the same
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectHead(attestation phase0.PendingAttestation) bool {
	if !p.IsCorrectTarget(attestation) { // needs to be matching target first of all
		return false
	}

	head := attestation.Data.BeaconBlockRoot

	expected := p.baseMetrics.SecondState.BlockRoots[attestation.Data.Slot%spec.SlotsPerHistoricalRoot]

	res := bytes.Compare(head[:], expected)
	return res == 0 // if 0, then block roots are the same
}

// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helpers
func (p Phase0Metrics) GetBaseReward(valEffectiveBalance phase0.Gwei) phase0.Gwei {

	baseReward := valEffectiveBalance * phase0.Gwei(spec.BaseRewardFactor)
	baseReward = baseReward / phase0.Gwei(math.Sqrt(float64(p.baseMetrics.SecondState.TotalActiveBalance)))
	baseReward = baseReward / phase0.Gwei(spec.BaseRewardPerEpoch)

	return baseReward
}
