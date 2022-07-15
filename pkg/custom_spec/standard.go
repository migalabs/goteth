package custom_spec

import (
	"fmt"
	"math"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

const (
	BASE_REWARD_FACTOR          = 64.0
	BASE_REWARD_PER_EPOCH       = 4.0
	EFFECTIVE_BALANCE_INCREMENT = 1000000000
	ATTESTATION_FACTOR          = 0.84375 // 14+26+14/64
	PROPOSER_WEIGHT             = 0.125
	SYNC_COMMITTEE_FACTOR       = 0.00000190734
	EPOCH                       = 32
	SHUFFLE_ROUND_COUNT         = uint64(90)
	// participationRate   = 0.945 // about to calculate participation rate
)

// directly calculated on the MaxReward fucntion
func GetBaseReward(valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {
	// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )
	var baseReward float64

	sqrt := uint64(math.Sqrt(float64(totalEffectiveBalance)))

	denom := float64(BASE_REWARD_PER_EPOCH * sqrt)

	num := (float64(valEffectiveBalance) * BASE_REWARD_FACTOR)
	baseReward = num / denom

	return baseReward
}

func GetBaseRewardPerInc(totalEffectiveBalance uint64) float64 {
	// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )
	var baseReward float64

	sqrt := uint64(math.Sqrt(float64(totalEffectiveBalance)))

	num := EFFECTIVE_BALANCE_INCREMENT * BASE_REWARD_FACTOR
	baseReward = num / float64(sqrt)

	return baseReward
}

type CustomBeaconState interface {
	PreviousEpochAttestations() uint64
	PreviousEpochValNum() uint64 // those activated before current Epoch
	CurrentEpoch() uint64
	CurrentSlot() uint64
	GetDoubleVotes() uint64
	Balance(valIdx uint64) (uint64, error)
	GetMaxReward(valIdx uint64, validators *map[phase0.ValidatorIndex]*api.Validator, totalEffectiveBalance uint64) (uint64, error)
}

func BStateByForkVersion(bstate *spec.VersionedBeaconState, prevBstate spec.VersionedBeaconState, iApi *http.Service) (CustomBeaconState, error) {
	switch bstate.Version {

	case spec.DataVersionPhase0:
		return NewPhase0Spec(bstate, prevBstate, iApi), nil

	case spec.DataVersionAltair:
		return NewAltairSpec(bstate), nil

	case spec.DataVersionBellatrix:
		return NewBellatrixSpec(bstate), nil
	default:
		return nil, fmt.Errorf("could not figure out the Beacon State Fork Version")
	}
}
