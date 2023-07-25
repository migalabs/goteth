package clientapi

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

func (s APIClient) RequestBeaconState(slot phase0.Slot) (*local_spec.AgnosticState, error) {
	startTime := time.Now()
	newState, err := s.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))

	if newState == nil {
		return nil, fmt.Errorf("unable to retrieve Beacon State from the beacon node, closing requester routine. nil State")
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())

	}

	log.Infof("state at slot %d downloaded in %f seconds", slot, time.Since(startTime).Seconds())
	resultState, err := local_spec.GetCustomState(*newState, s.NewEpochData(slot))
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
	}
	// We have used HashTreeRoot method to hash the downloaded state, but it does not work ok
	// meantime, we use this
	resultState.StateRoot = s.RequestStateRoot(slot)

	return &resultState, nil
}

func (s APIClient) RequestStateRoot(slot phase0.Slot) phase0.Root {
	root, err := s.Api.BeaconStateRoot(s.ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		log.Panicf("could not download the state root at %d: %s", slot, err)
	}

	return *root
}

// Finalized Checkpoints happen at the beginning of an epoch
// This method returns the finalized slot at the end of an epoch
// Usually, it is the slot before the finalized one
func (s APIClient) GetFinalizedEndSlotStateRoot() (phase0.Slot, phase0.Root) {
	currentFinalized, err := s.Api.Finality(s.ctx, "head")

	if err != nil {
		log.Panicf("could not determine the current finalized checkpoint")
	}

	finalizedSlot := phase0.Slot(currentFinalized.Finalized.Epoch*local_spec.SlotsPerEpoch - 1)

	root := s.RequestStateRoot(finalizedSlot)

	return finalizedSlot, root
}
