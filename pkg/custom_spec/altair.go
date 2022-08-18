package custom_spec

import (
	"bytes"
	"fmt"
	"math"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var ( // spec weight constants
	TIMELY_SOURCE_WEIGHT       = 14
	TIMELY_TARGET_WEIGHT       = 26
	TIMELY_HEAD_WEIGHT         = 14
	PARTICIPATING_FLAGS_WEIGHT = []int{TIMELY_SOURCE_WEIGHT, TIMELY_TARGET_WEIGHT, TIMELY_HEAD_WEIGHT}
	SYNC_REWARD_WEIGHT         = 2
	PROPOSER_WEIGHT            = 8
	WEIGHT_DENOMINATOR         = 64
	SYNC_COMMITTEE_SIZE        = 512
)

type AltairSpec struct {
	WrappedState     ForkStateContent
	AttestingVals    [][]uint64 // one array of validators per participating flag
	AttestingBalance []uint64   // one attesting balance per participation flag
}

func NewAltairSpec(bstate *spec.VersionedBeaconState, prevBstate spec.VersionedBeaconState, iApi *http.Service) AltairSpec {

	if prevBstate.Altair == nil {
		prevBstate = *bstate
	}

	attestingVals := make([][]uint64, 3)

	for i := range attestingVals {
		attestingVals[i] = make([]uint64, len(prevBstate.Altair.Validators))
	}

	altairObj := AltairSpec{
		WrappedState: ForkStateContent{
			PrevBState:   prevBstate,
			BState:       *bstate,
			Api:          iApi,
			EpochStructs: NewEpochData(iApi, bstate.Altair.Slot),
		},

		AttestingVals:    attestingVals,
		AttestingBalance: make([]uint64, 3),
	}

	// initialize missing flags arrays
	altairObj.WrappedState.InitializeArrays(uint64(len(bstate.Altair.Validators)))

	// calculate attesting vals only once
	altairObj.CalculatePreviousAttestingVals()

	// leave attestingBalance already calculated
	for i := range altairObj.AttestingBalance {
		altairObj.AttestingBalance[i] = altairObj.ValsEffectiveBalance(altairObj.AttestingVals[i])
	}
	altairObj.TrackMissingBlocks()

	return altairObj
}

// This method will calculate attesting vals to the previous epoch per flag
func (p *AltairSpec) CalculatePreviousAttestingVals() {

	flags := []altair.ParticipationFlag{
		altair.TimelySourceFlagIndex,
		altair.TimelyTargetFlagIndex,
		altair.TimelyHeadFlagIndex}

	for participatingFlag := range flags {

		flag := altair.ParticipationFlags(math.Pow(2, float64(participatingFlag)))

		for valIndex, item := range p.WrappedState.BState.Altair.PreviousEpochParticipation {
			// Here we have one item per validator
			// Item is a 3-bit string
			// each bit represents a flag

			if (item & flag) == flag {
				// The attestation has a timely flag, therefore we consider it correct flag
				p.AttestingVals[participatingFlag][valIndex] += uint64(1)
				p.WrappedState.CorrectFlags[participatingFlag][valIndex] = true
			}
		}
	}
}

// the length of the valList = number of validators
// each position represents a valIdx
// if the item has a number > 0, count it
// The method returns the sum of effective balance of selected validators.
func (p AltairSpec) ValsEffectiveBalance(valList []uint64) uint64 {

	combinedEffectiveBalance := uint64(0)

	for valIdx, numAtt := range valList { // loop over validators
		if numAtt > 0 {
			combinedEffectiveBalance += uint64(p.WrappedState.BState.Altair.Validators[valIdx].EffectiveBalance)
		}
	}

	return uint64(combinedEffectiveBalance)
}

// This method returns the Balance of the given validator at the current state
func (p AltairSpec) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.WrappedState.BState.Altair.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.WrappedState.BState.Altair.Slot)
		return 0, err
	}
	balance := p.WrappedState.BState.Altair.Balances[valIdx]

	return balance, nil
}

// This method returns the Effective Balance of all active validators
func (p AltairSpec) TotalActiveBalance() uint64 {

	if p.CurrentSlot() < 32 {
		// genesis epoch, validators preactivated with default balance
		return uint64(len(p.WrappedState.BState.Altair.Validators) * EFFECTIVE_BALANCE_INCREMENT * MAX_EFFECTIVE_INCREMENTS)
	}

	all_vals := p.WrappedState.BState.Altair.Validators
	val_array := make([]uint64, len(all_vals))

	for idx := range val_array {
		if IsActive(*all_vals[idx], phase0.Epoch(p.CurrentEpoch())) {
			val_array[idx] += 1
		}

	}

	return p.ValsEffectiveBalance(val_array)
}

func (p AltairSpec) GetMaxProposerAttReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	return 0

}

