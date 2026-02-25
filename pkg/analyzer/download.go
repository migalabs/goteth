package analyzer

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
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
		s.cancel()
		return
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

	var state *spec.AgnosticState
	var err error

	// Try to use the cached state root from the Head SSE event (root-based lookup).
	// This avoids the racy slot-based resolution in Lighthouse v8.1.0+ where the
	// Head event is emitted before canonical_head is updated.
	// A short delay + retry is needed because the state may not be queryable yet
	// at the moment the Head event fires.
	if root, ok := s.takeEpochBoundaryStateRoot(slot); ok {
		log.Debugf("using cached state root for slot %d", slot)
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(1 * time.Second)
				log.Debugf("retrying state-by-root for slot %d (attempt %d)", slot, attempt+1)
			}
			state, err = s.cli.RequestBeaconStateByRoot(slot, root)
			if err == nil {
				break
			}
		}
	} else {
		state, err = s.cli.RequestBeaconState(slot)
	}

	if err != nil {
		log.Errorf("unable to retrieve beacon state from the beacon node, closing requester routine. %s", err.Error())
		s.stop = true
		s.cancel()
		return
	}

	if err := s.downloadCache.AddNewState(s.ctx, state); err != nil {
		log.Errorf("context cancelled adding state at slot %d: %s", slot, err)
		return
	}
}

func (s *ChainAnalyzer) WaitForPrevState(slot phase0.Slot) {
	// check if state two epochs before is available
	// the idea is that blocks are too fast to download, wait for states as well

	if !s.metrics.Epoch {
		return // no states downloaded when epoch metrics are disabled
	}

	if slot < spec.SlotsPerEpoch*2 {
		return
	}
	prevStateEpoch := slot/spec.SlotsPerEpoch - 2              // epoch to check if state downloaded
	prevStateSlot := (prevStateEpoch+1)*spec.SlotsPerEpoch - 1 // slot at which the check state was downloaded

	prevStateAvailable := s.downloadCache.StateHistory.Available(uint64(prevStateEpoch))
	prevStateProcessing := s.processerBook.CheckPageActive(fmt.Sprintf("%s%d", epochProcesserTag, prevStateEpoch))

	// do not continue until previous state is available and is not being processed anymore
	// also check that prevstate was supposed to be downloaded
	if (!prevStateAvailable || prevStateProcessing) && prevStateSlot >= s.initSlot {
		ticker := time.NewTicker(4 * time.Second) // average max time for a state to be downloaded
		defer ticker.Stop()
	stateWaitLoop:
		for {
			select {
			case <-s.ctx.Done():
				log.Infof("context cancelled while waiting for prev state at slot %d", slot)
				return
			case <-ticker.C:
				if slot%spec.SlotsPerEpoch == 0 { // only print for first slot of epoch
					log.Debugf("slot %d waiting for state at slot %d (epoch %d) to be downloaded or processed...", slot, prevStateSlot, prevStateEpoch)
				}

				prevStateAvailable = s.downloadCache.StateHistory.Available(uint64(prevStateEpoch))
				prevStateProcessing = s.processerBook.CheckPageActive(fmt.Sprintf("%s%d", epochProcesserTag, prevStateEpoch))
				if prevStateAvailable && !prevStateProcessing {
					// it was available in the queue and processed
					break stateWaitLoop
				}
			}
		}
	}
}
