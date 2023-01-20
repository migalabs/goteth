package blocks

import (
	"sync"

	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/jackc/pgx/v4"
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
				task.Block.ExecutionPayload.Timestamp,
				task.Slot/uint64(EPOCH_SLOTS),
				task.Slot,
				task.Block.Graffiti,
				task.Block.ProposerIndex,
				task.Proposed,
				task.Block.Attestations,
				task.Block.Deposits,
				task.Block.ProposerSlashings,
				task.Block.AttesterSlashings,
				task.Block.VoluntaryExits,
				task.Block.SyncAggregate,
				task.Block.ExecutionPayload.FeeRecipient,
				task.Block.ExecutionPayload.GasLimit,
				task.Block.ExecutionPayload.GasUsed,
				task.Block.ExecutionPayload.BaseFeePerGas,
				task.Block.ExecutionPayload.BlockHash,
				task.Block.ExecutionPayload.Transactions)

			blockBatch.Queue(model.UpsertBlock,
				blockMetrics.ELTimestamp,
				blockMetrics.Epoch,
				blockMetrics.Slot,
				blockMetrics.Graffiti,
				blockMetrics.ProposerIndex,
				blockMetrics.Proposed,
				blockMetrics.Attestatons,
				blockMetrics.Deposits,
				blockMetrics.ProposerSlashings,
				blockMetrics.AttSlashings,
				blockMetrics.VoluntaryExits,
				blockMetrics.SyncBits,
				blockMetrics.ELFeeRecp,
				blockMetrics.ELGasLimit,
				blockMetrics.ELGasUsed,
				blockMetrics.ELBaseFeePerGas,
				blockMetrics.ELBlockHash,
				blockMetrics.ELTransactions)
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
