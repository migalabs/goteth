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
			for _, valIdx := range valTask.ValIdxs {

				// get max reward at given epoch using the formulas
				maxRewards, err := stateMetrics.GetMaxReward(valIdx)

				if err != nil {
					log.Errorf("Error obtaining max reward: ", err.Error())
					continue
				}

				// calculate the current balance of validator
				balance := stateMetrics.GetMetricsBase().NextState.Balances[valIdx]

				if err != nil {
					log.Errorf("Error obtaining validator balance: ", err.Error())
					continue
				}

				// keep in mind that att rewards for epoch 10 can be seen at beginning of epoch 12,
				// after state_transition
				// https://notes.ethereum.org/@vbuterin/Sys3GLJbD#Epoch-processing

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
					maxRewards.InclusionDelay,
					maxRewards.FlagIndex,
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

				batch.Queue(model.UpsertValidator,
					validatorDBRow.ValidatorIndex,
					validatorDBRow.Slot,
					validatorDBRow.Epoch,
					validatorDBRow.ValidatorBalance,
					validatorDBRow.Reward,
					validatorDBRow.MaxReward,
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

			wlog.Debugf("Validator group processed, worker freed for next group. Took %f seconds", time.Since(snapshot).Seconds())

		case <-s.ctx.Done():
			log.Info("context has died, closing state worker routine")
			return
		default:
		}

	}
	wlog.Infof("Validator worker finished, no more tasks to process")
}
