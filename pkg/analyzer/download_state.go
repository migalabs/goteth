package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
)

func (s StateAnalyzer) runDownloadStates(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Requester")
	// loop over the list of slots that we need to analyze
	var prevBState spec.VersionedBeaconState // to be checked, it may make calculation easier to store previous state
	var bstate *spec.VersionedBeaconState
	var err error
	for _, slot := range s.SlotRanges {
		ticker := time.NewTicker(minReqTime)
		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing state requester routine")
			close(s.EpochTaskChan)
			return

		default:
			firstIteration := false
			// make the state query
			log.Infof("requesting Beacon State from endpoint: slot %d", slot)
			if bstate != nil { // in case we already had a bstate (only false the first time)
				prevBState = *bstate
			} else {
				firstIteration = true
			}
			snapshot := time.Now()
			bstate, err = s.cli.Api.BeaconState(s.ctx, fmt.Sprintf("%d", slot))
			s.MonitorMetrics.AddDownload(time.Since(snapshot).Seconds())
			if !firstIteration {
				// only execute tasks if it is not the first iteration
				if err != nil {
					// close the channel (to tell other routines to stop processing and end)
					log.Errorf("Unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())
					return
				}

				// we now only compose one single task that contains a list of validator indexes
				// compose the next task
				epochTask := &EpochTask{
					ValIdxs:   s.ValidatorIndexes,
					Slot:      slot,
					State:     bstate,
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
