package blocks

import (
	"fmt"
	"sync"
	"time"

	"github.com/cortze/eth2-state-analyzer/pkg/block_metrics/fork_block"
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

			// make the state query
			newBlock, err := s.cli.Api.SignedBeaconBlock(s.ctx, fmt.Sprintf("%d", slot))

			if newBlock == nil { // if the block was not found, no error and nil block
				log.Errorf("the Beacon Block is unavailable, nil block")
				blockTask := &BlockTask{
					Block: fork_block.ForkBlockContentBase{
						Slot:          uint64(slot),
						ProposerIndex: 0,
						Graffiti:      []byte(""),
						Attestations:  nil,
						Deposits:      nil,
					},
					Slot:     uint64(slot),
					Proposed: false,
				}
				log.Debugf("sending a new missed task for slot %d", slot)
				s.BlockTaskChan <- blockTask
				continue

			}
			if err != nil {
				// close the channel (to tell other routines to stop processing and end)
				log.Errorf("unable to retrieve the beacon block at slot %d. %s", slot, err.Error())
				return
			}

			customBlock, err := fork_block.GetCustomBlock(*newBlock, s.cli.Api)
			if err != nil {
				log.Errorf("could not determine the fork of block %d", slot, err)
			}

			// send task to be processed
			blockTask := &BlockTask{
				Block:    customBlock,
				Slot:     slot,
				Proposed: true,
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
				// send task to be processed
				blockTask := &BlockTask{
					Block: fork_block.ForkBlockContentBase{
						Slot:          uint64(finalizedSlot),
						ProposerIndex: 0,
						Graffiti:      []byte(""),
						Attestations:  nil,
						Deposits:      nil,
					},
					Slot:     uint64(finalizedSlot),
					Proposed: false,
				}
				log.Debugf("sending a new missed task for slot %d", finalizedSlot)
				s.BlockTaskChan <- blockTask
				continue
			}
			if err != nil {
				// close the channel (to tell other routines to stop processing and end)
				log.Errorf("unable to retrieve the beacon block at slot %d. %s", finalizedSlot, err.Error())
				return
			}

			customBlock, err := fork_block.GetCustomBlock(*newBlock, s.cli.Api)
			if err != nil {
				log.Errorf("could not determine the fork of block %d", finalizedSlot, err)
			}

			// send task to be processed
			blockTask := &BlockTask{
				Block:    customBlock,
				Slot:     uint64(finalizedSlot),
				Proposed: true,
			}
			log.Debugf("sending a new task for slot %d", finalizedSlot)
			s.BlockTaskChan <- blockTask

			<-ticker.C
			// check if the min Request time has been completed (to avoid spaming the API)
		default:

		}

	}
}
