package state

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/sirupsen/logrus"
)

func (s *StateAnalyzer) runWorker(wlog *logrus.Entry, wgWorkers *sync.WaitGroup, processFinishedFlag *bool) {
	defer wgWorkers.Done()
	// keep iterating until the channel is closed due to finishing
loop:
	for {

		if *processFinishedFlag && len(s.ValTaskChan) == 0 {
			wlog.Warn("the task channel has been closed, finishing worker routine")
			break loop
		}

		select {
		case valTask, ok := <-s.ValTaskChan:
			// check if the channel has been closed
			if !ok {
				wlog.Warn("the task channel has been closed, finishing worker routine")
				return
			}

			wlog.Debugf("task received for val %d - %d in epoch %d", valTask.ValIdxs[0], valTask.ValIdxs[len(valTask.ValIdxs)-1], valTask.StateMetricsObj.GetMetricsBase().CurrentState.Epoch)
			// Proccess State
			snapshot := time.Now()

			// batch metrics
			summaryMet := model.PoolSummary{
				PoolName: valTask.PoolName,
				Epoch:    valTask.StateMetricsObj.GetMetricsBase().NextState.Epoch,
			}

			// process each validator
			for _, valIdx := range valTask.ValIdxs {

				if int(valIdx) >= len(valTask.StateMetricsObj.GetMetricsBase().NextState.Validators) {
					continue // validator is not in the chain yet
				}
				// get max reward at given epoch using the formulas
				maxRewards, err := valTask.StateMetricsObj.GetMaxReward(valIdx)

				if err != nil {
					log.Errorf("Error obtaining max reward: ", err.Error())
					continue
				}

				if valTask.Finalized {
					// Only update validator last status on Finalized
					// We will always receive higher epochs
					s.dbClient.Persist(db.WriteTask{
						Model: model.ValidatorLastStatus{
							ValIdx:         phase0.ValidatorIndex(valIdx),
							Epoch:          valTask.StateMetricsObj.GetMetricsBase().CurrentState.Epoch,
							CurrentBalance: valTask.StateMetricsObj.GetMetricsBase().NextState.Balances[valIdx],
							CurrentStatus:  maxRewards.Status,
						},
						Op: model.INSERT_OP,
					})
				}
				if s.Metrics.ValidatorRewards { // only if flag is activated
					s.dbClient.Persist(db.WriteTask{
						Model: maxRewards,
						Op:    model.INSERT_OP,
					})
				}

				if s.Metrics.PoolSummary && valTask.PoolName != "" {
					summaryMet.AddValidator(maxRewards)
				}

			}

			if s.Metrics.PoolSummary && summaryMet.PoolName != "" {
				// only send summary batch in case pools were introduced by the user and we have a name to identify it

				wlog.Debugf("Sending pool summary batch (%s) to be stored...", summaryMet.PoolName)
				s.dbClient.Persist(db.WriteTask{
					Model: summaryMet,
					Op:    model.INSERT_OP,
				})
			}
			wlog.Debugf("Validator group processed, worker freed for next group. Took %f seconds", time.Since(snapshot).Seconds())

		case <-s.ctx.Done():
			log.Info("context has died, closing state worker routine")
			return
		}

	}
	wlog.Infof("Validator worker finished, no more tasks to process")
}
