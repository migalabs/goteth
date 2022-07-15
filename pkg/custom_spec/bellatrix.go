package custom_spec

import (
	"errors"
	"fmt"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
	"github.com/prysmaticlabs/go-bitfield"
)

type BellatrixSpec struct {
	BState      spec.VersionedBeaconState
	Committees  map[string]bitfield.Bitlist
	DoubleVotes uint64
}

func NewBellatrixSpec(bstate *spec.VersionedBeaconState) BellatrixSpec {
	bellatrixObj := BellatrixSpec{
		BState:      *bstate,
		Committees:  make(map[string]bitfield.Bitlist),
		DoubleVotes: 0,
	}
	bellatrixObj.PreviousEpochAttestations()

	return bellatrixObj
}

func (p BellatrixSpec) CurrentSlot() uint64 {
	return p.BState.Bellatrix.Slot
}

func (p BellatrixSpec) CurrentEpoch() uint64 {
	return uint64(p.CurrentSlot() / 32)
}

func (p BellatrixSpec) PreviousEpochAttestations() uint64 {
	numOfAttestations := 0
	attestationsPerVal := p.BState.Bellatrix.PreviousEpochParticipation

	for _, item := range attestationsPerVal {
		// Here we have one item per validator
		// This is an 8-bit string, where the bit 0 is the source
		// If it is set, we consider there was a vote from this validator
		if utils.IsBitSet(uint8(item), 0) {
			numOfAttestations++
		}
	}

	return uint64(numOfAttestations)
}

func (p BellatrixSpec) PreviousEpochMissedAttestations() uint64 {
	numOfMissedAttestations := 0
	attestationsPerVal := p.BState.Bellatrix.PreviousEpochParticipation

	for _, item := range attestationsPerVal {
		// Here we have one item per validator
		if !utils.IsBitSet(uint8(item), 0) {
			numOfMissedAttestations++
		}
	}

	return uint64(numOfMissedAttestations)

}

func (p BellatrixSpec) PreviousEpochValNum() uint64 {
	vals := p.BState.Bellatrix.Validators
	totalAttestingVals := 0

	for _, item := range vals {
		// validator must be either active, exiting or slashed
		if item.ActivationEligibilityEpoch < phase0.Epoch(p.CurrentEpoch()) &&
			item.ExitEpoch > phase0.Epoch(p.CurrentEpoch()) {
			totalAttestingVals += 1
		}
	}
	return uint64(totalAttestingVals)
}

func (p BellatrixSpec) GetDoubleVotes() uint64 {
	return p.DoubleVotes
}

func (p BellatrixSpec) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.BState.Bellatrix.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.BState.Bellatrix.Slot)
		return 0, err
	}
	balance := p.BState.Bellatrix.Balances[valIdx]

	return balance, nil
}

func (p BellatrixSpec) GetMaxSyncComReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {
	inCommittee := false
	for _, val := range p.BState.Bellatrix.CurrentSyncCommittee.Pubkeys {
		if val == valPubKey {
			inCommittee = true
		}
	}

	if inCommittee {
		increments := totalEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT
		return SYNC_COMMITTEE_FACTOR * float64(increments) * GetBaseReward(valEffectiveBalance, totalEffectiveBalance) * EPOCH
	}

	return 0

}

func (p BellatrixSpec) GetMaxAttestationReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	increments := valEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT

	return ATTESTATION_FACTOR * float64(increments) * GetBaseReward(valEffectiveBalance, totalEffectiveBalance)
}

func (p BellatrixSpec) GetMaxReward(valIdx uint64, totValStatus *map[phase0.ValidatorIndex]*api.Validator, totalEffectiveBalance uint64) (uint64, error) {
	valStatus, ok := (*totValStatus)[phase0.ValidatorIndex(valIdx)]

	if !ok {
		return 0, errors.New("could not get validator effective balance")
	}

	valEffectiveBalance := valStatus.Validator.EffectiveBalance

	maxAttReward := p.GetMaxAttestationReward(valIdx, uint64(valEffectiveBalance), totalEffectiveBalance)
	maxSyncReward := p.GetMaxSyncComReward(valIdx, valStatus.Validator.PublicKey, uint64(valEffectiveBalance), totalEffectiveBalance)

	maxReward := maxAttReward + maxSyncReward

	return uint64(maxReward), nil
}
