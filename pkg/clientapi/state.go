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

	routineKey := fmt.Sprintf("%s%d", stateKeyTag, slot)
	s.statesBook.Acquire(routineKey)
	defer s.statesBook.FreePage(routineKey)

	startTime := time.Now()

	err := errors.New("first attempt")
	var newState *api.Response[*spec.VersionedBeaconState]

	attempts := 0
	for err != nil && attempts < maxRetries {

		newState, err = s.Api.BeaconState(s.ctx, &api.BeaconStateOpts{
			State: fmt.Sprintf("%d", slot),
		})

		if newState == nil {
			return nil, fmt.Errorf("unable to retrieve Beacon State from the beacon node, closing requester routine. nil State")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			ticker := time.NewTicker(utils.RoutineFlushTimeout)
			log.Warnf("retrying request: %s", routineKey)
			<-ticker.C

		}
		attempts += 1

	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())

	}

	log.Infof("state at slot %d downloaded in %f seconds", slot, time.Since(startTime).Seconds())
	resultState, err := local_spec.GetCustomState(*newState.Data, s.NewEpochData(slot))
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
	}
	// We have used HashTreeRoot method to hash the downloaded state, but it does not work ok
	// meantime, we use this
	resultState.StateRoot = s.RequestStateRoot(slot)

	return &resultState, nil
}

func (s *APIClient) RequestStateRoot(slot phase0.Slot) phase0.Root {

	root, err := s.Api.BeaconStateRoot(s.ctx, &api.BeaconStateRootOpts{
		State: fmt.Sprintf("%d", slot),
	})
	if err != nil {
		log.Panicf("could not download the state root at %d: %s", slot, err)
	}

	return *root.Data
}

// Finalized Checkpoints happen at the beginning of an epoch
// This method returns the finalized slot at the end of an epoch
// Usually, it is the slot before the finalized one
func (s *APIClient) GetFinalizedEndSlotStateRoot() (phase0.Slot, phase0.Root) {

	currentFinalized, err := s.Api.Finality(s.ctx, &api.FinalityOpts{
		State: "head",
	})

	if err != nil {
		log.Panicf("could not determine the current finalized checkpoint")
	}

	finalizedSlot := phase0.Slot(currentFinalized.Data.Finalized.Epoch*local_spec.SlotsPerEpoch - 1)

	root := s.RequestStateRoot(finalizedSlot)

	return finalizedSlot, root
}
