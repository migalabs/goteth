package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics/fork_state"
)

func (s *StateAnalyzer) runDownloadStates(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Requester")
	// loop over the list of slots that we need to analyze

	// We need three consecutive states to compute max rewards easier
	prevBState := fork_state.ForkStateContentBase{}
	bstate := fork_state.ForkStateContentBase{}
	nextBstate := fork_state.ForkStateContentBase{}
	for _, slot := range s.SlotRanges {

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			close(s.EpochTaskChan)
			return

		default:
			if s.finishDownload {
				log.Info("sudden shutdown detected, state downloader routine")
				close(s.EpochTaskChan)
				return
			}

			err := s.DownloadNewState(&prevBState, &bstate, &nextBstate, int(slot), false)
			if err != nil {
				log.Errorf("error downloading state at slot %d", slot, err)
				return
			}

		}

	}

	log.Infof("All states for the slot ranges has been successfully retrieved, clossing go routine")
}

func (s *StateAnalyzer) runDownloadStatesFinalized(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Finalized Requester")
	prevBState := fork_state.ForkStateContentBase{}
	bstate := fork_state.ForkStateContentBase{}
	nextBstate := fork_state.ForkStateContentBase{}

	// ------ fill from last epoch in database to current head -------

	// obtain last epoch in database
	lastRequestEpoch, err := s.dbClient.ObtainLastEpoch()
	if err != nil {
		log.Errorf("could not obtain last epoch in database")
	}

	// obtain current head
	headSlot := -1
	header, err := s.cli.Api.BeaconBlockHeader(s.ctx, "head")
	if err != nil {
		log.Errorf("could not obtain current head to fill historical")
	} else {
		headSlot = int(header.Header.Message.Slot)
	}

	// it means we could obtain both
	if lastRequestEpoch > 0 && headSlot > 0 {
		headEpoch := int(headSlot / EPOCH_SLOTS)
		lastRequestEpoch = lastRequestEpoch - 4 // start 4 epochs before for safety
		for (lastRequestEpoch) < (headEpoch - 1) {
			lastRequestEpoch = lastRequestEpoch + 1
			log.Infof("filling missing epochs: %d", lastRequestEpoch)
			slot := (lastRequestEpoch * EPOCH_SLOTS) + 31 // last slot of the epoch

			err := s.DownloadNewState(&prevBState, &bstate, &nextBstate, slot, true)
			if err != nil {
				log.Errorf("error downloading state at slot %d", slot, err)
				continue
			}
		}

	}

	// -----------------------------------------------------------------------------------
	s.eventsObj.SubscribeToHeadEvents()
	for {

		select {
		default:

			if s.finishDownload {
				log.Info("sudden shutdown detected, state downloader routine")
				close(s.EpochTaskChan)
				return
			}
		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			close(s.EpochTaskChan)
			return

		case newHead := <-s.eventsObj.HeadChan:
			// new epoch
			headEpoch := int(newHead / EPOCH_SLOTS)
			if lastRequestEpoch <= 0 {
				lastRequestEpoch = headEpoch
			}
			// reqEpoch => headEpoch - 1
			log.Infof("Head Epoch: %d; Last Requested Epoch: %d", headEpoch, lastRequestEpoch)
			log.Infof("Pending slots for new epoch: %d", ((headEpoch+1)*EPOCH_SLOTS)-newHead)

			if (lastRequestEpoch + 1) >= headEpoch {
				continue // wait for new epoch to arrive
			}

			slot := ((lastRequestEpoch + 1) * EPOCH_SLOTS) + 31 // last slot of the epoch
			lastRequestEpoch = lastRequestEpoch + 1

			err := s.DownloadNewState(&prevBState, &bstate, &nextBstate, slot, true)
			if err != nil {
				log.Errorf("error downloading state at slot %d", slot, err)
				continue
			}

		}

	}
}

func (s StateAnalyzer) RequestBeaconState(slot int) (*spec.VersionedBeaconState, error) {
	newState, err := s.cli.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
	if newState == nil {
		return nil, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. nil State")
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return nil, fmt.Errorf("unable to retrieve Finalized Beacon State from the beacon node, closing requester routine. %s", err.Error())

	}
	return newState, nil
}

func (s *StateAnalyzer) DownloadNewState(
	prevBState *fork_state.ForkStateContentBase,
	bstate *fork_state.ForkStateContentBase,
	nextBstate *fork_state.ForkStateContentBase,
	slot int,
	finalized bool) error {

	log := log.WithField("routine", "download")
	ticker := time.NewTicker(minReqTime)
	// We need three states to calculate both, rewards and maxRewards

	if bstate.AttestingBalance != nil { // in case we already had a bstate
		*prevBState = *bstate
	}
	if nextBstate.AttestingBalance != nil { // in case we already had a nextBstate
		*bstate = *nextBstate
	}
	log.Infof("requesting Beacon State from endpoint: slot %d", slot)
	newState, err := s.RequestBeaconState(slot)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return fmt.Errorf("unable to retrieve beacon state from the beacon node, closing requester routine. %s", err.Error())

	}
	*nextBstate, err = fork_state.GetCustomState(*newState, s.cli.Api)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return fmt.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
	}

	if prevBState.AttestingBalance != nil {
		// only execute tasks if prevBState is something (we have state and nextState in this case)

		epochTask := &EpochTask{
			ValIdxs:   s.ValidatorIndexes,
			NextState: *nextBstate,
			State:     *bstate,
			PrevState: *prevBState,
			Finalized: finalized,
		}

		log.Debugf("sending task for slot: %d", epochTask.State.Slot)
		s.EpochTaskChan <- epochTask
	}
	// check if the min Request time has been completed (to avoid spaming the API)
	<-ticker.C
	return nil
}
