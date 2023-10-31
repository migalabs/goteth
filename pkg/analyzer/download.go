package analyzer

import (
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

func (s *ChainAnalyzer) DownloadBlockCotrolled(slot phase0.Slot) {
	s.WaitForPrevState(slot)
	s.DownloadBlock(slot)
}

func (s *ChainAnalyzer) DownloadBlock(slot phase0.Slot) {
	if !s.metrics.Block {
		log.Infof("skipping block download at slot %d: no metrics activated for block...", slot)
		return
	}

	newBlock, err := s.cli.RequestBeaconBlock(slot)
	if err != nil {
		log.Errorf("block error at slot %d: %s", slot, err)
		s.stop = true
	}
	s.downloadCache.AddNewBlock(newBlock)
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

	s.downloadCache.AddNewState(*state)
	// check if the min Request time has been completed (to avoid spaming the API)
}

func (s *ChainAnalyzer) WaitForPrevState(slot phase0.Slot) {
	// check if state two epochs before is available
	// the idea is that blocks are too fast to download, wait for states as well

	prevStateEpoch := slot/spec.SlotsPerEpoch - 2              // epoch to check if state downloaded
	prevStateSlot := (prevStateEpoch+1)*spec.SlotsPerEpoch - 1 // slot at which the check state was downloaded

	prevStateAvailable := s.downloadCache.StateHistory.Available(uint64(prevStateEpoch))

	// do not continue until previous state is available
	if !prevStateAvailable && prevStateSlot >= s.initSlot {
		ticker := time.NewTicker(utils.RoutineFlushTimeout)
	stateWaitLoop:
		for range ticker.C {
			log.Tracef("slot %d waiting for state at slot %d (epoch %d) to be downloaded...", slot, prevStateSlot, prevStateEpoch)
			prevStateAvailable = s.downloadCache.StateHistory.Available(uint64(prevStateEpoch))
			if prevStateAvailable {
				ticker.Stop()
				break stateWaitLoop
			}
		}
	}
}