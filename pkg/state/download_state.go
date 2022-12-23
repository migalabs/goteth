package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/cortze/eth2-state-analyzer/pkg/state_metrics/fork_state"
)

func (s *StateAnalyzer) runDownloadStates(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon State Requester")
	// loop over the list of slots that we need to analyze

	// We need three consecutive states to compute max rewards easier
	prevBState := fork_state.ForkStateContentBase{}
	bstate := fork_state.ForkStateContentBase{}
	nextBstate := fork_state.ForkStateContentBase{}
	ticker := time.NewTicker(minReqTime)
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
			ticker.Reset(minReqTime)
			log.Infof("requesting Beacon State from endpoint: slot %d", slot)

			// We need three states to calculate both, rewards and maxRewards

			if bstate.AttestingBalance != nil { // in case we already had a bstate
				prevBState = bstate
			}
			if nextBstate.AttestingBalance != nil { // in case we already had a nextBstate
				bstate = nextBstate
			}
			newState, err := s.RequestBeaconState(int(slot))
			if err != nil {
				// close the channel (to tell other routines to stop processing and end)
				log.Errorf("unable to retrieve beacon state from the beacon node, closing requester routine. %s", err.Error())
				return
			}
			nextBstate, err = fork_state.GetCustomState(*newState, s.cli.Api)
			if err != nil {
				// close the channel (to tell other routines to stop processing and end)
				log.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
				return
			}

			if prevBState.AttestingBalance != nil {
				// only execute tasks if prevBState is something (we have state and nextState in this case)

				epochTask := &EpochTask{
					ValIdxs:   s.ValidatorIndexes,
					NextState: nextBstate,
					State:     bstate,
					PrevState: prevBState,
					Finalized: false,
				}

				log.Debugf("sending task for slot: %d", epochTask.State.Slot)
				s.EpochTaskChan <- epochTask
			}
			// check if the min Request time has been completed (to avoid spaming the API)
			<-ticker.C

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
	finalizedSlot := 0
	// tick every epoch, 384 seconds
	epochTicker := time.After(384 * time.Second)
	ticker := time.NewTicker(minReqTime)
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

		case <-epochTicker:
			epochTicker = time.After(384 * time.Second)
			ticker.Reset(minReqTime)
			log.Infof("requesting Beacon State from endpoint: finalized")
			if bstate.AttestingBalance != nil { // in case we already had a bstate
				prevBState = bstate
			}
			if nextBstate.AttestingBalance != nil { // in case we already had a nextBstate
				bstate = nextBstate
			}
			header, err := s.cli.Api.BeaconBlockHeader(s.ctx, "finalized")
			if err != nil {
				log.Errorf("Unable to retrieve Beacon State from the beacon node, closing finalized requester routine. %s", err.Error())
				continue
			}
			if int(header.Header.Message.Slot) == finalizedSlot {
				log.Infof("No new finalized state yet")
				continue
			}

			finalizedSlot = int(header.Header.Message.Slot) - 1
			log.Infof("New finalized state at slot: %d", finalizedSlot)
			newState, err := s.RequestBeaconState(finalizedSlot)
			if err != nil {
				log.Errorf("Unable to retrieve Finalized Beacon State from the beacon node. %s", err.Error())
				continue
			}

			nextBstate, err = fork_state.GetCustomState(*newState, s.cli.Api)
			if err != nil {
				log.Errorf("Unable to retrieve Beacon State from the beacon node, closing requester routine. %s", err.Error())
				continue
			}

			if prevBState.AttestingBalance != nil {
				epochTask := &EpochTask{
					ValIdxs:   s.ValidatorIndexes,
					NextState: nextBstate,
					State:     bstate,
					PrevState: prevBState,
					Finalized: true,
				}

				log.Debugf("sending task for slot: %d", epochTask.State.Slot)
				s.EpochTaskChan <- epochTask
			}
			<-ticker.C
			// check if the min Request time has been completed (to avoid spaming the API)

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
