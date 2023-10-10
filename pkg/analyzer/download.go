package analyzer

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func (s ChainAnalyzer) DownloadBlock(slot phase0.Slot) {
	if !s.metrics.Block {
		log.Infof("skipping block download at slot %d: no metrics activated for block...", slot)
		return
	}

	newBlock, err := s.cli.RequestBeaconBlock(slot)
	if err != nil {
		log.Errorf("block error at slot %d: %s", slot, err)
		s.stop = true
	}
	s.queue.AddNewBlock(newBlock)
	// check if the min Request time has been completed (to avoid spaming the API)
}

func (s ChainAnalyzer) DownloadState(slot phase0.Slot) {
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
