package custom_spec

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "custom_spec",
	)
)

type Phase0Spec struct {
	BState              spec.VersionedBeaconState
	PrevBState          spec.VersionedBeaconState
	PreviousCommittees  map[string]bitfield.Bitlist
	PreviousDoubleVotes uint64
	CurrentCommittees   map[string]bitfield.Bitlist
	CurrentDoubleVotes  uint64
	TotalActiveVals     uint64
	Api                 *http.Service
}

func NewPhase0Spec(bstate *spec.VersionedBeaconState, prevBstate spec.VersionedBeaconState, iApi *http.Service) Phase0Spec {
	// func NewPhase0Spec(bstate *spec.VersionedBeaconState, iCli *clientapi.APIClient) Phase0Spec {
	phase0Obj := Phase0Spec{
		BState:              *bstate,
		PrevBState:          prevBstate,
		PreviousCommittees:  make(map[string]bitfield.Bitlist),
		PreviousDoubleVotes: 0,
		CurrentCommittees:   make(map[string]bitfield.Bitlist),
		CurrentDoubleVotes:  0,
		TotalActiveVals:     0,
		Api:                 iApi,
	}
	phase0Obj.CalculatePreviousEpochAggregations()

	return phase0Obj

}

func (p Phase0Spec) CurrentSlot() uint64 {
	return p.BState.Phase0.Slot
}

func (p Phase0Spec) CurrentEpoch() uint64 {
	return uint64(p.CurrentSlot() / 32)
}

func (p *Phase0Spec) CalculatePreviousEpochAggregations() {

	totalAttPreviousEpoch := 0
	totalActiveVals := 0

	previousAttestatons := p.PreviousEpochAggregations()
	slashings := p.BState.Phase0.Slashings
	slashing_sum := 0

	for _, val := range slashings {
		slashing_sum += int(val)
	}

	doubleVotes := 0
	vals := p.BState.Phase0.Validators

	for _, item := range vals {
		// validator must be either active, exiting or slashed
		if item.ActivationEligibilityEpoch < phase0.Epoch(p.CurrentEpoch()) &&
			item.ExitEpoch > phase0.Epoch(p.CurrentEpoch()) {
			totalActiveVals += 1
		}
	}
	p.TotalActiveVals = uint64(totalActiveVals)

	for _, item := range previousAttestatons {
		slot := item.Data.Slot // Block that is being attested, not included
		committeeIndex := item.Data.Index
		mapKey := strconv.Itoa(int(slot)) + "_" + strconv.Itoa(int(committeeIndex))

		resultBits := bitfield.NewBitlist(0)

		if val, ok := p.PreviousCommittees[mapKey]; ok {
			// the committeeIndex for the given slot already had an aggregation
			// TODO: check error
			allZero, err := val.And(item.AggregationBits) // if the same position of the aggregations has a 1 in the same position, there was a double vote
			if err != nil {
				log.Debugf(err.Error(), "error while processing aggregation bits of committee %d in slot %d", slot, committeeIndex)
			}
			if allZero.Count() > 0 {
				// there was a double vote
				doubleVotes += int(allZero.Count())
			}

			resultBitstmp, err := val.Or(item.AggregationBits) // to join all aggregation we do Or operation

			if err == nil {
				resultBits = resultBitstmp
			} else {
				log.Debugf(err.Error(), "error while processing aggregation bits of committee %d in slot %d", slot, committeeIndex)
			}

		} else {
			// we had not received any aggregation for this committeeIndex at the given slot
			resultBits = item.AggregationBits
		}
		p.PreviousCommittees[mapKey] = resultBits
		attPreviousEpoch := int(item.AggregationBits.Count())
		totalAttPreviousEpoch += attPreviousEpoch // we are counting bits set to 1 aggregation by aggregation
		// if we do the And at the same committee we can catch the double votes
		// doing that the number of votes is less than the number of validators

	}
	p.PreviousDoubleVotes = uint64(doubleVotes)
}

func (p Phase0Spec) PreviousEpochAttestations() uint64 {

	numOf1Bits := 0 // it should be equal to the number of validators that attested

	for _, val := range p.PreviousCommittees {

		numOf1Bits += int(val.Count())
	}

	return uint64(numOf1Bits)
}

