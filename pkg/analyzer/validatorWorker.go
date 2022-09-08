package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/altair"
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
			snapshot := time.Now()
			for _, valIdx := range valTask.ValIdxs {

				// get max reward at given epoch using the formulas
				maxRewards, err := customBState.GetMaxReward(valIdx)

				if err != nil {
					log.Errorf("Error obtaining max reward: ", err.Error())
					continue
				}

				// calculate the current balance of validator
				balance, err := customBState.Balance(valIdx)

				if err != nil {
					log.Errorf("Error obtaining validator balance: ", err.Error())
					continue
				}

				flags := customBState.MissingFlags(valIdx)

				//TODO: Added specific flag missing support for validators
				// TODO: But pending for optimizations before further processing
				// create a model to be inserted into the db
				validatorDBRow := model.NewValidatorRewards(
					valIdx,
					customBState.CurrentSlot()+uint64(EPOCH_SLOTS),
					customBState.CurrentEpoch()+1,
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
					maxRewards.InSyncCommittee,
					float64(maxRewards.ProposerSlot),
					flags[altair.TimelySourceFlagIndex],
					flags[altair.TimelyTargetFlagIndex],
					flags[altair.TimelyHeadFlagIndex])

				batch.Queue(model.InsertNewValidatorLineTable,
					validatorDBRow.ValidatorIndex,
					validatorDBRow.Slot,
					validatorDBRow.Epoch,
					validatorDBRow.ValidatorBalance,
					validatorDBRow.Reward,
					validatorDBRow.MaxReward,
					validatorDBRow.AttSlot,
					validatorDBRow.InclusionDelay,
					validatorDBRow.BaseReward,
					validatorDBRow.InSyncCommittee,
					validatorDBRow.ProposerSlot,
					validatorDBRow.MissingSource,
					validatorDBRow.MissingTarget,
					validatorDBRow.MissingHead)

				if customBState.CurrentSlot() >= 63 {
					reward := customBState.PrevEpochReward(valIdx)

					// keep in mind that rewards for epoch 10 can be seen at beginning of epoch 12,
					// after state_transition
					// https://notes.ethereum.org/@vbuterin/Sys3GLJbD#Epoch-processing
					validatorDBRow = model.NewValidatorRewards(valIdx,
						customBState.CurrentSlot(),
						customBState.CurrentEpoch(),
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
						-1,
						false,
						false,
						false)

					batch.Queue(model.UpdateValidatorLineTable,
						validatorDBRow.ValidatorIndex,
						validatorDBRow.Slot,
						validatorDBRow.Reward)

					if batch.Len() > postgresql.MAX_BATCH_QUEUE || (*processFinishedFlag && len(s.ValTaskChan) == 0) {
						wlog.Debugf("Sending batch to be stored...")
						s.dbClient.WriteChan <- batch
						batch = pgx.Batch{}
					}
				}

			}

			wlog.Debugf("Validator group processed, worker freed for next group. Took %f seconds", time.Since(snapshot).Seconds())

		case <-s.ctx.Done():
			log.Info("context has died, closing state processer routine")
			return

		}

	}
}
