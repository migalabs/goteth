package custom_spec

import (
	"fmt"
	"strconv"

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
	PreviousCommittees  map[string]bitfield.Bitlist
	PreviousDoubleVotes uint64
	CurrentCommittees   map[string]bitfield.Bitlist
	CurrentDoubleVotes  uint64
}

func NewPhase0Spec(bstate *spec.VersionedBeaconState) Phase0Spec {
	phase0Obj := Phase0Spec{
		BState:              *bstate,
		PreviousCommittees:  make(map[string]bitfield.Bitlist),
		PreviousDoubleVotes: 0,
		CurrentCommittees:   make(map[string]bitfield.Bitlist),
		CurrentDoubleVotes:  0,
	}
	phase0Obj.CalculatePreviousEpochAggregations()

	phase0Obj.CalculateCurrentEpochAggregations()
	numOfAttCurrentEpoch := phase0Obj.CurrentEpochAttestations()
	fmt.Println(numOfAttCurrentEpoch)

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
	totalAttestingVals := 0

	previousAttestatons := p.PreviousEpochAggregations()
	slashings := p.BState.Phase0.Slashings
	slashing_sum := 0

	for _, val := range slashings {
		slashing_sum += int(val)
	}
	fmt.Println(slashing_sum)

	// proposers := p.BState.Phase0
	doubleVotes := 0
	vals := p.BState.Phase0.Validators

	for _, item := range vals {
		// validator must be either active, exiting or slashed
		if item.ActivationEligibilityEpoch < phase0.Epoch(p.CurrentEpoch()) &&
			item.ExitEpoch > phase0.Epoch(p.CurrentEpoch()) {
			totalAttestingVals += 1
		}
	}

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

func (p Phase0Spec) PreviousEpochAggregations() []*phase0.PendingAttestation {
	return p.BState.Phase0.PreviousEpochAttestations
}

func (p Phase0Spec) PreviousEpochValNum() uint64 {

	numOfBits := 0 // it should be equal to the number of validators

	for _, val := range p.PreviousCommittees {

		numOfBits += int(val.Len())
	}

	return uint64(numOfBits)
}

func (p *Phase0Spec) CalculateCurrentEpochAggregations() {

	totalAttestingVals := 0

	currentAttestatons := p.BState.Phase0.CurrentEpochAttestations
	doubleVotes := 0
	vals := p.BState.Phase0.Validators

	for _, item := range vals {
		// validator must be either active, exiting or slashed
		if item.ActivationEligibilityEpoch <= phase0.Epoch(p.CurrentEpoch()) &&
			item.ExitEpoch > phase0.Epoch(p.CurrentEpoch()) {
			totalAttestingVals += 1
		}
	}

	for _, item := range currentAttestatons {
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
		p.CurrentCommittees[mapKey] = resultBits
		// if we do the And at the same committee we can catch the double votes
		// doing that the number of votes is less than the number of validators

	}

	p.CurrentDoubleVotes = uint64(doubleVotes)

}

func (p Phase0Spec) CurrentEpochAttestations() uint64 {

	numOf1Bits := 0 // it should be equal to the number of validators that attested

	for _, val := range p.CurrentCommittees {

		numOf1Bits += int(val.Count())
	}

	return uint64(numOf1Bits)
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
