package clientapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

var (
	stateKeyTag string = "state="
)

func (s *APIClient) RequestBeaconState(slot phase0.Slot) (*local_spec.AgnosticState, error) {
	return s.requestBeaconStateWithID(slot, fmt.Sprintf("%d", slot), phase0.Root{})
}

// RequestBeaconStateByRoot fetches the beacon state using the state root
// instead of the slot number. This avoids the racy slot-based resolution
// in Lighthouse v8.1.0+ where the Head SSE event is emitted before
// canonical_head is updated.
func (s *APIClient) RequestBeaconStateByRoot(slot phase0.Slot, root phase0.Root) (*local_spec.AgnosticState, error) {
	return s.requestBeaconStateWithID(slot, fmt.Sprintf("%#x", root), root)
}

func (s *APIClient) requestBeaconStateWithID(slot phase0.Slot, stateID string, knownRoot phase0.Root) (*local_spec.AgnosticState, error) {
	routineKey := fmt.Sprintf("%s%d", stateKeyTag, slot)
	s.statesBook.Acquire(routineKey)
	defer s.statesBook.FreePage(routineKey)

	startTime := time.Now()

	err := errors.New("first attempt")
	var newState *api.Response[*spec.VersionedBeaconState]

	attempts := 0
	for err != nil && attempts < s.maxRetries {

		newState, err = s.Api.BeaconState(s.ctx, &api.BeaconStateOpts{
			State: stateID,
		})

		if errors.Is(err, context.DeadlineExceeded) {
			ticker := time.NewTicker(utils.RoutineFlushTimeout)
			log.Warnf("retrying request: %s", routineKey)
			<-ticker.C

		}
		attempts += 1

	}
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())

	}

	log.Infof("state at slot %d downloaded in %f seconds (id=%s)", slot, time.Since(startTime).Seconds(), stateID)
	resultState, err := local_spec.GetCustomState(*newState.Data, s.NewEpochData(slot))
	if err != nil {
		return nil, fmt.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
	}

	var zeroRoot phase0.Root
	if knownRoot != zeroRoot {
		resultState.StateRoot = knownRoot
	} else {
		stateRoot, err := s.RequestStateRoot(slot)
		if err != nil {
			return nil, fmt.Errorf("unable to get state root at slot %d: %w", slot, err)
		}
		resultState.StateRoot = stateRoot
	}

	return &resultState, nil
}

func (s *APIClient) RequestStateRoot(slot phase0.Slot) (phase0.Root, error) {

	root, err := s.Api.BeaconStateRoot(s.ctx, &api.BeaconStateRootOpts{
		State: fmt.Sprintf("%d", slot),
	})
	if err != nil {
		return phase0.Root{}, fmt.Errorf("could not download the state root at %d: %w", slot, err)
	}

	return *root.Data, nil
}

// Finalized Checkpoints happen at the beginning of an epoch
// This method returns the finalized slot at the end of an epoch
// Usually, it is the slot before the finalized one
func (s *APIClient) GetFinalizedEndSlotStateRoot() (phase0.Slot, phase0.Root, error) {

	currentFinalized, err := s.Api.Finality(s.ctx, &api.FinalityOpts{
		State: "head",
	})

	if err != nil {
		return 0, phase0.Root{}, fmt.Errorf("could not determine the current finalized checkpoint: %w", err)
	}

	finalizedSlot := phase0.Slot(currentFinalized.Data.Finalized.Epoch*local_spec.SlotsPerEpoch - 1)

	root, err := s.RequestStateRoot(finalizedSlot)
	if err != nil {
		return 0, phase0.Root{}, fmt.Errorf("could not get finalized state root: %w", err)
	}

	return finalizedSlot, root, nil
}
