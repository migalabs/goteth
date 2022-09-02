package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/cortze/eth2-state-analyzer/pkg/custom_spec"
	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql"
	"github.com/cortze/eth2-state-analyzer/pkg/model"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
	"github.com/jackc/pgx/v4"
)

func (s StateAnalyzer) runProcessState(wgProcess *sync.WaitGroup, downloadFinishedFlag *bool) {
	defer wgProcess.Done()

	epochBatch := pgx.Batch{}

	for {

		if *downloadFinishedFlag && len(s.EpochTaskChan) == 0 && epochBatch.Len() == 0 {
			log.Warn("the task channel has been closed, finishing epoch routine")
			return
		}

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing state processer routine")
			return

		case task, ok := <-s.EpochTaskChan:

			// check if the channel has been closed

			if !ok {
				log.Warn("the task channel has been closed, finishing epoch routine")
				return
			}
			log.Infof("epoch task received for slot %d, analyzing...", task.Slot)

			snapshot := time.Now()
			// returns the state in a custom struct for Phase0, Altair of Bellatrix
			customBState, err := custom_spec.BStateByForkVersion(task.State, task.PrevState, s.cli.Api)
			s.MonitorMetrics.AddPreprocessTime(time.Since(snapshot).Seconds())

			if err != nil {
				log.Errorf(err.Error())
			}

			log.Debugf("Creating validator batches for slot %d...", task.Slot)
			snapshot = time.Now()

			for i := range task.ValIdxs {
				valTask := &ValTask{
					ValIdxs:     task.ValIdxs[i : i+1],
					CustomState: customBState,
				}
				s.ValTaskChan <- valTask
			}

			s.MonitorMetrics.AddBatchingTime(time.Since(snapshot).Seconds())

			log.Debugf("Writing epoch metrics to DB for slot %d...", task.Slot)
			// create a model to be inserted into the db
			epochDBRow := model.NewEpochMetrics(
				customBState.CurrentEpoch(),
				customBState.CurrentSlot(),
				0,
				0,
				0,
				0,
				0,
				0,
				0,
				0,
				uint64(len(customBState.GetMissedBlocks())))

			epochBatch.Queue(model.InsertNewEpochLineTable,
				epochDBRow.Epoch,
				epochDBRow.Slot,
				epochDBRow.PrevNumAttestations,
				epochDBRow.PrevNumAttValidators,
				epochDBRow.PrevNumValidators,
				epochDBRow.TotalBalance,
				epochDBRow.TotalEffectiveBalance,
				epochDBRow.MissingSource,
				epochDBRow.MissingTarget,
				epochDBRow.MissingHead,
				epochDBRow.MissedBlocks)

			// err = s.dbClient.InsertNewEpochRow(epochDBRow)
			// if err != nil {
			// 	log.Errorf(err.Error())
			// }

			epochDBRow.PrevNumAttestations = customBState.GetAttNum()
			epochDBRow.PrevNumAttValidators = customBState.GetAttestingValNum()
			epochDBRow.PrevNumValidators = customBState.GetNumVals()
			epochDBRow.TotalBalance = customBState.GetTotalActiveBalance()
			epochDBRow.TotalEffectiveBalance = customBState.GetTotalActiveEffBalance()

			epochDBRow.MissingSource = customBState.GetMissingFlag(int(altair.TimelySourceFlagIndex))
			epochDBRow.MissingTarget = customBState.GetMissingFlag(int(altair.TimelyTargetFlagIndex))
			epochDBRow.MissingHead = customBState.GetMissingFlag(int(altair.TimelyHeadFlagIndex))

			// err = s.dbClient.UpdatePrevEpochMetrics(epochDBRow)
			// if err != nil {
			// 	log.Errorf(err.Error())
			// }

			epochBatch.Queue(model.UpdateRow,
				epochDBRow.Slot-utils.SlotBase,
				epochDBRow.PrevNumAttestations,
				epochDBRow.PrevNumAttValidators,
				epochDBRow.PrevNumValidators,
				epochDBRow.TotalBalance,
				epochDBRow.TotalEffectiveBalance,
				epochDBRow.MissingSource,
				epochDBRow.MissingTarget,
				epochDBRow.MissingHead)

			if epochBatch.Len() >= postgresql.MAX_BATCH_QUEUE || (*downloadFinishedFlag && len(s.EpochTaskChan) == 0) {
				s.dbClient.WriteChan <- epochBatch
				epochBatch = pgx.Batch{}

			}
		}

	}
}
