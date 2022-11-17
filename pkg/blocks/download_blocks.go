package blocks

import (
	"fmt"
	"sync"
	"time"
)

func (s *BlockAnalyzer) runDownloadBlocks(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon Block Requester")
	// loop over the list of slots that we need to analyze

	ticker := time.NewTicker(minReqTime)
	for _, slot := range s.SlotRanges {

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing block requester routine")
			close(s.BlockTaskChan)
			return

		default:
			if s.finishDownload {
				log.Info("sudden shutdown detected, block downloader routine")
				close(s.BlockTaskChan)
				return
			}
			ticker.Reset(minReqTime)

			// make the state query
			log.Infof("requesting Beacon Block from endpoint: slot %d", slot)

			// We need three states to calculate both, rewards and maxRewards

			newBlock, err := s.cli.Api.SignedBeaconBlock(s.ctx, fmt.Sprintf("%d", slot))

			if newBlock == nil {
				log.Errorf("the Beacon Block is unavailable, nil block")
				return
			}
			if err != nil {
				// close the channel (to tell other routines to stop processing and end)
				log.Errorf("unable to retrieve the beacon block at slot %d. %s", slot, err.Error())
				return
			}

			blockTask := &BlockTask{
				Block: *newBlock,
				Slot:  slot,
			}
			log.Debugf("sending a new task for block %d", slot)
			s.BlockTaskChan <- blockTask

			// check if the min Request time has been completed (to avoid spaming the API)
			<-ticker.C

		}

	}

	log.Infof("All blocks for the slot ranges has been successfully retrieved, clossing go routine")
}

func (s *BlockAnalyzer) runDownloadBlocksFinalized(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon Block Finalized Requester")
	// loop over the list of slots that we need to analyze

	finalizedSlot := 0
	timerCh := time.NewTicker(time.Second * 12) // each slot = 12 seconds
	ticker := time.NewTicker(minReqTime)
	for {

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing block requester routine")
			close(s.BlockTaskChan)
			return

		case <-timerCh.C:
			ticker.Reset(minReqTime)

			// make the block query
			log.Infof("requesting Beacon State from endpoint: finalized")

			header, err := s.cli.Api.BeaconBlockHeader(s.ctx, "finalized")
			if err != nil {
				log.Errorf("Unable to retrieve Beacon State from the beacon node, closing finalized requester routine. %s", err.Error())
				return
			}
			if int(header.Header.Message.Slot) == finalizedSlot {
				log.Infof("No new finalized state yet")
				continue
			}

			finalizedSlot = int(header.Header.Message.Slot)
			log.Infof("New finalized state at slot: %d", finalizedSlot)
			newBlock, err := s.cli.Api.SignedBeaconBlock(s.ctx, fmt.Sprintf("%d", finalizedSlot))
			if newBlock == nil {
				log.Errorf("the Beacon Block is unavailable, nil block")
				return
			}
			if err != nil {
				// close the channel (to tell other routines to stop processing and end)
				log.Errorf("unable to retrieve the beacon block at slot %d. %s", finalizedSlot, err.Error())
				return
			}

			blockTask := &BlockTask{
				Block: *newBlock,
				Slot:  uint64(finalizedSlot),
			}
			log.Debugf("sending a new task for slot %d", finalizedSlot)
			s.BlockTaskChan <- blockTask

			<-ticker.C
			// check if the min Request time has been completed (to avoid spaming the API)
		default:

		}

	}
}
