package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

func (s *StateAnalyzer) runDownloadStates(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Requester")
	// loop over the list of slots that we need to analyze

	// We need three consecutive states to compute max rewards easier
	queue := StateQueue{
		prevState:    local_spec.AgnosticState{},
		currentState: local_spec.AgnosticState{},
		nextState:    local_spec.AgnosticState{},
	}
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

			s.DownloadNewState(&queue, phase0.Slot(slot), false)

		}

	}

	log.Infof("State Downloader routine finished")
}

func (s *StateAnalyzer) runDownloadStatesFinalized(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Finalized Requester")

	slot, root := s.cli.GetFinalizedEndSlotStateRoot()

	queue := NewStateQueue(slot, root)

	nextEpochDownload := phase0.Epoch(slot / spec.SlotsPerEpoch)

	// obtain last epoch in database
	lastRequestEpoch, err := s.dbClient.ObtainLastEpoch()
	if err != nil {
		log.Errorf("could not obtain last epoch in database")
	}
	headEpoch := phase0.Epoch(s.cli.RequestCurrentHead() / spec.SlotsPerEpoch)

	if lastRequestEpoch > epochsToFinalizedTentative { // otherwise when we subsctract it would be less than 0
		log.Infof("database detected, last epoch: %d", lastRequestEpoch)
		nextEpochDownload = lastRequestEpoch - epochsToFinalizedTentative
	}

	// download until the previous epoch to the head: pre-fill
	for nextEpochDownload < headEpoch {
		headEpoch = phase0.Epoch(s.cli.RequestCurrentHead() / spec.SlotsPerEpoch)
		log.Infof("filling epochs until head: %d - %d", nextEpochDownload, headEpoch)
		slot := phase0.Slot(int(nextEpochDownload)*local_spec.EpochSlots) + (spec.SlotsPerEpoch - 1) // last slot of the epoch
		s.DownloadNewState(&queue, slot, true)
		nextEpochDownload += 1
	}

	s.eventsObj.SubscribeToHeadEvents()
	s.eventsObj.SubscribeToFinalizedCheckpointEvents()
	ticker := time.NewTicker(utils.RoutineFlushTimeout)

	// from now on we are following the chain head
	for {

		select {

		case newHead := <-s.eventsObj.HeadChan:
			headEpoch = phase0.Epoch(newHead / phase0.Slot(local_spec.EpochSlots))

			// download until the previous epoch to the head
			for nextEpochDownload < headEpoch {
				log.Infof("Head Epoch: %d; Last Requested Epoch: %d", headEpoch, nextEpochDownload-1)
				slot := phase0.Slot(int(nextEpochDownload)*local_spec.EpochSlots) + (spec.SlotsPerEpoch - 1) // last slot of the epoch
				s.DownloadNewState(&queue, slot, true)
				nextEpochDownload += 1
			}

		case newReorg := <-s.eventsObj.ReorgChan:
			s.dbClient.Persist(db.ReorgTypeFromReorg(newReorg))
			headReorgEpoch := phase0.Epoch(newReorg.Slot / spec.SlotsPerEpoch)
			baseReorgEpoch := phase0.Epoch((newReorg.Slot - phase0.Slot(newReorg.Depth)) / spec.SlotsPerEpoch)

			// if the reorg is at the end of an epoch, or an epoch boundary has been crossed
			if newReorg.Slot%spec.SlotsPerEpoch == 31 ||
				headReorgEpoch > baseReorgEpoch {

				slot, root := s.cli.GetFinalizedEndSlotStateRoot()
				queue = NewStateQueue(slot, root)
				nextEpochDownload = phase0.Epoch(slot / spec.SlotsPerEpoch)
				log.Infof("rewinding to finalized %d", nextEpochDownload)
				s.ReorgRewind(baseReorgEpoch - 1) // delete metrics from next and current state (check statequeue)
			}

		case newFinalCheckpoint := <-s.eventsObj.FinalizedChan:
			s.dbClient.Persist(db.ChepointTypeFromCheckpoint(newFinalCheckpoint))

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

func (s *StateAnalyzer) ReorgRewind(epoch phase0.Epoch) {

	log.Infof("deleting epoch data from %d (included) onwards", epoch)
	s.dbClient.Persist(db.EpochDropType(epoch))
	s.dbClient.Persist(db.ProposerDutiesDropType(epoch))
	s.dbClient.Persist(db.ValidatorRewardsDropType(epoch))

}
