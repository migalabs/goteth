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

type AltairSpec struct {
	BState      spec.VersionedBeaconState
	Committees  map[string]bitfield.Bitlist
	DoubleVotes uint64
}

func NewAltairSpec(bstate *spec.VersionedBeaconState) AltairSpec {
	altairObj := AltairSpec{
		BState:      *bstate,
		Committees:  make(map[string]bitfield.Bitlist),
		DoubleVotes: 0,
	}
	altairObj.PreviousEpochAttestations()

	return altairObj
}

func (p AltairSpec) CurrentSlot() uint64 {
	return p.BState.Altair.Slot
}

func (p AltairSpec) CurrentEpoch() uint64 {
	return uint64(p.CurrentSlot() / 32)
}

func (p AltairSpec) PreviousEpochAttestations() uint64 {
	numOfAttestations := 0
	attestationsPerVal := p.BState.Altair.PreviousEpochParticipation

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

func (p AltairSpec) GetValidatorBalance(valIdx uint64) uint64 {
	return p.BState.Altair.Balances[valIdx]

}

func (p AltairSpec) PreviousEpochMissedAttestations() uint64 {
	numOfMissedAttestations := 0
	attestationsPerVal := p.BState.Altair.PreviousEpochParticipation

	for _, item := range attestationsPerVal {
		// Here we have one item per validator
		if !utils.IsBitSet(uint8(item), 0) {
			numOfMissedAttestations++
		}
	}

	return uint64(numOfMissedAttestations)

}

func (p AltairSpec) PreviousEpochValNum() uint64 {
	vals := p.BState.Altair.Validators
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

func (p AltairSpec) GetDoubleVotes() uint64 {
	return p.DoubleVotes
}

func (p AltairSpec) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.BState.Altair.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.BState.Altair.Slot)
		return 0, err
	}
	balance := p.BState.Altair.Balances[valIdx]

	return balance, nil
}

func (p AltairSpec) GetMaxSyncComReward(valIdx uint64, valPubKey phase0.BLSPubKey, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {
	inCommittee := false
	for _, val := range p.BState.Altair.CurrentSyncCommittee.Pubkeys {
		if val == valPubKey {
			inCommittee = true
		}
	}

	if inCommittee {
		increments := float64(totalEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT)
		return SYNC_COMMITTEE_FACTOR * increments * GetBaseRewardPerInc(totalEffectiveBalance) * EPOCH
	}

	return 0

}

func (p AltairSpec) GetMaxAttestationReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	increments := float64(valEffectiveBalance / EFFECTIVE_BALANCE_INCREMENT)

	return ATTESTATION_FACTOR * increments * GetBaseRewardPerInc(totalEffectiveBalance)
}

func (p AltairSpec) GetMaxReward(valIdx uint64, totValStatus *map[phase0.ValidatorIndex]*api.Validator, totalEffectiveBalance uint64) (uint64, error) {
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
