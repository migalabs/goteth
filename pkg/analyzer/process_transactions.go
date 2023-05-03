package analyzer

import (
	"sync"
)

// process transactions and persist the data
func (s *BlockAnalyzer) runProcessTransactions(wgProcess *sync.WaitGroup, downloadFinishedFlag *bool) {
	defer wgProcess.Done()

	log.Info("Launching Beacon Block Transactions Processor")

loop:
	for {
		// in case the downloads have finished, and there are no more tasks to execute
		if *downloadFinishedFlag && len(s.TransactionTaskChan) == 0 {
			log.Warn("the transactions task channel has been closed, finishing transaction routine")
			break loop
		}

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing transaction processor routine")
			break loop

		case task, ok := <-s.TransactionTaskChan:

			if !ok {
				log.Warn("the transactions task channel has closed, finishing transaction routine")
				return
			}
			log.Infof("transaction task received for slot %d, analyzing...", task.Slot)

			for _, tx := range task.Transactions {
				s.dbClient.Persist(tx)
			}
		}

	}
	log.Infof("Block transactions processor routine finished...")
}
