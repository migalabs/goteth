package custom_spec

import (
	"bytes"
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "custom_spec",
	)
)

type Phase0Spec struct {
	WrappedState                  ForkStateContent
	PreviousEpochAttestingVals    []uint64
	PreviousEpochAttestingBalance uint64
	ValAttestationInclusion       map[uint64]ValVote // key is valID
	AttestedValsPerSlot           map[uint64][]uint64
}

func NewPhase0Spec(bstate *spec.VersionedBeaconState, prevBstate spec.VersionedBeaconState, iApi *http.Service) Phase0Spec {
	// func NewPhase0Spec(bstate *spec.VersionedBeaconState, iCli *clientapi.APIClient) Phase0Spec {

	if prevBstate.Phase0 == nil {
		prevBstate = *bstate
	}

	phase0Obj := Phase0Spec{
		WrappedState: ForkStateContent{
			BState:           *bstate,
			PrevBState:       prevBstate,
			Api:              iApi,
			PrevEpochStructs: NewEpochData(iApi, prevBstate.Phase0.Slot),
			EpochStructs:     NewEpochData(iApi, bstate.Phase0.Slot),
		},

		PreviousEpochAttestingVals:    make([]uint64, len(prevBstate.Phase0.Validators)),
		PreviousEpochAttestingBalance: 0,
		ValAttestationInclusion:       make(map[uint64]ValVote),
		AttestedValsPerSlot:           make(map[uint64][]uint64),
		// the maximum inclusionDelay is 32, and we are counting aggregations from the current Epoch
	}

	phase0Obj.WrappedState.InitializeArrays(uint64(len(bstate.Phase0.Validators)))

	var attestations []*phase0.PendingAttestation
	if len(bstate.Phase0.PreviousEpochAttestations) > 0 {
		// we are not in genesis
		attestations = bstate.Phase0.PreviousEpochAttestations
	} else {
		// we are in genesis
		attestations = bstate.Phase0.CurrentEpochAttestations
	}

	phase0Obj.PreviousEpochAttestingVals = phase0Obj.CalculateAttestingVals(attestations, uint64(len(prevBstate.Phase0.Validators)))
	phase0Obj.PreviousEpochAttestingBalance = phase0Obj.ValsEffectiveBalance(phase0Obj.PreviousEpochAttestingVals)
	phase0Obj.WrappedState.TotalActiveBalance = phase0Obj.GetTotalActiveEffBalance()
	phase0Obj.CalculateValAttestationInclusion()
	phase0Obj.TrackMissingBlocks()
	return phase0Obj

}

func (p *Phase0Spec) CalculateAttestingVals(attestations []*phase0.PendingAttestation, valNum uint64) []uint64 {

	resultAttVals := make([]uint64, valNum)

	for _, item := range attestations {

		slot := item.Data.Slot            // Block that is being attested, not included
		committeeIndex := item.Data.Index // committee in the attested slot

		validatorIDs := p.WrappedState.PrevEpochStructs.GetValList(uint64(slot), uint64(committeeIndex))

		attestingIndices := item.AggregationBits.BitIndices()

		for _, index := range attestingIndices {
			attestingValIdx := validatorIDs[index]

			resultAttVals[attestingValIdx] = resultAttVals[attestingValIdx] + 1

			if p.IsCorrectSource() {
				p.WrappedState.CorrectFlags[altair.TimelySourceFlagIndex][attestingValIdx] = true
			}

			if p.IsCorrectTarget(*item) {
				p.WrappedState.CorrectFlags[altair.TimelyTargetFlagIndex][attestingValIdx] = true
			}

			if p.IsCorrectHead(*item) {
				p.WrappedState.CorrectFlags[altair.TimelyHeadFlagIndex][attestingValIdx] = true
			}
		}

		// measure missing head target and source

	}

	return resultAttVals
}

// the length of the valList = number of validators
// each position represents a valIdx
// if the item has a number > 0, count it
func (p Phase0Spec) ValsEffectiveBalance(valList []uint64) uint64 {

	attestingBalance := uint64(0)

	for valIdx, numAtt := range valList { // loop over validators
		if numAtt > 0 {
			attestingBalance += uint64(p.WrappedState.BState.Phase0.Validators[valIdx].EffectiveBalance)
		}
	}

	return uint64(attestingBalance)
}

func (p Phase0Spec) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.WrappedState.BState.Phase0.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.WrappedState.BState.Phase0.Slot)
		return 0, err
	}
	balance := p.WrappedState.BState.Phase0.Balances[valIdx]

	return balance, nil
}

func (p Phase0Spec) GetTotalActiveEffBalance() uint64 {

	all_vals := p.WrappedState.BState.Phase0.Validators
	val_array := make([]uint64, len(all_vals))

	for idx := range val_array {
		if IsActive(*all_vals[idx], phase0.Epoch(p.CurrentEpoch())) {
			val_array[idx] += 1
		}

	}

	return p.ValsEffectiveBalance(val_array)
}

