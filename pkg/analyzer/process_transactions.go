package analyzer

import (
	"sync"
	"time"

	"github.com/migalabs/goteth/pkg/utils"
)

// process transactions and persist the data
func (s *ChainAnalyzer) runProcessTransactions(wgProcess *sync.WaitGroup) {
	defer wgProcess.Done()

	log.Info("Launching Beacon Block Transactions Processor")
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
	var wgTransactionWorkers sync.WaitGroup

	for i := 0; i < s.validatorWorkerNum; i++ {

		wgTransactionWorkers.Add(1)

		go func() {
			defer wgTransactionWorkers.Done()
		loop:
			for {
				// in case the downloads have finished, and there are no more tasks to execute

				select {

				case task, ok := <-s.transactionTaskChan:

					if !ok {
						log.Warn("the transactions task channel has closed, finishing transaction routine")
						return
					}
					log.Tracef("transaction task received for slot %d, analyzing...", task.Slot)

					detailedTx, err := s.cli.RequestTransactionDetails(task.Transaction, task.Slot, task.BlockNumber, task.BlockTimestamp)

					if err != nil {
						log.Errorf("could not request transaction details in slot %s", task.Slot, err)
					}
					log.Debugf("persisting transaction metrics: slot %d", task.Slot)
					s.dbClient.Persist(detailedTx)

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
		}()
	}

	wgTransactionWorkers.Wait()
	log.Infof("Block transactions processor routine finished...")
}
