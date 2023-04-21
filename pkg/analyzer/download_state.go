package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

func (s *StateAnalyzer) runDownloadStates(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Requester")
	// loop over the list of slots that we need to analyze

	// We need three consecutive states to compute max rewards easier
	prevBState := local_spec.AgnosticState{}
	bstate := local_spec.AgnosticState{}
	nextBstate := local_spec.AgnosticState{}

	// Start two epochs before and end one epoch after
	for slot := s.initSlot; slot < s.finalSlot; slot += local_spec.SlotSeconds {

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

			err := s.DownloadNewState(&prevBState, &bstate, &nextBstate, phase0.Slot(slot), false)
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
	prevBState := local_spec.AgnosticState{}
	bstate := local_spec.AgnosticState{}
	nextBstate := local_spec.AgnosticState{}

	// ------ fill from last epoch in database to current head -------

	// obtain last epoch in database
	lastRequestEpoch, err := s.dbClient.ObtainLastEpoch()
	if err != nil {
		log.Errorf("could not obtain last epoch in database")
	}

	// obtain current head
	headSlot := phase0.Slot(0)
	header, err := s.cli.Api.BeaconBlockHeader(s.ctx, "head")
	if err != nil {
		log.Errorf("could not obtain current head to fill historical")
	} else {
		headSlot = header.Header.Message.Slot
	}

	// it means we could obtain both
	if lastRequestEpoch > 0 && headSlot > 0 {
		headEpoch := phase0.Epoch(headSlot / phase0.Slot(local_spec.EpochSlots))
		lastRequestEpoch = lastRequestEpoch - 4 // start 4 epochs before for safety
		for (lastRequestEpoch) < (headEpoch - 1) {
			lastRequestEpoch = lastRequestEpoch + 1
			log.Infof("filling missing epochs: %d", lastRequestEpoch)
			slot := phase0.Slot(int(lastRequestEpoch)*local_spec.EpochSlots) + 31 // last slot of the epoch

			err := s.DownloadNewState(&prevBState, &bstate, &nextBstate, slot, true)
			if err != nil {
				log.Errorf("error downloading state at slot %d", slot, err)
				continue
			}
			if s.finishDownload {
				log.Info("sudden shutdown detected, state downloader routine")
				close(s.EpochTaskChan)
				return
			}
		}

	}

	// -----------------------------------------------------------------------------------
	s.eventsObj.SubscribeToHeadEvents()
	for {
		if s.finishDownload {
			log.Info("sudden shutdown detected, state downloader routine")
			close(s.EpochTaskChan)
			return
		}
		select {

		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			close(s.EpochTaskChan)
			return

		case newHead := <-s.eventsObj.HeadChan:
			// new epoch
			headEpoch := phase0.Epoch(newHead / phase0.Slot(local_spec.EpochSlots))
			if lastRequestEpoch == 0 {
				lastRequestEpoch = headEpoch
			}
			// reqEpoch => headEpoch - 1
			log.Infof("Head Epoch: %d; Last Requested Epoch: %d", headEpoch, lastRequestEpoch)
			log.Infof("Pending slots for new epoch: %d", (int(headEpoch+1)*local_spec.EpochSlots)-int(newHead))

			if (lastRequestEpoch + 1) >= headEpoch {
				continue // wait for new epoch to arrive
			}

			slot := phase0.Slot(int(lastRequestEpoch+1)*local_spec.EpochSlots) + 31 // last slot of the epoch
			lastRequestEpoch = lastRequestEpoch + 1

			err := s.DownloadNewState(&prevBState, &bstate, &nextBstate, slot, true)
			if err != nil {
				log.Errorf("error downloading state at slot %d", slot, err)
				continue
			}

		}

	}
}

func (s StateAnalyzer) RequestBeaconState(slot phase0.Slot) (*spec.VersionedBeaconState, error) {
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
	prevBState *local_spec.AgnosticState,
	bstate *local_spec.AgnosticState,
	nextBstate *local_spec.AgnosticState,
	slot phase0.Slot,
	finalized bool) error {

	log := log.WithField("routine", "download")
	ticker := time.NewTicker(minStateReqTime)
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
	*nextBstate, err = local_spec.GetCustomState(*newState, s.cli.Api)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return fmt.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
	}

	if prevBState.AttestingBalance != nil {
		// only execute tasks if prevBState is something (we have state and nextState in this case)

		epochTask := &EpochTask{
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
