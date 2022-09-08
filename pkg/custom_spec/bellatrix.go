package custom_spec

import (
	"bytes"
	"fmt"
	"math"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
)

type BellatrixSpec struct {
	WrappedState     ForkStateContent
	AttestingVals    [][]uint64 // one array of validators per participating flag
	AttestingBalance []uint64   // one attesting balance per participation flag
}

func NewBellatrixSpec(nextBstate *spec.VersionedBeaconState, bstate spec.VersionedBeaconState, prevBstate spec.VersionedBeaconState, iApi *http.Service) BellatrixSpec {

	if prevBstate.Bellatrix == nil {
		prevBstate = bstate
	}

	attestingVals := make([][]uint64, 3)

	for i := range attestingVals {
		attestingVals[i] = make([]uint64, len(prevBstate.Bellatrix.Validators))
	}

	BellatrixObj := BellatrixSpec{
		WrappedState: ForkStateContent{
			NextState:        *nextBstate,
			PrevBState:       prevBstate,
			BState:           bstate,
			Api:              iApi,
			EpochStructs:     NewEpochData(iApi, bstate.Bellatrix.Slot),
			PrevEpochStructs: NewEpochData(iApi, prevBstate.Bellatrix.Slot),
			NextEpochStructs: NewEpochData(iApi, nextBstate.Bellatrix.Slot),
		},

		AttestingVals:    attestingVals,
		AttestingBalance: make([]uint64, 3),
	}

	// initialize missing flags arrays
	BellatrixObj.WrappedState.InitializeArrays(uint64(len(bstate.Bellatrix.Validators)))

	// calculate attesting vals only once
	BellatrixObj.CalculatePreviousAttestingVals()
	BellatrixObj.WrappedState.TotalActiveBalance = BellatrixObj.GetTotalActiveEffBalance()
	// leave attestingBalance already calculated
	for i := range BellatrixObj.AttestingBalance {
		BellatrixObj.AttestingBalance[i] = BellatrixObj.ValsEffectiveBalance(BellatrixObj.AttestingVals[i])
	}
	BellatrixObj.TrackMissingBlocks()

	return BellatrixObj
}

// This method will calculate attesting vals to the previous epoch per flag
func (p *BellatrixSpec) CalculatePreviousAttestingVals() {

	flags := []altair.ParticipationFlag{
		altair.TimelySourceFlagIndex,
		altair.TimelyTargetFlagIndex,
		altair.TimelyHeadFlagIndex}

	for participatingFlag := range flags {

		flag := altair.ParticipationFlags(math.Pow(2, float64(participatingFlag)))

		for valIndex, item := range p.WrappedState.BState.Bellatrix.PreviousEpochParticipation {
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
func (p BellatrixSpec) ValsEffectiveBalance(valList []uint64) uint64 {

	combinedEffectiveBalance := uint64(0)

	for valIdx, numAtt := range valList { // loop over validators
		if numAtt > 0 {
			combinedEffectiveBalance += uint64(p.WrappedState.BState.Bellatrix.Validators[valIdx].EffectiveBalance)
		}
	}

	return uint64(combinedEffectiveBalance)
}

// This method returns the Balance of the given validator at the current state
func (p BellatrixSpec) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.WrappedState.BState.Bellatrix.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.WrappedState.BState.Bellatrix.Slot)
		return 0, err
	}
	balance := p.WrappedState.BState.Bellatrix.Balances[valIdx]

	return balance, nil
}

// This method returns the Effective Balance of all active validators
func (p BellatrixSpec) GetTotalActiveEffBalance() uint64 {

	if p.CurrentSlot() < 32 {
		// genesis epoch, validators preactivated with default balance
		return uint64(len(p.WrappedState.BState.Bellatrix.Validators) * EFFECTIVE_BALANCE_INCREMENT * MAX_EFFECTIVE_INCREMENTS)
	}

	all_vals := p.WrappedState.BState.Bellatrix.Validators
	val_array := make([]uint64, len(all_vals))

	for idx := range val_array {
		if IsActive(*all_vals[idx], phase0.Epoch(p.CurrentEpoch())) {
			val_array[idx] += 1
		}

	}

	return p.ValsEffectiveBalance(val_array)
}

