package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
)

func (s *StateAnalyzer) runDownloadStates(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Requester")
	// loop over the list of slots that we need to analyze
	var prevBState spec.VersionedBeaconState // to be checked, it may make calculation easier to store previous state
	var bstate *spec.VersionedBeaconState
	var nextBstate *spec.VersionedBeaconState
	var err error
	for _, slot := range s.SlotRanges {
		ticker := time.NewTicker(minReqTime)
		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			close(s.EpochTaskChan)
			return

		default:
			firstIteration := true
			secondIteration := true
			// make the state query
			log.Infof("requesting Beacon State from endpoint: slot %d", slot)

			// We need three states to calculate both, rewards and maxRewards

			if bstate != nil { // in case we already had a bstate (only false the first time)
				prevBState = *bstate
				firstIteration = false
			}
			if nextBstate != nil { // in case we already had a nextBstate (only false the first time)
				*bstate = *nextBstate
				secondIteration = false
			}
			// snapshot := time.Now()

			if firstIteration {
				bstate, err = s.cli.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
				if err != nil {
					// close the channel (to tell other routines to stop processing and end)
					log.Errorf("Unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())
					return
				}
			} else {
				nextBstate, err = s.cli.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
				if err != nil {
					// close the channel (to tell other routines to stop processing and end)
					log.Errorf("Unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())
					return
				}
			}

			// s.MonitorMetrics.AddDownload(time.Since(snapshot).Seconds())
			if !firstIteration && !secondIteration {
				// only execute tasks if it is not the first iteration

				// we now only compose one single task that contains a list of validator indexes
				// compose the next task
				epochTask := &EpochTask{
					ValIdxs:   s.ValidatorIndexes,
					Slot:      slot,
					NextState: nextBstate,
					State:     *bstate,
					PrevState: prevBState,
				}

				log.Debugf("sending task for slot: %d", slot)
				s.EpochTaskChan <- epochTask
			}

		}
		// check if the min Request time has been completed (to avoid spaming the API)
		<-ticker.C
	}

	log.Infof("All states for the slot ranges has been successfully retrieved, clossing go routine")
}

func (s *StateAnalyzer) runDownloadStatesFinalized(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Finalized Requester")
	// loop over the list of slots that we need to analyze
	var prevBState spec.VersionedBeaconState // to be checked, it may make calculation easier to store previous state
	var bstate *spec.VersionedBeaconState
	var nextBstate *spec.VersionedBeaconState
	finalizedSlot := 0
	// epochSeconds := SLOT_SECONDS * EPOCH_SLOTS
	timerCh := time.Tick(time.Second * 384)
	for {
		ticker := time.NewTicker(minReqTime)
		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			close(s.EpochTaskChan)
			return
		case <-timerCh:

			firstIteration := true
			secondIteration := true
			// make the state query
			log.Infof("requesting Beacon State from endpoint: finalized")
			if bstate != nil { // in case we already had a bstate (only false the first time)
				prevBState = *bstate
				secondIteration = false
			}
			if nextBstate != nil { // in case we already had a nextBstate (only false the first time)
				*bstate = *nextBstate
				firstIteration = false
			}
			// snapshot := time.Now()
			header, err := s.cli.Api.BeaconBlockHeader(s.ctx, "finalized")
			if err != nil {
				log.Errorf("Unable to retrieve Beacon State from the beacon node, closing finalized requester routine. %s", err.Error())
				return
			}
			if int(header.Header.Message.Slot) == finalizedSlot {
				log.Infof("No new finalized state yet")
				continue
			}

			finalizedSlot = int(header.Header.Message.Slot) - 1
			log.Infof("New finalized state at slot: %d", finalizedSlot)
			if firstIteration {
				bstate, err = s.cli.Api.BeaconState(s.ctx, fmt.Sprintf("%d", finalizedSlot))
				if err != nil {
					// close the channel (to tell other routines to stop processing and end)
					log.Errorf("Unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())
					return
				}
			} else {
				nextBstate, err = s.cli.Api.BeaconState(s.ctx, fmt.Sprintf("%d", finalizedSlot))
				if err != nil {
					// close the channel (to tell other routines to stop processing and end)
					log.Errorf("Unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())
					return
				}
			}
			// s.MonitorMetrics.AddDownload(time.Since(snapshot).Seconds())
			if !firstIteration && !secondIteration {
				// only execute tasks if it is not the first iteration

				// we now only compose one single task that contains a list of validator indexes
				// compose the next task
				epochTask := &EpochTask{
					ValIdxs:   s.ValidatorIndexes,
					Slot:      uint64(finalizedSlot),
					NextState: nextBstate,
					State:     *bstate,
					PrevState: prevBState,
				}

				log.Debugf("sending task for slot: %d", finalizedSlot)
				s.EpochTaskChan <- epochTask
			}
		default:

		}
		// check if the min Request time has been completed (to avoid spaming the API)
		<-ticker.C
	}

	// log.Infof("All states for the slot ranges has been successfully retrieved, clossing go routine")
}
