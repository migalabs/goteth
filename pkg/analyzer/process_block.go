package analyzer

import (
	"sync"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
)

func (s *BlockAnalyzer) runProcessBlock(wgProcess *sync.WaitGroup, downloadFinishedFlag *bool) {
	defer wgProcess.Done()

	log.Info("Launching Beacon Block Processor")
loop:
	for {
		// in case the downloads have finished, and there are no more tasks to execute
		if *downloadFinishedFlag && len(s.BlockTaskChan) == 0 {
			log.Warn("the task channel has been closed, finishing block routine")
			break loop
		}

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing block processer routine")
			break loop

		case task, ok := <-s.BlockTaskChan:

			// check if the channel has been closed
			if !ok {
				log.Warn("the task channel has been closed, finishing block routine")
				return
			}
			log.Infof("block task received for slot %d, analyzing...", task.Slot)

			s.dbClient.Persist(task.Block)

			for _, item := range task.Block.ExecutionPayload.Withdrawals {
				s.dbClient.Persist(spec.Withdrawal{
					Slot:           task.Block.Slot,
					Index:          item.Index,
					ValidatorIndex: item.ValidatorIndex,
					Address:        item.Address,
					Amount:         item.Amount,
				})

			}
		}

	}
	log.Infof("Block process routine finished...")
}