func (p Phase0Spec) PreviousEpochAttestedBalance() uint64 {

	attestingVals := make([]uint64, 0)
	// all committees of the whole epoch at the given slot containing validatorIndexes
	epochCommittees, err := p.Api.BeaconCommittees(context.Background(), strconv.Itoa(int(p.PrevBState.Phase0.Slot)))

	if err != nil {
		log.Errorf(err.Error())
	}
	// loop over committees of the previous epoch
	for index, val := range p.PreviousCommittees {
		splitted_committee := strings.Split(index, "_")
		slot, err := strconv.Atoi(splitted_committee[0])
		if err != nil {
			log.Errorf(err.Error())
		}
		committeIndex, err := strconv.Atoi(splitted_committee[1])

		if err != nil {
			log.Errorf(err.Error())
		}

		slotCommittees := make([]api.BeaconCommittee, 0)

		for _, singleCommittee := range epochCommittees {
			if int(singleCommittee.Slot) == slot {
				slotCommittees = append(slotCommittees, *singleCommittee)
			}
		}

		if committeIndex >= len(slotCommittees) {
			fmt.Println(committeIndex)
		}

		committee := slotCommittees[committeIndex]
		attestingComIndices := val.BitIndices()

		for _, index := range attestingComIndices {
			if index >= len(committee.Validators) {
				fmt.Println(index)
			}
			attestingVals = append(attestingVals, uint64(committee.Validators[index]))
		}
	}

	previousAttestingBalance := 0

	for _, valIdx := range attestingVals {
		newEffectiveBalance := math.Min(float64(p.PrevBState.Phase0.Balances[valIdx]), 32*EFFECTIVE_BALANCE_INCREMENT)
		previousAttestingBalance += int(newEffectiveBalance)
	}

	return uint64(previousAttestingBalance)
}

func (p Phase0Spec) PreviousEpochAggregations() []*phase0.PendingAttestation {
	return p.BState.Phase0.PreviousEpochAttestations
}

func (p Phase0Spec) PreviousEpochValNum() uint64 {

	return p.TotalActiveVals
}

func (p Phase0Spec) PreviousEpochActiveValBalance() uint64 {

	activeEffectiveBalance := 0
	vals := p.BState.Phase0.Validators

	for _, item := range vals {
		// validator must be either active, exiting or slashed
		if item.ActivationEligibilityEpoch < phase0.Epoch(p.CurrentEpoch()) &&
			item.ExitEpoch > phase0.Epoch(p.CurrentEpoch()) {
			activeEffectiveBalance += int(item.EffectiveBalance)
		}
	}
	return uint64(activeEffectiveBalance)
}

func (p Phase0Spec) GetDoubleVotes() uint64 {
	return p.PreviousDoubleVotes
}

func (p Phase0Spec) GetCurrentDoubleVotes() uint64 {
	return p.CurrentDoubleVotes
}

func (p Phase0Spec) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.BState.Phase0.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.BState.Phase0.Slot)
		return 0, err
	}
	balance := p.BState.Phase0.Balances[valIdx]

	return balance, nil
}

func (p Phase0Spec) GetMaxProposerReward(valIdx uint64, valEffectiveBalance uint64, totalEffectiveBalance uint64) float64 {

	return 0
}

func (p Phase0Spec) GetMaxReward(valIdx uint64, totValStatus *map[phase0.ValidatorIndex]*api.Validator, totalEffectiveBalance uint64) (uint64, error) {
	previousAttestedBalance := p.PreviousEpochAttestedBalance()

	previousEpochActiveBalance := p.PreviousEpochActiveValBalance()

	participationRate := float64(float64(previousAttestedBalance) / float64(previousEpochActiveBalance))

	// First iteration just taking 31/8*BaseReward as Max value
	// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )

	idx := phase0.ValidatorIndex(valIdx)

	valStatus, ok := (*totValStatus)[idx]
	if !ok {
		return 0, errors.New("")
	}
	// apply formula
	baseReward := GetBaseReward(uint64(valStatus.Validator.EffectiveBalance), totalEffectiveBalance)
	voteReward := 3.0 * baseReward * participationRate
	inclusionDelay := baseReward * 7.0 / 8.0

	maxReward := voteReward + inclusionDelay

	return uint64(maxReward), nil
}
