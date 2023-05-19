package clientapi

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func (s APIClient) RequestBeaconState(slot phase0.Slot) (*spec.VersionedBeaconState, error) {
	newState, err := s.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
	if newState == nil {
		return nil, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. nil State")
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. %s", err.Error())

	}
	return newState, nil
}