func (p BellatrixSpec) GetMaxProposerAttReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	return 0

}

func (p BellatrixSpec) GetMaxProposerSyncReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	return 0

}

// So far we have computed the max sync committee proposer reward for a slot. Since the validator remains in the sync committee for the full epoch, we multiply the reward for the 32 slots in the epoch.
// TODO: Tracking missing blocks in the epoch would help us have an even more accurate theoretical sync proposer max reward per epoch.
func (p BellatrixSpec) GetMaxSyncComReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	inCommittee := false

	valPubKey := p.WrappedState.BState.Bellatrix.Validators[valIdx].PublicKey

	syncCommitteePubKeys := p.WrappedState.BState.Bellatrix.NextSyncCommittee

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

func (p BellatrixSpec) GetMaxAttestationReward(valIdx uint64, baseReward float64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	maxFlagsReward := float64(0)
	// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR

	for i := range p.AttestingBalance {

		// apply formula
		attestingBalanceInc := p.AttestingBalance[i] / EFFECTIVE_BALANCE_INCREMENT

		flagReward := float64(PARTICIPATING_FLAGS_WEIGHT[i]) * baseReward * float64(attestingBalanceInc)
		flagReward = flagReward / ((float64(totalEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT)) * float64(WEIGHT_DENOMINATOR))
		maxFlagsReward += flagReward
	}

	return maxFlagsReward
}

// This method returns the Max Reward the validator could gain in the current
func (p BellatrixSpec) GetMaxReward(valIdx uint64) (ValidatorSepRewards, error) {

	valEffectiveBalance := float64(p.WrappedState.PrevBState.Bellatrix.Validators[valIdx].EffectiveBalance)
	totalEffectiveBalance := p.WrappedState.TotalActiveBalance

	valIncrements := valEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT
	baseReward := float64(valIncrements * float64(GetBaseRewardPerInc(totalEffectiveBalance)))

	flagIndexMaxReward := p.GetMaxAttestationReward(valIdx, baseReward, uint64(valEffectiveBalance), totalEffectiveBalance)

	// syncComMaxReward := p.GetMaxSyncComReward(valIdx, uint64(valEffectiveBalance), totalEffectiveBalance)

	maxReward := flagIndexMaxReward //+ syncComMaxReward

	result := ValidatorSepRewards{
		Attestation:     0,
		InclusionDelay:  0,
		FlagIndex:       flagIndexMaxReward,
		SyncCommittee:   0,
		MaxReward:       maxReward,
		BaseReward:      baseReward,
		ProposerSlot:    -1,
		InSyncCommittee: false,
	}
	return result, nil

}

func (p BellatrixSpec) GetAttestingSlot(valIdx uint64) uint64 {
	return 0
}

func (p BellatrixSpec) PrevEpochReward(valIdx uint64) int64 {

	return int64(p.WrappedState.BState.Bellatrix.Balances[valIdx] - p.WrappedState.PrevBState.Bellatrix.Balances[valIdx])
}

func (p BellatrixSpec) CurrentSlot() uint64 {
	return p.WrappedState.BState.Bellatrix.Slot
}

func (p BellatrixSpec) CurrentEpoch() uint64 {
	return uint64(p.CurrentSlot() / 32)
}

func (p BellatrixSpec) PrevStateSlot() uint64 {
	return p.WrappedState.PrevBState.Bellatrix.Slot
}

func (p BellatrixSpec) PrevStateEpoch() uint64 {
	return uint64(p.PrevStateSlot() / 32)
}

// Argument: 0 for source, 1 for target and 2 for head
func (p BellatrixSpec) GetMissingFlag(flagIndex int) uint64 {
	result := uint64(0)
	for _, item := range p.WrappedState.CorrectFlags[flagIndex] {
		if !item {
			result += 1
		}
	}

	return result
}