func (p Phase0Spec) CalculateValAttestationInclusion() {

	// we only look at attestations referring the previous epoch
	attestations := p.WrappedState.BState.Phase0.PreviousEpochAttestations

	for _, item := range attestations {

		slot := item.Data.Slot            // Block that is being attested, not included
		committeeIndex := item.Data.Index // committee in the attested slot
		inclusionSlot := slot + item.InclusionDelay

		attestingIndices := item.AggregationBits.BitIndices()

		committee := p.WrappedState.PrevEpochStructs.GetValList(uint64(slot), uint64(committeeIndex))

		if committee == nil {
			committee = p.WrappedState.EpochStructs.GetValList(uint64(slot), uint64(committeeIndex))
		}

		// loop over the vals that attested
		for _, index := range attestingIndices {
			valID := committee[index]

			if val, ok := p.ValAttestationInclusion[uint64(valID)]; ok {
				// it already existed
				val.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.ValAttestationInclusion[uint64(valID)] = val
			} else {

				// it did not exist
				newAtt := ValVote{
					ValId: uint64(valID),
				}
				newAtt.AddNewAtt(uint64(slot), uint64(inclusionSlot))
				p.ValAttestationInclusion[uint64(valID)] = newAtt

			}
		}
	}

}

func (p Phase0Spec) PrevEpochReward(valIdx uint64) int64 {
	return int64(p.WrappedState.BState.Phase0.Balances[valIdx] - p.WrappedState.PrevBState.Phase0.Balances[valIdx])
}

func (p Phase0Spec) GetMaxProposerReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	isProposer := false
	proposerSlot := 0
	duties := append(p.WrappedState.EpochStructs.ProposerDuties, p.WrappedState.PrevEpochStructs.ProposerDuties...)
	for _, duty := range duties {
		if duty.ValidatorIndex == phase0.ValidatorIndex(valIdx) {
			isProposer = true
			proposerSlot = int(duty.Slot)
			break
		}
	}

	if isProposer {
		votesIncluded := 0
		for _, valAttestation := range p.ValAttestationInclusion {
			for _, item := range valAttestation.InclusionSlot {
				if item == uint64(proposerSlot) {
					// the block the attestation was included is the same as the slot the val proposed a block
					// therefore, proposer included this attestation
					votesIncluded += 1
				}
			}
		}

		baseReward := GetBaseReward(valEffectiveBalance, totalEffectiveBalance)
		return (baseReward / PROPOSER_REWARD_QUOTIENT) * float64(votesIncluded)
	}

	return 0
}

func (p Phase0Spec) GetMaxReward(valIdx uint64) (uint64, error) {

	if p.CurrentEpoch() == GENESIS_EPOCH { // No rewards are applied at genesis
		return 0, nil
	}
	valEffectiveBalance := p.WrappedState.PrevBState.Phase0.Validators[valIdx].EffectiveBalance
	activeBalance := p.GetTotalActiveEffBalance()
	baseReward := GetBaseReward(uint64(valEffectiveBalance), activeBalance)
	voteReward := float64(0)
	inclusionDelayReward := float64(0)

	for _, item := range p.WrappedState.CorrectFlags {

		if !item[valIdx] { // if this validators vote was not in an attestation with correct flag, dont sum
			continue
		}

		previousAttestedBalance := p.ValsEffectiveBalance(utils.BoolToUint(item))

		participationRate := float64(previousAttestedBalance) / float64(activeBalance)

		// First iteration just taking 31/8*BaseReward as Max value
		// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )

		// apply formula

		voteReward += baseReward * participationRate
	}

	if p.WrappedState.CorrectFlags[altair.TimelySourceFlagIndex][valIdx] {
		// only add it when there was an attestation (correct source)
		inclusionDelayReward = baseReward * 7.0 / 8.0
	}

	proposerReward := p.GetMaxProposerReward(valIdx, uint64(valEffectiveBalance), activeBalance)

	maxReward := voteReward + inclusionDelayReward + proposerReward

	return uint64(maxReward), nil
}

func (p Phase0Spec) GetAttSlot(valIdx uint64) int64 {

	for _, item := range p.ValAttestationInclusion[valIdx].AttestedSlot {
		// we are looking for a vote to the previous epoch
		if item >= p.WrappedState.PrevBState.Phase0.Slot+1-SLOTS_PER_EPOCH &&
			item <= p.WrappedState.PrevBState.Phase0.Slot {
			return int64(item)
		}
	}
	return -1
}

