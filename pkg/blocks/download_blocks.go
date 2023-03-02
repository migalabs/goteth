package blocks

import (
	"fmt"
	"sync"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/block_metrics/fork_block"
)

// This routine is able to download block by block in the slot range
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

			log.Infof("requesting Beacon Block from endpoint: slot %d", slot)
			err := s.DownloadNewBlock(int(slot))

			if err != nil {
				log.Errorf("error downloading block at slot %d: %s", slot, err)
			}

		}

	}

	log.Infof("All blocks for the slot ranges has been successfully retrieved, clossing go routine")
}

func (s *BlockAnalyzer) runDownloadBlocksFinalized(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon Block Finalized Requester")

	// ------ fill from last epoch in database to current head -------

	// obtain last epoch in database
	lastRequestSlot, err := s.dbClient.ObtainLastSlot()
	if err != nil {
		log.Errorf("could not obtain last slot in database: %s", err)
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
	if lastRequestSlot > 0 && headSlot > 0 {

		for (lastRequestSlot) < (headSlot - 1) {
			lastRequestSlot = lastRequestSlot + 1

			log.Infof("filling missing blocks: %d", lastRequestSlot)

			err := s.DownloadNewBlock(lastRequestSlot)

			if err != nil {
				log.Errorf("error downloading block at slot %d: %s", lastRequestSlot, err)
			}
		}

	}

	// -----------------------------------------------------------------------------------
	s.eventsObj.SubscribeToHeadEvents()

	// loop over the list of slots that we need to analyze

	for {

		select {
		default:
			if s.finishDownload {
				log.Info("sudden shutdown detected, block downloader routine")
				close(s.BlockTaskChan)
				return
			}
		case <-s.ctx.Done():
			log.Info("context has died, closing block requester routine")
			close(s.BlockTaskChan)
			return

		case headSlot := <-s.eventsObj.HeadChan: // wait for new head event
			// make the block query
			log.Infof("received new head signal: %d", headSlot)

			if lastRequestSlot >= headSlot {
				log.Infof("No new head block yet")
				continue
			}
			if lastRequestSlot < 0 {
				lastRequestSlot = headSlot
			}
			for lastRequestSlot < headSlot {
				lastRequestSlot = lastRequestSlot + 1
				err := s.DownloadNewBlock(lastRequestSlot)

				if err != nil {
					log.Errorf("error downloading block at slot %d: %s", lastRequestSlot, err)
				}

			}

		}

	}
}

func (s BlockAnalyzer) RequestBeaconBlock(slot int) (fork_block.ForkBlockContentBase, bool, error) {
	newBlock, err := s.cli.Api.SignedBeaconBlock(s.ctx, fmt.Sprintf("%d", slot))
	if newBlock == nil {
		log.Warnf("the beacon block at slot %d does not exist, missing block", slot)
		return s.CreateMissingBlock(slot), false, nil
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return fork_block.ForkBlockContentBase{}, false, fmt.Errorf("unable to retrieve Beacon Block at slot %d: %s", slot, err.Error())
	}

	customBlock, err := fork_block.GetCustomBlock(*newBlock)

	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return fork_block.ForkBlockContentBase{}, false, fmt.Errorf("unable to parse Beacon Block at slot %d: %s", slot, err.Error())
	}
	return customBlock, true, nil
}

func (s BlockAnalyzer) DownloadNewBlock(slot int) error {

	ticker := time.NewTicker(minReqTime)
	newBlock, proposed, err := s.RequestBeaconBlock(slot)
	if err != nil {
		return fmt.Errorf("block error at slot %d: %s", slot, err)
	}

	// send task to be processed
	blockTask := &BlockTask{
		Block:    newBlock,
		Slot:     uint64(slot),
		Proposed: proposed,
	}
	log.Debugf("sending a new task for slot %d", slot)
	s.BlockTaskChan <- blockTask

	<-ticker.C
	// check if the min Request time has been completed (to avoid spaming the API)
	return nil
}
