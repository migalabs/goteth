package state_metrics

import (
	"bytes"
	"math"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

type Phase0Metrics struct {
	StateMetricsBase
}

func NewPhase0Metrics(nextBstate fork_state.ForkStateContentBase, currentState fork_state.ForkStateContentBase, prevState fork_state.ForkStateContentBase) Phase0Metrics {

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
			if p.IsCorrectSource() && p.CurrentState.CorrectFlags[altair.TimelySourceFlagIndex][attestingValIdx] == 0 {
				p.CurrentState.CorrectFlags[altair.TimelySourceFlagIndex][attestingValIdx] += 1
				p.CurrentState.AttestingBalance[altair.TimelySourceFlagIndex] += p.CurrentState.Validators[attestingValIdx].EffectiveBalance
			}

			if p.IsCorrectTarget(*item) && p.CurrentState.CorrectFlags[altair.TimelyTargetFlagIndex][attestingValIdx] == 0 {
				p.CurrentState.CorrectFlags[altair.TimelyTargetFlagIndex][attestingValIdx] += 1
				p.CurrentState.AttestingBalance[altair.TimelyTargetFlagIndex] += p.CurrentState.Validators[attestingValIdx].EffectiveBalance
			}

			if p.IsCorrectHead(*item) && p.CurrentState.CorrectFlags[altair.TimelyHeadFlagIndex][attestingValIdx] == 0 {
				p.CurrentState.CorrectFlags[altair.TimelyHeadFlagIndex][attestingValIdx] += 1
				p.CurrentState.AttestingBalance[altair.TimelyHeadFlagIndex] += p.CurrentState.Validators[attestingValIdx].EffectiveBalance
			}

			// we also organize which validator attested when, and when was the attestation included
			if val, ok := p.CurrentState.ValAttestationInclusion[attestingValIdx]; ok {
				// it already existed
				val.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.CurrentState.ValAttestationInclusion[attestingValIdx] = val
			} else {

				// it did not exist
				newAtt := fork_state.ValVote{
					ValId: uint64(attestingValIdx),
				}
				newAtt.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.CurrentState.ValAttestationInclusion[attestingValIdx] = newAtt

			}
		}

	}
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#components-of-attestation-deltas
func (p Phase0Metrics) GetMaxProposerReward(valIdx phase0.ValidatorIndex, baseReward phase0.Gwei) (phase0.Gwei, phase0.Slot) {

	isProposer := false
	proposerSlot := phase0.Slot(0)
	duties := append(p.CurrentState.EpochStructs.ProposerDuties, p.PrevState.EpochStructs.ProposerDuties...)
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
			return phase0.Gwei(baseReward/utils.PROPOSER_REWARD_QUOTIENT) * phase0.Gwei(votesIncluded), proposerSlot
		}

	}

	return 0, 0
}

// TODO: review formulas
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#rewards-and-penalties-1
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#components-of-attestation-deltas
func (p Phase0Metrics) GetMaxReward(valIdx phase0.ValidatorIndex) (model.ValidatorRewards, error) {

	if p.CurrentState.Epoch == utils.GENESIS_EPOCH { // No rewards are applied at genesis
		return model.ValidatorRewards{}, nil
	}

	if valIdx >= phase0.ValidatorIndex(len(p.NextState.Validators)) || !fork_state.IsActive(*p.NextState.Validators[valIdx], phase0.Epoch(p.PrevState.Epoch)) {
		return model.ValidatorRewards{}, nil
	}
	// only consider attestations rewards in case the validator was active in the previous epoch
	baseReward := p.GetBaseReward(p.CurrentState.Validators[valIdx].EffectiveBalance)
	voteReward := phase0.Gwei(0)
	proposerReward := phase0.Gwei(0)
	proposerSlot := phase0.Slot(0)
	maxReward := phase0.Gwei(0)
	inclusionDelayReward := phase0.Gwei(0)

	for i := range p.CurrentState.CorrectFlags {

		previousAttestedBalance := p.CurrentState.AttestingBalance[i]

		// participationRate per flag ==> previousAttestBalance / TotalActiveBalance
		singleReward := baseReward * (previousAttestedBalance / utils.EFFECTIVE_BALANCE_INCREMENT)

		// for each flag, we add baseReward * participationRate
		voteReward += singleReward / (p.CurrentState.TotalActiveBalance / utils.EFFECTIVE_BALANCE_INCREMENT)
	}

	proposerReward = baseReward / utils.PROPOSER_REWARD_QUOTIENT
	// only add it when there was an attestation (correct source)
	inclusionDelayReward = baseReward - proposerReward

	_, proposerSlot = p.GetMaxProposerReward(valIdx, baseReward)
	maxReward = voteReward + inclusionDelayReward + proposerReward

	result := model.ValidatorRewards{
		ValidatorIndex:      valIdx,
		Epoch:               p.NextState.Epoch,
		ValidatorBalance:    p.CurrentState.Balances[valIdx],
		Reward:              p.EpochReward(valIdx),
		MaxReward:           maxReward,
		AttestationReward:   voteReward + inclusionDelayReward,
		SyncCommitteeReward: 0,
		AttSlot:             p.CurrentState.EpochStructs.ValidatorAttSlot[valIdx],
		MissingSource:       false,
		MissingTarget:       false,
		MissingHead:         false,
		Status:              0,
		BaseReward:          baseReward,
		ProposerSlot:        proposerSlot,
		InSyncCommittee:     false,
	}
	return result, nil
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectSource() bool {
	epoch := phase0.Epoch(p.CurrentState.Slot / utils.SLOTS_PER_EPOCH)
	if epoch == p.CurrentState.Epoch || epoch == p.PrevState.Epoch {
		return true
	}
	return false
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectTarget(attestation phase0.PendingAttestation) bool {
	target := attestation.Data.Target.Root

	slot := p.PrevState.Slot / utils.SLOTS_PER_EPOCH
	slot = slot * utils.SLOTS_PER_EPOCH
	expected := p.PrevState.BlockRoots[slot%utils.SLOTS_PER_HISTORICAL_ROOT]

	res := bytes.Compare(target[:], expected)

	return res == 0 // if 0, then block roots are the same
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectHead(attestation phase0.PendingAttestation) bool {
	head := attestation.Data.BeaconBlockRoot

	index := attestation.Data.Slot % utils.SLOTS_PER_HISTORICAL_ROOT
	expected := p.CurrentState.BlockRoots[index]

	res := bytes.Compare(head[:], expected)
	return res == 0 // if 0, then block roots are the same
}

// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helpers
func (p Phase0Metrics) GetBaseReward(valEffectiveBalance phase0.Gwei) phase0.Gwei {

	var baseReward phase0.Gwei

	sqrt := math.Sqrt(float64(p.CurrentState.TotalActiveBalance))

	denom := utils.BASE_REWARD_PER_EPOCH * sqrt

	num := (valEffectiveBalance * utils.BASE_REWARD_FACTOR)
	baseReward = phase0.Gwei(num) / phase0.Gwei(denom)

	return baseReward
}
