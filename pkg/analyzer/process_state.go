package analyzer

import (
	"math"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/cortze/eth2-state-analyzer/pkg/custom_spec"
	"github.com/cortze/eth2-state-analyzer/pkg/db/postgresql"
	"github.com/cortze/eth2-state-analyzer/pkg/model"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
	"github.com/jackc/pgx/v4"
)

func (s *StateAnalyzer) runProcessState(wgProcess *sync.WaitGroup, downloadFinishedFlag *bool, coworkers int) {
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
			if len(task.ValIdxs) == 0 {
				task.ValIdxs = customBState.GetValList()
			}
			stepSize := int(math.Max(float64(len(task.ValIdxs)/MAX_VAL_BATCHES), 1))
			for i := 0; i < len(task.ValIdxs); i += stepSize {
				endIndex := int(math.Min(float64(len(task.ValIdxs)), float64(i+stepSize)))
				// subslice does not include the endIndex
				valTask := &ValTask{
					ValIdxs:     task.ValIdxs[i:endIndex],
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

			epochDBRow.PrevNumAttestations = customBState.GetAttNum()
			epochDBRow.PrevNumAttValidators = customBState.GetAttestingValNum()
			epochDBRow.PrevNumValidators = customBState.GetNumVals()
			epochDBRow.TotalBalance = customBState.GetTotalActiveBalance()
			epochDBRow.TotalEffectiveBalance = customBState.GetTotalActiveEffBalance()

			epochDBRow.MissingSource = customBState.GetMissingFlag(int(altair.TimelySourceFlagIndex))
			epochDBRow.MissingTarget = customBState.GetMissingFlag(int(altair.TimelyTargetFlagIndex))
			epochDBRow.MissingHead = customBState.GetMissingFlag(int(altair.TimelyHeadFlagIndex))

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
