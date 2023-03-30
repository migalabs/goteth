package state

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/postgresql/model"
	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
)

func (s *StateAnalyzer) runWorker(wlog *logrus.Entry, wgWorkers *sync.WaitGroup, processFinishedFlag *bool) {
	defer wgWorkers.Done()
	batch := pgx.Batch{}
	// keep iterating until the channel is closed due to finishing
loop:
	for {

		if *processFinishedFlag && len(s.ValTaskChan) == 0 {
			wlog.Warn("the task channel has been closed, finishing worker routine")
			if batch.Len() > 0 {

				wlog.Debugf("Sending last validator batch to be stored...")
				s.dbClient.WriteChan <- batch
				batch = pgx.Batch{}

			}
			break loop
		}

		select {
		case valTask, ok := <-s.ValTaskChan:
			// check if the channel has been closed
			if !ok {
				wlog.Warn("the task channel has been closed, finishing worker routine")
				return
			}

			stateMetrics := valTask.StateMetricsObj
			wlog.Debugf("task received for val %d - %d in slot %d", valTask.ValIdxs[0], valTask.ValIdxs[len(valTask.ValIdxs)-1], stateMetrics.GetMetricsBase().CurrentState.Slot)
			// Proccess State
			snapshot := time.Now()

			// batch metrics
			summaryMet := NewSummaryMetrics()

			// process each validator
			for _, valIdx := range valTask.ValIdxs {

				if valIdx >= uint64(len(stateMetrics.GetMetricsBase().NextState.Validators)) {
					continue // validator is not in the chain yet
				}
				// get max reward at given epoch using the formulas
				maxRewards, err := stateMetrics.GetMaxReward(valIdx)

				if err != nil {
					log.Errorf("Error obtaining max reward: ", err.Error())
					continue
				}

				// calculate the current balance of validator
				balance := stateMetrics.GetMetricsBase().NextState.Balances[valIdx]

				// keep in mind that att rewards for epoch 10 can be seen at beginning of epoch 12,
				// after state_transition
				// https://notes.ethereum.org/@vbuterin/Sys3GLJbD#Epoch-processing

				// CurrentState Flags measure previous epoch flags
				flags := stateMetrics.GetMetricsBase().CurrentState.MissingFlags(valIdx)

				// create a model to be inserted into the db in the next epoch
				validatorDBRow := model.NewValidatorRewards(
					valIdx,
					stateMetrics.GetMetricsBase().NextState.Slot,
					stateMetrics.GetMetricsBase().NextState.Epoch,
					balance,
					stateMetrics.GetMetricsBase().EpochReward(valIdx), // reward is written after state transition
					maxRewards.MaxReward,
					maxRewards.Attestation,
					maxRewards.SyncCommittee,
					stateMetrics.GetMetricsBase().GetAttSlot(valIdx),
					stateMetrics.GetMetricsBase().GetAttInclusionSlot(valIdx),
					maxRewards.BaseReward,
					maxRewards.InSyncCommittee,
					maxRewards.ProposerSlot, // TODO: there can be several proposer slots, deprecate
					flags[altair.TimelySourceFlagIndex],
					flags[altair.TimelyTargetFlagIndex],
					flags[altair.TimelyHeadFlagIndex],
					stateMetrics.GetMetricsBase().NextState.GetValStatus(valIdx))

				if s.Metrics.Validator {
					batch.Queue(model.UpsertValidator,
						validatorDBRow.ValidatorIndex,
						validatorDBRow.Slot,
						validatorDBRow.Epoch,
						validatorDBRow.ValidatorBalance,
						validatorDBRow.Reward,
						validatorDBRow.MaxReward,
						validatorDBRow.AttestationReward,
						validatorDBRow.SyncCommitteeReward,
						validatorDBRow.AttSlot,
						validatorDBRow.BaseReward,
						validatorDBRow.InSyncCommittee,
						validatorDBRow.MissingSource,
						validatorDBRow.MissingTarget,
						validatorDBRow.MissingHead,
						validatorDBRow.Status)

					if batch.Len() > postgresql.MAX_BATCH_QUEUE {
						wlog.Debugf("Sending batch to be stored...")
						s.dbClient.WriteChan <- batch
						batch = pgx.Batch{}
					}
				}

				summaryMet.AddMetrics(maxRewards, stateMetrics, valIdx, validatorDBRow)

			}

			if s.Metrics.PoolSummary && valTask.PoolName != "" {
				// only send summary batch in case pools were introduced by the user and we have a name to identify it
				summaryMet.Aggregate()
				// create and send summary batch
				summaryBatch := pgx.Batch{}
				summaryBatch.Queue(model.UpsertPoolSummary,
					valTask.PoolName,
					stateMetrics.GetMetricsBase().NextState.Epoch,
					summaryMet.AvgReward,
					summaryMet.AvgMaxReward,
					summaryMet.AvgAttMaxReward,
					summaryMet.AvgSyncMaxReward,
					summaryMet.AvgBaseReward,
					summaryMet.MissingSourceCount,
					summaryMet.MissingTargetCount,
					summaryMet.MissingHeadCount,
					summaryMet.NumActiveVals,
					summaryMet.NumSyncVals)

				wlog.Debugf("Sending pool summary batch (%s) to be stored...", valTask.PoolName)
				s.dbClient.WriteChan <- summaryBatch
			}
			wlog.Debugf("Validator group processed, worker freed for next group. Took %f seconds", time.Since(snapshot).Seconds())

		case <-s.ctx.Done():
			log.Info("context has died, closing state worker routine")
			return
		default:
			// if there is something to persist and no more incoming tasks, flush validator batch
			if batch.Len() > 0 && len(s.ValTaskChan) == 0 {
				wlog.Debugf("Sending batch to be stored (no more tasks)...")
				s.dbClient.WriteChan <- batch
				batch = pgx.Batch{}
			}
		}

	}
	wlog.Infof("Validator worker finished, no more tasks to process")
}
