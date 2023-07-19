package analyzer

import (
	"sync"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

func (s *ChainAnalyzer) runProcessBlock(wgProcess *sync.WaitGroup) {
	defer wgProcess.Done()

	log.Info("Launching Beacon Block Processor")
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
loop:
	for {

		select {

		case task, ok := <-s.blockTaskChan:

			// check if the channel has been closed
			if !ok {
				log.Warn("the task channel has been closed, finishing block routine")
				return
			}
			log.Tracef("block task received for slot %d, analyzing...", task.Slot)

			log.Debugf("persisting block metrics: slot %d", task.Block.Slot)
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

		case <-ticker.C:
			// in case the downloads have finished, and there are no more tasks to execute
			if s.downloadFinished && len(s.blockTaskChan) == 0 {
				log.Warn("the task channel has been closed, finishing block routine")
				break loop
			}
		case <-s.ctx.Done():
			log.Info("context has died, closing block processer routine")
			break loop
		}

	}
	log.Infof("Block process routine finished...")
}
