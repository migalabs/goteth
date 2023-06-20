package clientapi

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

func (s APIClient) RequestBeaconState(slot phase0.Slot) (*local_spec.AgnosticState, error) {
	newState, err := s.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
	if newState == nil {
		return nil, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. nil State")
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. %s", err.Error())

	}

	resultState, err := local_spec.GetCustomState(*newState, s.NewEpochData(slot))
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
	}

	return &resultState, nil
}
