package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

func (s *StateAnalyzer) runDownloadStates(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Requester")
	// loop over the list of slots that we need to analyze

	// We need three consecutive states to compute max rewards easier
	prevBState := local_spec.AgnosticState{}
	bstate := local_spec.AgnosticState{}
	nextBstate := local_spec.AgnosticState{}
loop:
	for slot := s.initSlot; slot < s.finalSlot; slot += local_spec.SlotsPerEpoch {

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			break loop

		default:
			if s.stop {
				log.Info("sudden shutdown detected, state downloader routine")
				break loop
			}

			err := s.DownloadNewState(&prevBState, &bstate, &nextBstate, phase0.Slot(slot), false)
			if err != nil {
				log.Errorf("error downloading state at slot %d: %s", slot, err)
				return
			}

		}

	}

	log.Infof("State Downloader routine finished")
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
				log.Errorf("error downloading state at slot %d: %s", slot, err)
				continue
			}
			if s.stop {
				log.Info("sudden shutdown detected, state downloader routine")
				return
			}
		}

	}

	// -----------------------------------------------------------------------------------
	s.eventsObj.SubscribeToHeadEvents()
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
	for {

		select {

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
				log.Errorf("error downloading state at slot %d: %s", slot, err.Error())
				continue
			}
		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			return

		case <-ticker.C:
			if s.stop {
				log.Info("sudden shutdown detected, state downloader routine")
				return
			}
		}

	}
}

func (s *StateAnalyzer) DownloadNewState(
	queue *StateQueue,
	slot phase0.Slot,
	finalized bool) {

	log := log.WithField("routine", "download")
	ticker := time.NewTicker(minStateReqTime)

	log.Infof("requesting Beacon State from endpoint: epoch %d", slot/spec.SlotsPerEpoch)
	nextBstate, err := s.cli.RequestBeaconState(slot)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		log.Panicf("unable to retrieve beacon state from the beacon node, closing requester routine. %s", err.Error())
	}

	queue.AddNewState(*nextBstate)

	if queue.Complete() {
		// only execute tasks if prevBState is something (we have state and nextState in this case)

		epochTask := &EpochTask{
			NextState: queue.nextState,
			State:     queue.currentState,
			PrevState: queue.prevState,
			Finalized: finalized,
		}

		log.Debugf("sending task for slot: %d", epochTask.State.Slot)
		s.epochTaskChan <- epochTask
	}
	// check if the min Request time has been completed (to avoid spaming the API)
	<-ticker.C
}
