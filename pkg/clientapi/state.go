package clientapi

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

func (s APIClient) RequestBeaconState(epoch phase0.Epoch) (*spec.VersionedBeaconState, error) {
	slot := (epoch+1)*local_spec.SlotsPerEpoch - 1
	initTime := time.Now()
	newState, err := s.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
	downloadTime := time.Since(initTime).Seconds()
	if newState == nil {
		return nil, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. nil State")
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. %s", err.Error())

	}
	log.Debugf("state at epoch %d took %f seconds to download", phase0.Epoch(slot/32), downloadTime)

	return newState, nil
}
