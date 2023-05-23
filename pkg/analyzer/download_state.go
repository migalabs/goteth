package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

func (s StateAnalyzer) runBackfill(wgDownload *sync.WaitGroup,
	initSlot phase0.Slot,
	endSlot phase0.Slot) {

	slotChan := make(chan phase0.Slot, 1)
	if initSlot > endSlot {
		log.Errorf("backfill not allowed: initSlot greater than endSlot")
		return
	}

	// make sure init and end Slot are a start and end of epochs, never in the middle
	initSlot = phase0.Slot(phase0.Epoch(initSlot/local_spec.SlotsPerEpoch) * local_spec.SlotsPerEpoch)
	endSlot = phase0.Slot((phase0.Epoch(endSlot/local_spec.SlotsPerEpoch)+1)*local_spec.SlotsPerEpoch) - 1

	go s.runDownloads(wgDownload, slotChan, false)
	for i := initSlot; i <= endSlot; i++ {
		if s.stop {
			return
		}
		slotChan <- i
	}
}

func (s StateAnalyzer) runDownloads(
	wgDownload *sync.WaitGroup,
	triggerChan <-chan phase0.Slot,
	finalized bool) {

	defer wgDownload.Done()
	log.Info("Launching Beacon State Requester")
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
	// loop over the list of slots that we need to analyze
	states := NewStateQueue()
	states.Finalized = finalized
	blockList := make([]local_spec.AgnosticBlock, 0)
	nextSlot := <-triggerChan // wait to know which slot are we at
	epoch := phase0.Epoch(nextSlot / local_spec.SlotsPerEpoch)

	// prevSlot always starts at the beginning of an epoch -> full epoch metrics
	slot := phase0.Slot((epoch - 3) * local_spec.SlotsPerEpoch) // first slot of three epochs before

loop:
	for {

		select {

		case nextSlot := <-triggerChan:
			if s.stop {
				log.Info("sudden shutdown detected, state downloader routine")
				break loop
			}
			for slot <= nextSlot { // always fill from previous requested slot to current chan slot
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
					// End of epoch, add new state state

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

		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			break loop
		case <-ticker.C:
			if s.stop {
				log.Info("sudden shutdown detected, state downloader routine")
				return
			}

		}

	}

	log.Infof("State Downloader routine finished")
}

func (s *StateAnalyzer) runDownloadFinalized(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()

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

	if lastRequestEpoch > 0 && headSlot > 0 &&
		phase0.Epoch(headSlot/local_spec.SlotsPerEpoch) > lastRequestEpoch {
		// it means we could obtain both, there is probably some data in the database
		go s.runBackfill(wgDownload, phase0.Slot((lastRequestEpoch-4)*32), headSlot)

	}

	// --------------------------------- Begin the infinte loop -----------------------------------
	s.eventsObj.SubscribeToHeadEvents()

	go s.runDownloads(wgDownload, s.eventsObj.HeadChan, true)

}
