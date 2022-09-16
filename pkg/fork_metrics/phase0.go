package fork_metrics

import (
	"bytes"
	"math"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth2-state-analyzer/pkg/fork_metrics/fork_state"
)

// var (
// 	log = logrus.WithField(
// 		"module", "custom_spec",
// 	)
// )

type Phase0Metrics struct {
	StateMetricsBase
}

func NewPhase0Spec(nextBstate fork_state.ForkStateContentBase, currentState fork_state.ForkStateContentBase, prevState fork_state.ForkStateContentBase) Phase0Metrics {

	phase0Obj := Phase0Metrics{}
	phase0Obj.NextState = nextBstate
	phase0Obj.CurrentState = currentState
	phase0Obj.PrevState = prevState

	phase0Obj.CalculateAttestingVals()
	return phase0Obj

}

func (p Phase0Metrics) GetMetricsBase() StateMetricsBase {
	return p.StateMetricsBase
}

// Processes attestations and fills several structs
func (p *Phase0Metrics) CalculateAttestingVals() {

	for _, item := range p.CurrentState.PrevAttestations {

		slot := item.Data.Slot            // Block that is being attested, not included
		committeeIndex := item.Data.Index // committee in the attested slot
		inclusionSlot := slot + item.InclusionDelay

		validatorIDs := p.PrevState.EpochStructs.GetValList(uint64(slot), uint64(committeeIndex)) // Beacon Committee

		attestingIndices := item.AggregationBits.BitIndices() // we only get the 1s, meaning the validator voted

		for _, index := range attestingIndices {
			attestingValIdx := validatorIDs[index]

			p.CurrentState.AttestingVals[attestingValIdx] = true

			// add correct flags and balances
			if p.IsCorrectSource() {
				p.CurrentState.CorrectFlags[altair.TimelySourceFlagIndex][attestingValIdx] += 1
				p.CurrentState.AttestingBalance[altair.TimelySourceFlagIndex] += uint64(p.CurrentState.Validators[attestingValIdx].EffectiveBalance)
			}

			if p.IsCorrectTarget(*item) {
				p.CurrentState.CorrectFlags[altair.TimelyTargetFlagIndex][attestingValIdx] += 1
				p.CurrentState.AttestingBalance[altair.TimelyTargetFlagIndex] += uint64(p.CurrentState.Validators[attestingValIdx].EffectiveBalance)
			}

			if p.IsCorrectHead(*item) {
				p.CurrentState.CorrectFlags[altair.TimelyHeadFlagIndex][attestingValIdx] += 1
				p.CurrentState.AttestingBalance[altair.TimelyHeadFlagIndex] += uint64(p.CurrentState.Validators[attestingValIdx].EffectiveBalance)
			}

			// we also organize which validator attested when, and when was the attestation included
			if val, ok := p.CurrentState.ValAttestationInclusion[uint64(attestingValIdx)]; ok {
				// it already existed
				val.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.CurrentState.ValAttestationInclusion[uint64(attestingValIdx)] = val
			} else {

				// it did not exist
				newAtt := fork_state.ValVote{
					ValId: uint64(attestingValIdx),
				}
				newAtt.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.CurrentState.ValAttestationInclusion[uint64(attestingValIdx)] = newAtt

			}
		}

	}
}

func (p Phase0Metrics) GetMaxProposerReward(valIdx uint64, baseReward float64) (float64, int64) {

	isProposer := false
	proposerSlot := 0
	duties := append(p.CurrentState.EpochStructs.ProposerDuties, p.PrevState.EpochStructs.ProposerDuties...)
	// there will be no duties if the validator is not active
	for _, duty := range duties {
		if duty.ValidatorIndex == phase0.ValidatorIndex(valIdx) {
			isProposer = true
			proposerSlot = int(duty.Slot)
			break
		}
	}

	if isProposer {
		votesIncluded := 0
		for _, valAttestation := range p.CurrentState.ValAttestationInclusion {
			for _, item := range valAttestation.InclusionSlot {
				if item == uint64(proposerSlot) {
					// the block the attestation was included is the same as the slot the val proposed a block
					// therefore, proposer included this attestation
					votesIncluded += 1
				}
			}
		}
		if votesIncluded > 0 {
			return (baseReward / fork_state.PROPOSER_REWARD_QUOTIENT) * float64(votesIncluded), int64(proposerSlot)
		}

	}

	return 0, -1
}

