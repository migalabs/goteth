package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

// This routine is able to download block by block in the slot range
func (s *BlockAnalyzer) runDownloadBlocks(wgDownload *sync.WaitGroup) {
	defer wgDownload.Done()
	log.Info("Launching Beacon Block Requester")
	// loop over the list of slots that we need to analyze
	for _, slot := range s.SlotRanges {

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing block requester routine")
			close(s.BlockTaskChan)
			close(s.TransactionTaskChan)
			return

		default:
			if s.finishDownload {
				log.Info("sudden shutdown detected, block downloader routine")
				close(s.BlockTaskChan)
				close(s.TransactionTaskChan)
				return
			}

			log.Infof("requesting Beacon Block from endpoint: slot %d", slot)
			err := s.DownloadNewBlock(phase0.Slot(slot))

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
	headSlot := phase0.Slot(0)
	header, err := s.cli.Api.BeaconBlockHeader(s.ctx, "head")
	if err != nil {
		log.Errorf("could not obtain current head to fill historical")
	} else {
		headSlot = header.Header.Message.Slot
	}

	// it means we could obtain both
	if lastRequestSlot > 0 && headSlot > 0 {

		for lastRequestSlot < (headSlot - 1) {
			lastRequestSlot = lastRequestSlot + 1

			log.Infof("filling missing blocks: %d", lastRequestSlot)

			err := s.DownloadNewBlock(phase0.Slot(lastRequestSlot))

			if err != nil {
				log.Errorf("error downloading block at slot %d: %s", lastRequestSlot, err)
			}
		}

	}

	// -----------------------------------------------------------------------------------
	s.eventsObj.SubscribeToHeadEvents()

	// loop over the list of slots that we need to analyze

	for {
		if s.finishDownload {
			log.Info("sudden shutdown detected, block downloader routine")
			close(s.BlockTaskChan)
			close(s.TransactionTaskChan)
			return
		}
		select {

		case <-s.ctx.Done():
			log.Info("context has died, closing block requester routine")
			close(s.BlockTaskChan)
			close(s.TransactionTaskChan)
			return

		case headSlot := <-s.eventsObj.HeadChan: // wait for new head event
			// make the block query
			log.Infof("received new head signal: %d", headSlot)

			if lastRequestSlot >= headSlot {
				log.Infof("No new head block yet")
				continue
			}
			if lastRequestSlot == 0 {
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

func (s BlockAnalyzer) RequestBeaconBlock(slot phase0.Slot) (spec.AgnosticBlock, bool, error) {
	newBlock, err := s.cli.Api.SignedBeaconBlock(s.ctx, fmt.Sprintf("%d", slot))
	if newBlock == nil {
		log.Warnf("the beacon block at slot %d does not exist, missing block", slot)
		return s.CreateMissingBlock(slot), false, nil
	}
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return spec.AgnosticBlock{}, false, fmt.Errorf("unable to retrieve Beacon Block at slot %d: %s", slot, err.Error())
	}

	customBlock, err := spec.GetCustomBlock(*newBlock)

	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return spec.AgnosticBlock{}, false, fmt.Errorf("unable to parse Beacon Block at slot %d: %s", slot, err.Error())
	}
	return customBlock, true, nil
}

func (s BlockAnalyzer) DownloadNewBlock(slot phase0.Slot) error {

	ticker := time.NewTicker(minBlockReqTime)
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

	// store transactions if it has been enabled
	if s.enableTransactions {
		transactionTask := &TransactionTask{
			Slot:         uint64(slot),
			Transactions: spec.RequestTransactionDetails(newBlock),
		}
		log.Debugf("sending a new tx task for slot %d", slot)
		s.TransactionTaskChan <- transactionTask
	} else {
		log.Info("Here is the skip..............................")
	}

	<-ticker.C
	// check if the min Request time has been completed (to avoid spaming the API)
	return nil
}
