package clientapi

import (
	"errors"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

func (s *APIClient) NewEpochData(slot phase0.Slot) (spec.EpochDuties, error) {

	// Beacon committees are required to resolve the attesting indices of every
	// attestation. If the fetch fails we must surface the error so the caller can
	// retry the whole state download instead of building a state with empty
	// committees, which later nil-panics in (Electra)Metrics.GetAttestingIndices
	// (see https://github.com/migalabs/goteth/issues/271).
	epochCommittees, err := s.requestBeaconCommittees(slot)
	if err != nil {
		return spec.EpochDuties{}, fmt.Errorf("could not get beacon committees at slot %d: %w", slot, err)
	}

	validatorsAttSlot := make(map[phase0.ValidatorIndex]phase0.Slot) // each validator, when it had to attest
	validatorsPerSlot := make(map[phase0.Slot][]phase0.ValidatorIndex)

	for _, committee := range epochCommittees {
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

	result := spec.EpochDuties{
		ValidatorAttSlot: validatorsAttSlot,
		BeaconCommittees: epochCommittees,
	}

	// Proposer duties are not needed to resolve attesting indices, so a failure
	// here is logged but not fatal (preserves the previous lenient behavior).
	proposerDuties, err := s.requestProposerDuties(phase0.Epoch(slot / spec.SlotsPerEpoch))
	if err != nil {
		log.Errorf("could not get proposer duties at slot %d: %s", slot, err)
	} else {
		result.ProposerDuties = proposerDuties
	}

	return result, nil
}

// requestBeaconCommittees fetches the beacon committees for the epoch containing
// slot, retrying transient failures (e.g. "unexpected EOF" from a flaky beacon
// node or proxy) up to maxRetries with incremental backoff.
func (s *APIClient) requestBeaconCommittees(slot phase0.Slot) ([]*apiv1.BeaconCommittee, error) {
	err := errors.New("first attempt")
	var resp *api.Response[[]*apiv1.BeaconCommittee]

	attempts := 0
	for err != nil && attempts < s.maxRetries {
		resp, err = s.Api.BeaconCommittees(s.ctx, &api.BeaconCommitteesOpts{
			State: fmt.Sprintf("%d", slot),
		})
		if err != nil {
			log.Warnf("retrying beacon committees request at slot %d (attempt %d): %s", slot, attempts+1, err)
			time.Sleep(utils.RoutineFlushTimeout * time.Duration(attempts+1))
		}
		attempts += 1
	}
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// requestProposerDuties fetches the proposer duties for epoch, retrying transient
// failures up to maxRetries with incremental backoff.
func (s *APIClient) requestProposerDuties(epoch phase0.Epoch) ([]*apiv1.ProposerDuty, error) {
	err := errors.New("first attempt")
	var resp *api.Response[[]*apiv1.ProposerDuty]

	attempts := 0
	for err != nil && attempts < s.maxRetries {
		resp, err = s.Api.ProposerDuties(s.ctx, &api.ProposerDutiesOpts{
			Epoch: epoch,
		})
		if err != nil {
			log.Warnf("retrying proposer duties request at epoch %d (attempt %d): %s", epoch, attempts+1, err)
			time.Sleep(utils.RoutineFlushTimeout * time.Duration(attempts+1))
		}
		attempts += 1
	}
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}
