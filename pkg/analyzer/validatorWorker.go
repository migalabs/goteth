package analyzer

import (
	"sync"

	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql"
	"github.com/cortze/eth2-state-analyzer/pkg/model"
	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
)

func (s *StateAnalyzer) runWorker(wlog *logrus.Entry, wgWorkers *sync.WaitGroup, processFinishedFlag *bool) {
	defer wgWorkers.Done()
	batch := pgx.Batch{}
	// keep iterating until the channel is closed due to finishing
	for {

		if *processFinishedFlag && len(s.ValTaskChan) == 0 && batch.Len() == 0 {
			wlog.Warn("the task channel has been closed, finishing worker routine")
			return
		}

		select {
		case valTask, ok := <-s.ValTaskChan:
			// check if the channel has been closed
			if !ok {
				wlog.Warn("the task channel has been closed, finishing worker routine")
				return
			}

			customBState := valTask.CustomState
			wlog.Debugf("task received for val %d - %d in slot %d", valTask.ValIdxs[0], valTask.ValIdxs[len(valTask.ValIdxs)-1], valTask.CustomState.CurrentSlot())
			// Proccess State

			for _, valIdx := range valTask.ValIdxs {

				// get max reward at given epoch using the formulas
				maxRewards, err := customBState.GetMaxReward(valIdx)

				if err != nil {
					log.Errorf("Error obtaining max reward: ", err.Error())
				}

				// calculate the current balance of validator
				balance, err := customBState.Balance(valIdx)

				if err != nil {
					log.Errorf("Error obtaining validator balance: ", err.Error())
				}
				//TODO: Added specific flag missing support for validators
				// TODO: But pending for optimizations before further processing
				// create a model to be inserted into the db
				validatorDBRow := model.NewValidatorRewards(
					valIdx,
					customBState.CurrentSlot(),
					customBState.CurrentEpoch(),
					balance,
					0, // reward is written after state transition
					maxRewards.MaxReward,
					maxRewards.Attestation,
					maxRewards.InclusionDelay,
					maxRewards.FlagIndex,
					maxRewards.SyncCommittee,
					customBState.GetAttSlot(valIdx),
					customBState.GetAttInclusionSlot(valIdx),
					maxRewards.BaseReward,
					false,
					false,
					false)

				// err = s.dbClient.InsertNewValidatorRow(validatorDBRow)
				// if err != nil {
				// 	log.Errorf(err.Error())
				// }
				batch.Queue(model.InsertNewValidatorLineTable,
					validatorDBRow.ValidatorIndex,
					validatorDBRow.Slot,
					validatorDBRow.Epoch,
					validatorDBRow.ValidatorBalance,
					validatorDBRow.Reward,
					validatorDBRow.MaxReward,
					validatorDBRow.AttestationReward,
					validatorDBRow.InclusionDelayReward,
					validatorDBRow.FlagIndexReward,
					validatorDBRow.SyncCommitteeReward,
					validatorDBRow.AttSlot,
					validatorDBRow.InclusionDelay,
					validatorDBRow.BaseReward,
					validatorDBRow.MissingSource,
					validatorDBRow.MissingTarget,
					validatorDBRow.MissingHead)

				rewardSlot := int(customBState.PrevStateSlot())
				rewardEpoch := int(customBState.PrevStateEpoch())
				if rewardSlot >= 31 {
					reward := customBState.PrevEpochReward(valIdx)

					// log.Debugf("Slot %d Validator %d Reward: %d", rewardSlot, valIdx, reward)

					// keep in mind that rewards for epoch 10 can be seen at beginning of epoch 12,
					// after state_transition
					// https://notes.ethereum.org/@vbuterin/Sys3GLJbD#Epoch-processing
					validatorDBRow = model.NewValidatorRewards(valIdx,
						uint64(rewardSlot),
						uint64(rewardEpoch),
						0, // balance: was already filled in the last epoch
						reward,
						0, // maxReward: was already calculated in the previous epoch
						0,
						0,
						0,
						0,
						0,
						0,
						0,
						false,
						false,
						false)

					// err = s.dbClient.UpdateValidatorRowReward(validatorDBRow)
					// if err != nil {
					// 	log.Errorf(err.Error())
					// }
					batch.Queue(model.UpdateValidatorLineTable,
						validatorDBRow.ValidatorIndex,
						validatorDBRow.Slot,
						validatorDBRow.Reward)

					if batch.Len() > postgresql.MAX_BATCH_QUEUE || (*processFinishedFlag && len(s.ValTaskChan) == 0) {
						s.dbClient.WriteChan <- batch
						batch = pgx.Batch{}
					}
				}

			}

		case <-s.ctx.Done():
			log.Info("context has died, closing state processer routine")
			return

		}

	}
}
