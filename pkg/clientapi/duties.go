package clientapi

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s *APIClient) NewEpochData(slot phase0.Slot) spec.EpochDuties {

	epochCommittees, err := s.Api.BeaconCommittees(s.ctx, &api.BeaconCommitteesOpts{
		State: fmt.Sprintf("%d", slot),
	})

	if err != nil {
		log.Errorf("could not get beacon committees at slot %d: %s", slot, err)
	}

	validatorsAttSlot := make(map[phase0.ValidatorIndex]phase0.Slot) // each validator, when it had to attest
	validatorsPerSlot := make(map[phase0.Slot][]phase0.ValidatorIndex)

	if epochCommittees != nil {
		for _, committee := range epochCommittees.Data {
			for _, valID := range committee.Validators {
				validatorsAttSlot[valID] = committee.Slot

				if val, ok := validatorsPerSlot[committee.Slot]; ok {
					// the slot exists in the map
					validatorsPerSlot[committee.Slot] = append(val, valID)
				} else {
					// the slot does not exist, create
					validatorsPerSlot[committee.Slot] = []phase0.ValidatorIndex{valID}
				}
			}
		}
	}

	proposerDuties, err := s.Api.ProposerDuties(s.ctx, &api.ProposerDutiesOpts{
		Epoch: phase0.Epoch(slot / spec.SlotsPerEpoch),
	})

	if err != nil {
		log.Errorf("could not get proposer duties at slot %d: %s", slot, err)
	}

	result := spec.EpochDuties{
		ValidatorAttSlot: validatorsAttSlot,
	}
	if proposerDuties != nil {
		result.ProposerDuties = proposerDuties.Data
	}
	if epochCommittees != nil {
		result.BeaconCommittees = epochCommittees.Data
	}

	return result
}