func (p BellatrixSpec) GetMissedBlocks() []uint64 {
	return p.WrappedState.MissedBlocks
}

func (p *BellatrixSpec) TrackMissingBlocks() {
	firstIndex := (p.WrappedState.BState.Bellatrix.Slot - SLOTS_PER_EPOCH + 1) % SLOTS_PER_HISTORICAL_ROOT
	lastIndex := (p.WrappedState.BState.Bellatrix.Slot) % SLOTS_PER_HISTORICAL_ROOT

	for i := firstIndex; i <= lastIndex; i++ {
		if i == 0 {
			continue
		}
		lastItem := p.WrappedState.BState.Bellatrix.BlockRoots[i-1]
		item := p.WrappedState.BState.Bellatrix.BlockRoots[i]
		res := bytes.Compare(lastItem, item)

		if res == 0 {
			// both roots were the same ==> missed block
			slot := i - firstIndex + p.WrappedState.BState.Bellatrix.Slot - SLOTS_PER_EPOCH + 1
			p.WrappedState.MissedBlocks = append(p.WrappedState.MissedBlocks, uint64(slot))
		}
	}
}

func (p BellatrixSpec) GetTotalActiveBalance() uint64 {
	all_vals := p.WrappedState.BState.Bellatrix.Validators
	totalBalance := uint64(0)

	for idx := range all_vals {
		if IsActive(*all_vals[idx], phase0.Epoch(p.CurrentEpoch())) {
			totalBalance += p.WrappedState.BState.Bellatrix.Balances[idx]
		}

	}
	return totalBalance
}

func (p BellatrixSpec) GetAttestingValNum() uint64 {
	result := 0

	for i := 0; i < len(p.AttestingVals[altair.TimelySourceFlagIndex]); i++ {
		sourceFlag := p.AttestingVals[altair.TimelySourceFlagIndex][i]
		targetFlag := p.AttestingVals[altair.TimelyTargetFlagIndex][i]
		headFlag := p.AttestingVals[altair.TimelyHeadFlagIndex][i]

		// if any of the flags is 1, then we consider attest
		if (sourceFlag + targetFlag + headFlag) > 0 {
			result += 1
		}
	}

	return uint64(result)
}

func (p BellatrixSpec) GetAttNum() uint64 {

	return 0
}

func (p BellatrixSpec) GetAttSlot(valIdx uint64) int64 {

	return int64(p.WrappedState.PrevEpochStructs.ValidatorAttSlot[valIdx])
}

func (p BellatrixSpec) GetAttInclusionSlot(valIdx uint64) int64 {

	return -1
}

func (p BellatrixSpec) GetBaseReward(valIdx uint64) float64 {
	effectiveBalanceInc := p.WrappedState.BState.Bellatrix.Validators[valIdx].EffectiveBalance / EFFECTIVE_BALANCE_INCREMENT
	totalEffBalance := p.WrappedState.TotalActiveBalance
	return GetBaseRewardPerInc(totalEffBalance) * float64(effectiveBalanceInc)
}

func (p BellatrixSpec) GetNumVals() uint64 {
	result := uint64(0)

	for _, val := range p.WrappedState.BState.Bellatrix.Validators {
		if IsActive(*val, phase0.Epoch(p.CurrentEpoch())) {
			result += 1
		}

	}
	return result
}

func (p BellatrixSpec) GetPrevValList() []uint64 {
	result := []uint64{}

	for i, item := range p.WrappedState.PrevBState.Bellatrix.Validators {
		epoch := utils.GetEpochFromSlot(p.WrappedState.PrevBState.Bellatrix.Slot)
		if IsActive(*item, phase0.Epoch(epoch)) {
			result = append(result, uint64(i))
		}
	}
	return result
}

func (p BellatrixSpec) MissingFlags(valIdx uint64) []bool {
	result := []bool{false, false, false}

	for i, item := range p.WrappedState.CorrectFlags {
		result[i] = !item[valIdx]

	}
	return result
}
