package blocks

import (
	"sync"

	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql"
	"github.com/jackc/pgx/v4"
)

func (s *BlockAnalyzer) runProcessBlock(wgProcess *sync.WaitGroup, downloadFinishedFlag *bool) {
	defer wgProcess.Done()

	blockBatch := pgx.Batch{}
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

			// Flush the database batches
			if blockBatch.Len() >= postgresql.MAX_EPOCH_BATCH_QUEUE {
				s.dbClient.WriteChan <- blockBatch
				blockBatch = pgx.Batch{}
			}
		default:
		}

	}
	log.Infof("Pre process routine finished...")
}
