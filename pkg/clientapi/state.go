package clientapi

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

func (s APIClient) RequestBeaconState(epoch phase0.Epoch) (local_spec.AgnosticState, error) {
	slot := (epoch+1)*local_spec.SlotsPerEpoch - 1
	initTime := time.Now()
	log.Infof("downloading state at epoch %d (slot %d)", phase0.Epoch(slot/32), slot)
	newState, err := s.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
	downloadTime := time.Since(initTime).Seconds()
	if newState == nil {
		return local_spec.AgnosticState{}, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. nil State")
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return local_spec.AgnosticState{}, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. %s", err.Error())

	}
	log.Infof("state at epoch %d (slot %d) took %f seconds to download", phase0.Epoch(slot/32), slot, downloadTime)

	epochDuties := s.NewEpochDuties(epoch)

	resultState, err := local_spec.GetCustomState(*newState, epochDuties)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return local_spec.AgnosticState{}, fmt.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
	}

	return resultState, nil
}

func (s APIClient) DownloadBeaconStateAndBlocks(epoch phase0.Epoch) (local_spec.AgnosticState, error) {
	newState, err := s.RequestBeaconState(epoch)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return local_spec.AgnosticState{}, fmt.Errorf("unable to retrieve Beacon State at %d: %s", phase0.Slot((epoch+1)*local_spec.SlotsPerEpoch-1), err.Error())

	}
	blockList := make([]local_spec.AgnosticBlock, 0)
	for i := phase0.Slot((epoch) * local_spec.SlotsPerEpoch); i < phase0.Slot((epoch+1)*local_spec.SlotsPerEpoch-1); i++ {
		block, err := s.RequestBeaconBlock(i)
		if err != nil {
			return local_spec.AgnosticState{}, fmt.Errorf("unable to retrieve Beacon State at %d: %s", phase0.Slot((epoch+1)*local_spec.SlotsPerEpoch-1), err.Error())
		}
		blockList = append(blockList, block)
	}
	newState.BlockList = blockList

	return newState, nil
}
