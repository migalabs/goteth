package custom_spec

import (
	"fmt"
	"math"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/prysmaticlabs/go-bitfield"
)

var (
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
	PrevBState    spec.VersionedBeaconState
	BState        spec.VersionedBeaconState
	Committees    map[string]bitfield.Bitlist
	DoubleVotes   uint64
	Api           *http.Service
	EpochStructs  EpochData
	AttestingVals [][]uint64 // one array of validators per participating flag
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
		PrevBState:    prevBstate,
		BState:        *bstate,
		Committees:    make(map[string]bitfield.Bitlist),
		DoubleVotes:   0,
		Api:           iApi,
		EpochStructs:  NewEpochData(iApi, bstate.Altair.Slot),
		AttestingVals: attestingVals,
	}
	altairObj.CalculatePreviousAttestingVals()

	return altairObj
}

func (p AltairSpec) CurrentSlot() uint64 {
	return p.BState.Altair.Slot
}

func (p AltairSpec) CurrentEpoch() uint64 {
	return uint64(p.CurrentSlot() / 32)
}

func (p AltairSpec) PrevStateSlot() uint64 {
	return p.PrevBState.Altair.Slot
}

func (p AltairSpec) PrevStateEpoch() uint64 {
	return uint64(p.PrevStateSlot() / 32)
}

func (p *AltairSpec) CalculatePreviousAttestingVals() {

	flags := []altair.ParticipationFlag{
		altair.TimelySourceFlagIndex,
		altair.TimelyTargetFlagIndex,
		altair.TimelyHeadFlagIndex}

	for participatingFlag := range flags {

		flag := altair.ParticipationFlags(math.Pow(2, float64(participatingFlag)))

		for valIndex, item := range p.BState.Altair.PreviousEpochParticipation {
			// Here we have one item per validator
			// This is an 8-bit string, where the bit 0 is the source
			// If it is set, we consider there was a vote from this validator
			// if utils.IsBitSet(uint8(item), 0) {
			// }

			if (item & flag) == flag {
				// The attestation has a timely source value, therefore we consider it attest
				p.AttestingVals[participatingFlag][valIndex] += uint64(1)
			}
		}
	}
}

// the length of the valList = number of validators
// each position represents a valIdx
// if the item has a number > 0, count it
func (p AltairSpec) ValsBalance(valList []uint64) uint64 {

	attestingBalance := uint64(0)
	countedVals := 0 // for testing

	for valIdx, numAtt := range valList { // loop over validators
		if numAtt > 0 {
			countedVals += 1 // for testing
			attestingBalance += uint64(p.BState.Altair.Validators[valIdx].EffectiveBalance)
		}
	}

	return uint64(attestingBalance)
}

func (p AltairSpec) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.BState.Altair.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.BState.Altair.Slot)
		return 0, err
	}
	balance := p.BState.Altair.Balances[valIdx]

	return balance, nil
}

func (p AltairSpec) TotalActiveBalance() uint64 {

	if p.CurrentSlot() < 32 {
		// genesis epoch, validators preactivated
		return uint64(len(p.BState.Altair.Validators) * EFFECTIVE_BALANCE_INCREMENT * MAX_EFFECTIVE_INCREMENTS)
	}

	all_vals := p.BState.Altair.Validators
	val_array := make([]uint64, len(all_vals))

	for idx := range val_array {
		if IsActive(*all_vals[idx], phase0.Epoch(p.CurrentEpoch())) {
			val_array[idx] += 1
		}

	}

	return p.ValsBalance(val_array)
}

func (p AltairSpec) GetMaxProposerAttReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	return 0

}

func (p AltairSpec) GetMaxProposerSyncReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	return 0

}

// TODO: check missing blocks
// If there are missing blocks then these rewards would be less (not multiply by 32)
func (p AltairSpec) GetMaxSyncComReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	inCommittee := false

	valPubKey := p.PrevBState.Altair.Validators[valIdx].PublicKey

	syncCommitteePubKeys := p.PrevBState.Altair.CurrentSyncCommittee

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
	participantReward := maxParticipantRewards / float64(SYNC_COMMITTEE_SIZE)

	// this is the participantReward for a single slot
	// However, if the validator was in the current sync committee, it gained rewards over the whole epoch

	return participantReward * SLOTS_PER_EPOCH

}

func (p AltairSpec) GetMaxAttestationReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	maxReward := float64(0)
	// the maxReward would be each flag_index_weight * base_reward * (attesting_balance_inc / total_active_balance_inc) / WEIGHT_DENOMINATOR
	// ==> flag_factor = (14+26+14)/64 = 0.84375

	for i := range p.AttestingVals {
		attestingBalanceInc := p.ValsBalance(p.AttestingVals[i]) / EFFECTIVE_BALANCE_INCREMENT

		valIncrements := valEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT
		baseReward := float64(valIncrements * uint64(GetBaseRewardPerInc(totalEffectiveBalance)))
		flagReward := float64(PARTICIPATING_FLAGS_WEIGHT[i]) * baseReward * float64(attestingBalanceInc)
		flagReward = flagReward / ((float64(totalEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT)) * float64(WEIGHT_DENOMINATOR))
		maxReward += flagReward
	}

	return maxReward
}

func (p AltairSpec) GetMaxReward(valIdx uint64) (uint64, error) {

	vallEffectiveBalance := p.PrevBState.Altair.Validators[valIdx].EffectiveBalance

	totalEffectiveBalance := p.TotalActiveBalance()

	flagIndexMaxReward := p.GetMaxAttestationReward(valIdx, uint64(vallEffectiveBalance), totalEffectiveBalance)

	syncComMaxReward := p.GetMaxSyncComReward(valIdx, uint64(vallEffectiveBalance), totalEffectiveBalance)

	maxReward := flagIndexMaxReward + syncComMaxReward

	return uint64(maxReward), nil
}

func (p AltairSpec) GetAttestingSlot(valIdx uint64) uint64 {
	return 0
}

func (p AltairSpec) PrevEpochReward(valIdx uint64) uint64 {
	return p.BState.Altair.Balances[valIdx] - p.PrevBState.Altair.Balances[valIdx]
}
