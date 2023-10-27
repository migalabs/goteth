package analyzer

import (
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

func (s *ChainAnalyzer) DownloadBlock(slot phase0.Slot) {
	if !s.metrics.Block {
		log.Infof("skipping block download at slot %d: no metrics activated for block...", slot)
		return
	}

	prevStateAvailable := s.queue.StateHistory.Available(uint64(slot/spec.SlotsPerEpoch) - 1)
	prevStateSlot := slot/spec.SlotsPerEpoch*spec.SlotsPerEpoch - 1

	// do not continue until previous state is available
	if !prevStateAvailable && prevStateSlot >= s.initSlot {
		ticker := time.NewTicker(utils.RoutineFlushTimeout)
		for range ticker.C {
			if slot%spec.SlotsPerEpoch == 0 { // only show warning for the first slot in the epoch
				log.Warnf("waiting for state at epoch %d to be downloaded...", (slot/spec.SlotsPerEpoch)-1)
			}
			prevStateAvailable = s.queue.StateHistory.Available(uint64(slot/spec.SlotsPerEpoch) - 1)
			if prevStateAvailable {
				ticker.Stop()
			}
		}
	}

	newBlock, err := s.cli.RequestBeaconBlock(slot)
	if err != nil {
		log.Errorf("block error at slot %d: %s", slot, err)
		s.stop = true
	}
	s.queue.AddNewBlock(newBlock)
	// check if the min Request time has been completed (to avoid spaming the API)
}

func (s *ChainAnalyzer) DownloadState(slot phase0.Slot) {
	if !s.metrics.Epoch {
		log.Infof("skipping state download: no metrics activated for state...")
		return
	}
	log := log.WithField("routine", "download")

	state, err := s.cli.RequestBeaconState(slot)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		log.Errorf("unable to retrieve beacon state from the beacon node, closing requester routine. %s", err.Error())
		s.stop = true
	}

	s.queue.AddNewState(*state)
	// check if the min Request time has been completed (to avoid spaming the API)
}
