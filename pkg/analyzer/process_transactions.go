package analyzer

import (
	"sync"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

// process transactions and persist the data
func (s *BlockAnalyzer) runProcessTransactions(wgProcess *sync.WaitGroup) {
	defer wgProcess.Done()

	log.Info("Launching Beacon Block Transactions Processor")
	ticker := time.NewTicker(utils.RoutineFlushTimeout)

loop:
	for {
		// in case the downloads have finished, and there are no more tasks to execute

		select {

		case task, ok := <-s.transactionTaskChan:

			if !ok {
				log.Warn("the transactions task channel has closed, finishing transaction routine")
				return
			}
			log.Infof("transaction task received for slot %d, analyzing...", task.Slot)

			for _, tx := range task.Transactions {
				s.dbClient.Persist(tx)
			}
		case <-ticker.C:
			// in case the block processer has finished, and there are no more tasks to execute
			if s.processerFinished && len(s.transactionTaskChan) == 0 {
				log.Warn("the transactions task channel has been closed, finishing transaction routine")
				break loop
			}
		case <-s.ctx.Done():
			log.Info("context has died, closing block processer routine")
			break loop
		}
	}

	log.Infof("Block transactions processor routine finished...")
}
