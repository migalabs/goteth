package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (s *StateAnalyzer) prepareBackfill(wgDownload *sync.WaitGroup,
	initSlot phase0.Slot,
	endSlot phase0.Slot,
	id_name string) {
	slotChan := make(chan phase0.Slot, 1)
	finishChan := make(chan interface{}, 1)
	if initSlot > endSlot {
		log.Errorf("backfill not allowed: initSlot greater than endSlot")
		return
	}

	// make sure init and end Slot are a start and end of epochs, never in the middle
	initSlot = phase0.Slot(phase0.Epoch(initSlot/local_spec.SlotsPerEpoch) * local_spec.SlotsPerEpoch)
	backFillEpoch := phase0.Epoch(initSlot/local_spec.SlotsPerEpoch - 2)
	endSlot = phase0.Slot((phase0.Epoch(endSlot/local_spec.SlotsPerEpoch)+1)*local_spec.SlotsPerEpoch) - 1

	go s.runDownloads(wgDownload, slotChan, finishChan, false, id_name, backFillEpoch)
	for i := initSlot; i <= endSlot; i++ {
		if s.stop {
			return
		}
		slotChan <- i
	}
}

func (s *StateAnalyzer) prepareFinalized(wgDownload *sync.WaitGroup) {

	finishChan := make(chan interface{}, 1)
	// obtain last epoch in database
	lastRequestEpoch, err := s.dbClient.ObtainLastEpoch()
	if err == nil && lastRequestEpoch > 0 {
		log.Infof("database detected: backfilling from epoch %d", lastRequestEpoch)
		lastRequestEpoch = lastRequestEpoch - 2
	}

	// --------------------------------- Begin the infinte loop -----------------------------------
	s.eventsObj.SubscribeToHeadEvents()

	go s.runDownloads(wgDownload,
		s.eventsObj.HeadChan,
		finishChan,
		true,
		"finalized",
		lastRequestEpoch)

}

func (s *StateAnalyzer) runDownloads(
	wgDownload *sync.WaitGroup,
	triggerChan <-chan phase0.Slot,
	finishChan <-chan interface{},
	finalized bool,
	id_name string,
	backfill phase0.Epoch) {
	defer wgDownload.Done()
	log := logrus.WithField("download-routine", id_name)

	log.Info("Launching Beacon State Requester")
	ticker := time.NewTicker(utils.RoutineFlushTimeout)

	states := NewStateQueue()
	states.Finalized = finalized
	blockList := make([]local_spec.AgnosticBlock, 0)
	nextSlot := <-triggerChan // wait to know which slot are we at
	epoch := phase0.Epoch(nextSlot / local_spec.SlotsPerEpoch)

	// slot always starts at the beginning of an epoch -> full epoch metrics
	slot := phase0.Slot(backfill * local_spec.SlotsPerEpoch) // first slot to start downloading
	log.Infof("downloads starting from %d to %d", slot, nextSlot)

	// if no backfill provided we start 2 epochs before the current epoch
	if slot == 0 {
		slot = phase0.Slot(epoch-2) * local_spec.SlotsPerEpoch
	}

loop:
	for {

		select {

		case nextSlot := <-triggerChan:
			for slot <= nextSlot && !s.stop { // always fill from previous requested slot to current chan slot
				epoch = phase0.Epoch(slot / local_spec.SlotsPerEpoch)
				log.Debugf("filling from %d to %d", slot, nextSlot)
				block, err := s.downloadNewBlock(phase0.Slot(slot))
				if err != nil {
					log.Errorf("error downloading state at slot %d", slot, err)
					return
				}
				blockList = append(blockList, block)

				// If last slot of epoch
				if (slot+1)%local_spec.SlotsPerEpoch == 0 {
					// End of epoch, add new state

					newState, err := s.downloadNewState(epoch)
					if err != nil {
						log.Errorf("could not download state at epoch %d: %s", slot%local_spec.SlotsPerEpoch, err)
					}
					newState.BlockList = blockList // add blockList to the new state
					err = states.AddNewState(newState)
					if err != nil {
						log.Errorf("could not add state to the list: %s", err)
						return
					}
					blockList = make([]local_spec.AgnosticBlock, 0) // clean the temp block list

					// send for metrics processing

					if states.Complete() {
						s.epochTaskChan <- &states
					}

				}
				slot += 1
			}
		case <-finishChan:
			break loop
		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			break loop
		case <-ticker.C:
			if s.stop {
				log.Info("sudden shutdown detected, state downloader routine")
				break loop
			}

		}

	}

	log.Infof("State Downloader routine finished")
}
