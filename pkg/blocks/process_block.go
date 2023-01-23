package blocks

import (
	"sync"

	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/jackc/pgx/v4"
)

var (
	FORK_CHOICE_SLOTS = uint64(64)
)

func (s *BlockAnalyzer) runProcessBlock(wgProcess *sync.WaitGroup, downloadFinishedFlag *bool) {
	defer wgProcess.Done()

	blockBatch := pgx.Batch{} // we will place all the database queries here
	log.Info("Launching Beacon Block Processor")
loop:
	for {
		// in case the downloads have finished, and there are no more tasks to execute
		if *downloadFinishedFlag && len(s.BlockTaskChan) == 0 {
			log.Warn("the task channel has been closed, finishing block routine")
			if blockBatch.Len() == 0 {
				log.Debugf("Sending last block batch to be stored...")
				s.dbClient.WriteChan <- blockBatch
				blockBatch = pgx.Batch{}
			}

			break loop
		}

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing block processer routine")
			return

		case task, ok := <-s.BlockTaskChan:

			// check if the channel has been closed
			if !ok {
				log.Warn("the task channel has been closed, finishing block routine")
				return
			}
			log.Infof("block task received for slot %d, analyzing...", task.Slot)

			blockMetrics := model.NewBlockMetrics(
				task.Slot/uint64(EPOCH_SLOTS),
				task.Slot,
				task.Block.Graffiti,
				task.Block.ProposerIndex,
				task.Proposed)

			if task.Slot > (s.HeadSlot - FORK_CHOICE_SLOTS) { // if we are close to head, then a correction might happen

				_, proposed, err := s.RequestBeaconBlock(int(task.Slot - FORK_CHOICE_SLOTS))
				if !proposed && err == nil {
					blockBatch.Queue(model.UpdateBlock,
						task.Slot-FORK_CHOICE_SLOTS,
						proposed)
				}
			}
			blockBatch.Queue(model.UpsertBlock,
				blockMetrics.Epoch,
				blockMetrics.Slot,
				blockMetrics.Graffiti,
				blockMetrics.ProposerIndex,
				blockMetrics.Proposed)
			// Flush the database batches
			if blockBatch.Len() >= postgresql.MAX_EPOCH_BATCH_QUEUE {
				s.dbClient.WriteChan <- blockBatch
				blockBatch = pgx.Batch{}
			}
		default:
		}

	}
	log.Infof("Block process routine finished...")
}
