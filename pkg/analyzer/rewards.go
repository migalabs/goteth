package analyzer

import (
	"fmt"
	"math"

	"github.com/pkg/errors"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

const (
	participationRate   = 0.945
	baseRewardFactor    = 64
	baseRewardsPerEpoch = 4
)

func GetValidatorBalance(bstate *spec.VersionedBeaconState, valIdx uint64) (uint64, error) {
	var balance uint64
	var err error
	switch bstate.Version {
	case spec.DataVersionPhase0:
		if uint64(len(bstate.Phase0.Balances)) < valIdx {
			err = fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, bstate.Phase0.Slot)
		}
		balance = bstate.Phase0.Balances[valIdx]

	case spec.DataVersionAltair:
		if uint64(len(bstate.Altair.Balances)) < valIdx {
			err = fmt.Errorf("altair - validator index %d wasn't activated in slot %d", valIdx, bstate.Phase0.Slot)
		}
		balance = bstate.Altair.Balances[valIdx]

	case spec.DataVersionBellatrix:
		if uint64(len(bstate.Bellatrix.Balances)) < valIdx {
			err = fmt.Errorf("bellatrix - validator index %d wasn't activated in slot %d", valIdx, bstate.Phase0.Slot)
		}
		balance = bstate.Bellatrix.Balances[valIdx]
	default:

	}
	return balance, err
}

// https://kb.beaconcha.in/rewards-and-penalties
// https://consensys.net/blog/codefi/rewards-and-penalties-on-ethereum-20-phase-0/
// TODO: -would be nice to incorporate top the max value wheather there were 2-3 consecutive missed blocks afterwards
func GetMaxReward(valIdx uint64, totValStatus *map[phase0.ValidatorIndex]*api.Validator, totalActiveBalance uint64) (uint64, error) {
	// First iteration just taking 31/8*BaseReward as Max value
	// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )

	idx := phase0.ValidatorIndex(valIdx)

	valStatus, ok := (*totValStatus)[idx]
	if !ok {
		return 0, errors.New("")
	}
	// apply formula
	//baseReward := GetBaseReward(valStatus.Validator.EffectiveBalance, totalActiveBalance)
	maxReward := ((31.0 / 8.0) * participationRate * (float64(uint64(valStatus.Validator.EffectiveBalance)) * baseRewardFactor))
	maxReward = maxReward / (baseRewardsPerEpoch * math.Sqrt(float64(totalActiveBalance)))
	return uint64(maxReward), nil
}

// directly calculated on the MaxReward fucntion
func GetBaseReward(valEffectiveBalance phase0.Gwei, totalActiveBalance uint64) uint64 {
	// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )
	var baseReward uint64

	sqrt := math.Sqrt(float64(totalActiveBalance))

	denom := baseRewardsPerEpoch * sqrt

	bsRewrd := (float64(uint64(valEffectiveBalance)) * baseRewardFactor) / denom

	baseReward = uint64(bsRewrd)
	return baseReward
}