func (p AltairSpec) GetMaxProposerSyncReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	return 0

}

// So far we have computed the max sync committee proposer reward for a slot. Since the validator remains in the sync committee for the full epoch, we multiply the reward for the 32 slots in the epoch.
// TODO: Tracking missing blocks in the epoch would help us have an even more accurate theoretical sync proposer max reward per epoch.
func (p AltairSpec) GetMaxSyncComReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	inCommittee := false

	valPubKey := p.WrappedState.BState.Altair.Validators[valIdx].PublicKey

	syncCommitteePubKeys := p.WrappedState.BState.Altair.CurrentSyncCommittee

	for _, item := range syncCommitteePubKeys.Pubkeys {
		if valPubKey == item {
			inCommittee = true
		}
	}

	if !inCommittee {
		return 0
	}

	// at this point we know the validator was inside the sync committee

	totalActiveInc := totalEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT
	totalBaseRewards := GetBaseRewardPerInc(totalEffectiveBalance) * float64(totalActiveInc)
	maxParticipantRewards := totalBaseRewards * float64(SYNC_REWARD_WEIGHT) / float64(WEIGHT_DENOMINATOR) / SLOTS_PER_EPOCH
	participantReward := maxParticipantRewards / float64(SYNC_COMMITTEE_SIZE) // this is the participantReward for a single slot

	return participantReward * SLOTS_PER_EPOCH

}

func (p AltairSpec) GetMaxAttestationReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	maxFlagsReward := float64(0)
	// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

	for i := range p.AttestingBalance {

		// apply formula
		attestingBalanceInc := p.AttestingBalance[i] / EFFECTIVE_BALANCE_INCREMENT

		valIncrements := valEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT
		baseReward := float64(valIncrements * uint64(GetBaseRewardPerInc(totalEffectiveBalance)))
		flagReward := float64(PARTICIPATING_FLAGS_WEIGHT[i]) * baseReward * float64(attestingBalanceInc)
		flagReward = flagReward / ((float64(totalEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT)) * float64(WEIGHT_DENOMINATOR))
		maxFlagsReward += flagReward
	}

	return maxFlagsReward
}

// This method returns the Max Reward the validator could gain in the current
func (p AltairSpec) GetMaxReward(valIdx uint64) (uint64, error) {

	vallEffectiveBalance := p.WrappedState.PrevBState.Altair.Validators[valIdx].EffectiveBalance

	totalEffectiveBalance := p.TotalActiveBalance()

	flagIndexMaxReward := p.GetMaxAttestationReward(valIdx, uint64(vallEffectiveBalance), totalEffectiveBalance)

	syncComMaxReward := p.GetMaxSyncComReward(valIdx, uint64(vallEffectiveBalance), totalEffectiveBalance)

	maxReward := flagIndexMaxReward + syncComMaxReward

	return uint64(maxReward), nil
}

func (p AltairSpec) GetAttestingSlot(valIdx uint64) uint64 {
	return 0
}

func (p AltairSpec) PrevEpochReward(valIdx uint64) int64 {
	return int64(p.WrappedState.BState.Altair.Balances[valIdx] - p.WrappedState.PrevBState.Altair.Balances[valIdx])
}

func (p AltairSpec) CurrentSlot() uint64 {
	return p.WrappedState.BState.Altair.Slot
}

func (p AltairSpec) CurrentEpoch() uint64 {
	return uint64(p.CurrentSlot() / 32)
}

func (p AltairSpec) PrevStateSlot() uint64 {
	return p.WrappedState.PrevBState.Altair.Slot
}

func (p AltairSpec) PrevStateEpoch() uint64 {
	return uint64(p.PrevStateSlot() / 32)
}

// Argument: 0 for source, 1 for target and 2 for head
func (p AltairSpec) GetMissingFlag(flagIndex int) uint64 {
	result := uint64(0)
	for _, item := range p.WrappedState.CorrectFlags {
		if item[flagIndex] {
			result += 1
		}
	}

	return result
}

func (p AltairSpec) GetMissedBlocks() []uint64 {
	return p.WrappedState.MissedBlocks
}

func (p *AltairSpec) TrackMissingBlocks() {
	firstIndex := p.WrappedState.BState.Altair.Slot - SLOTS_PER_EPOCH + 1
	lastIndex := p.WrappedState.BState.Altair.Slot

	for i := firstIndex; i <= lastIndex; i++ {
		if i == 0 {
			continue
		}
		lastItem := p.WrappedState.BState.Altair.BlockRoots[i-1]
		item := p.WrappedState.BState.Altair.BlockRoots[i]
		res := bytes.Compare(lastItem, item)

		if res == 0 {
			// both roots were the same ==> missed block
			p.WrappedState.MissedBlocks = append(p.WrappedState.MissedBlocks, uint64(i))
		}
	}
}