func (p Phase0Spec) GetAttInclusionSlot(valIdx uint64) int64 {

	for i, item := range p.ValAttestationInclusion[valIdx].AttestedSlot {
		// we are looking for a vote to the previous epoch
		if item >= p.WrappedState.PrevBState.Phase0.Slot+1-SLOTS_PER_EPOCH &&
			item <= p.WrappedState.PrevBState.Phase0.Slot {
			return int64(p.ValAttestationInclusion[valIdx].InclusionSlot[i])
		}
	}
	return -1
}

func (p Phase0Spec) CurrentSlot() uint64 {
	return p.WrappedState.BState.Phase0.Slot
}

func (p Phase0Spec) CurrentEpoch() uint64 {
	return uint64(p.CurrentSlot() / 32)
}

func (p Phase0Spec) PrevStateSlot() uint64 {
	return p.WrappedState.PrevBState.Phase0.Slot
}

func (p Phase0Spec) PrevStateEpoch() uint64 {
	return uint64(p.PrevStateSlot() / 32)
}

func (p Phase0Spec) IsCorrectSource() bool {
	epoch := phase0.Epoch(p.WrappedState.BState.Phase0.Slot / SLOTS_PER_EPOCH)
	currentEpoch := phase0.Epoch(p.WrappedState.BState.Phase0.Slot / SLOTS_PER_EPOCH)
	prevEpoch := phase0.Epoch(p.WrappedState.PrevBState.Phase0.Slot / SLOTS_PER_EPOCH)
	if epoch == currentEpoch || epoch == prevEpoch {
		return true
	}
	return false
}

func (p Phase0Spec) IsCorrectTarget(attestation phase0.PendingAttestation) bool {
	target := attestation.Data.Target.Root

	slot := int(p.WrappedState.PrevBState.Phase0.Slot / SLOTS_PER_EPOCH)
	slot = slot * SLOTS_PER_EPOCH
	expected := p.WrappedState.PrevBState.Phase0.BlockRoots[slot%SLOTS_PER_HISTORICAL_ROOT]

	res := bytes.Compare(target[:], expected)

	return res == 0 // if 0, then block roots are the same
}

func (p Phase0Spec) IsCorrectHead(attestation phase0.PendingAttestation) bool {
	head := attestation.Data.BeaconBlockRoot

	index := attestation.Data.Slot % SLOTS_PER_HISTORICAL_ROOT
	expected := p.WrappedState.BState.Phase0.BlockRoots[index]

	res := bytes.Compare(head[:], expected)
	return res == 0 // if 0, then block roots are the same
}

func (p Phase0Spec) GetMissedBlocks() []uint64 {
	return p.WrappedState.MissedBlocks
}

func (p *Phase0Spec) TrackMissingBlocks() {
	firstIndex := p.WrappedState.BState.Phase0.Slot - SLOTS_PER_EPOCH + 1
	lastIndex := p.WrappedState.BState.Phase0.Slot

	for i := firstIndex; i <= lastIndex; i++ {
		if i == 0 {
			continue
		}
		lastItem := p.WrappedState.BState.Phase0.BlockRoots[i-1]
		item := p.WrappedState.BState.Phase0.BlockRoots[i]
		res := bytes.Compare(lastItem, item)

		if res == 0 {
			// both roots were the same ==> missed block
			p.WrappedState.MissedBlocks = append(p.WrappedState.MissedBlocks, uint64(i))
		}
	}
}

// Argument: 0 for source, 1 for target and 2 for head
func (p Phase0Spec) GetMissingFlag(flagIndex int) uint64 {
	result := uint64(0)
	for _, item := range p.WrappedState.CorrectFlags[flagIndex] {
		if !item {
			result += 1
		}
	}

	return result
}

func (p Phase0Spec) GetTotalActiveBalance() uint64 {
	all_vals := p.WrappedState.BState.Phase0.Validators
	totalBalance := uint64(0)

	for idx := range all_vals {
		if IsActive(*all_vals[idx], phase0.Epoch(p.CurrentEpoch())) {
			totalBalance += p.WrappedState.BState.Phase0.Balances[idx]
		}

	}
	return totalBalance
}

func (p Phase0Spec) GetAttestingValNum() uint64 {
	result := uint64(0)

	for _, item := range p.PreviousEpochAttestingVals {
		if item > 0 {
			result += 1
		}
	}

	return result
}

func (p Phase0Spec) GetAttNum() uint64 {

	return uint64(len(p.WrappedState.BState.Phase0.PreviousEpochAttestations))
}

func (p Phase0Spec) GetBaseReward(valIdx uint64) float64 {
	effectiveBalance := p.WrappedState.BState.Phase0.Validators[valIdx].EffectiveBalance
	totalEffBalance := p.GetTotalActiveEffBalance()
	return GetBaseReward(uint64(effectiveBalance), totalEffBalance)
}