func (p Phase0Metrics) GetMaxReward(valIdx uint64) (ValidatorSepRewards, error) {

	if p.CurrentState.Epoch == fork_state.GENESIS_EPOCH { // No rewards are applied at genesis
		return ValidatorSepRewards{}, nil
	}
	baseReward := p.GetBaseReward(uint64(p.CurrentState.Validators[valIdx].EffectiveBalance))
	voteReward := float64(0)
	proposerReward := float64(0)
	proposerSlot := int64(-1)
	maxReward := float64(0)
	inclusionDelayReward := float64(0)

	if fork_state.IsActive(*p.NextState.Validators[valIdx], phase0.Epoch(p.PrevState.Epoch)) {
		// only consider attestations rewards in case the validator was active in the previous epoch
		for i := range p.CurrentState.CorrectFlags {

			previousAttestedBalance := p.CurrentState.AttestingBalance[i]

			// participationRate per flag
			participationRate := float64(previousAttestedBalance) / float64(p.CurrentState.TotalActiveBalance)

			// for each flag, we add baseReward * participationRate
			voteReward += baseReward * participationRate
		}

		// TODO: remove this as we are calculating max reward
		if p.CurrentState.CorrectFlags[altair.TimelySourceFlagIndex][valIdx] > 0 {
			// only add it when there was an attestation (correct source)
			inclusionDelayReward = baseReward * 7.0 / 8.0
		}
	}

	_, proposerSlot = p.GetMaxProposerReward(valIdx, baseReward)
	maxReward = voteReward + inclusionDelayReward + proposerReward

	result := ValidatorSepRewards{
		Attestation:     voteReward,
		InclusionDelay:  inclusionDelayReward,
		FlagIndex:       0,
		SyncCommittee:   0,
		MaxReward:       maxReward,
		BaseReward:      baseReward,
		ProposerSlot:    proposerSlot,
		InSyncCommittee: false,
	}
	return result, nil
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectSource() bool {
	epoch := phase0.Epoch(p.CurrentState.Slot / fork_state.SLOTS_PER_EPOCH)
	if epoch == phase0.Epoch(p.CurrentState.Epoch) || epoch == phase0.Epoch(p.PrevState.Epoch) {
		return true
	}
	return false
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectTarget(attestation phase0.PendingAttestation) bool {
	target := attestation.Data.Target.Root

	slot := int(p.PrevState.Slot / fork_state.SLOTS_PER_EPOCH)
	slot = slot * fork_state.SLOTS_PER_EPOCH
	expected := p.PrevState.BlockRoots[slot%fork_state.SLOTS_PER_HISTORICAL_ROOT]

	res := bytes.Compare(target[:], expected)

	return res == 0 // if 0, then block roots are the same
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectHead(attestation phase0.PendingAttestation) bool {
	head := attestation.Data.BeaconBlockRoot

	index := attestation.Data.Slot % fork_state.SLOTS_PER_HISTORICAL_ROOT
	expected := p.CurrentState.BlockRoots[index]

	res := bytes.Compare(head[:], expected)
	return res == 0 // if 0, then block roots are the same
}

// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helpers
func (p Phase0Metrics) GetBaseReward(valEffectiveBalance uint64) float64 {

	var baseReward float64

	sqrt := uint64(math.Sqrt(float64(p.CurrentState.TotalActiveBalance)))

	denom := float64(fork_state.BASE_REWARD_PER_EPOCH * sqrt)

	num := (float64(valEffectiveBalance) * fork_state.BASE_REWARD_FACTOR)
	baseReward = num / denom

	return baseReward
}
