package analyzer

import (
	"math"
	"sync"

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
	log.Info("Launching Beacon State Pre-Processer")

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

			// snapshot := time.Now()
			// returns the state in a custom struct for Phase0, Altair of Bellatrix
			customBState, err := custom_spec.BStateByForkVersion(task.NextState, task.State, task.PrevState, s.cli.Api)
			// s.MonitorMetrics.AddPreprocessTime(time.Since(snapshot).Seconds())

			if err != nil {
				log.Errorf(err.Error())
				continue
			}

			log.Debugf("Creating validator batches for slot %d...", task.Slot)
			// snapshot = time.Now()
			if len(task.ValIdxs) == 0 {
				task.ValIdxs = customBState.GetPrevValList()
			}
			stepSize := int(math.Min(float64(MAX_VAL_BATCH_SIZE), float64(len(task.ValIdxs)/coworkers)))
			for i := 0; i < len(task.ValIdxs); i += stepSize {
				endIndex := int(math.Min(float64(len(task.ValIdxs)), float64(i+stepSize)))
				// subslice does not include the endIndex
				valTask := &ValTask{
					ValIdxs:     task.ValIdxs[i:endIndex],
					CustomState: customBState,
				}
				s.ValTaskChan <- valTask
			}

			// s.MonitorMetrics.AddBatchingTime(time.Since(snapshot).Seconds())

			log.Debugf("Writing epoch metrics to DB for slot %d...", task.Slot)
			// create a model to be inserted into the db
			epochDBRow := model.NewEpochMetrics(
				customBState.CurrentEpoch(),
				customBState.CurrentSlot(),
				0,
				0,
				customBState.GetNumVals(),
				0,
				0,
				0,
				0,
				0,
				0,
				customBState.GetMissedBlocks())

			epochBatch.Queue(model.InsertNewEpochLineTable,
				epochDBRow.Epoch,
				epochDBRow.Slot,
				epochDBRow.PrevNumAttestations,
				epochDBRow.PrevNumAttValidators,
				epochDBRow.PrevNumValidators,
				epochDBRow.TotalBalance,
				epochDBRow.AttEffectiveBalance,
				epochDBRow.TotalEffectiveBalance,
				epochDBRow.MissingSource,
				epochDBRow.MissingTarget,
				epochDBRow.MissingHead,
				epochDBRow.MissedBlocks)

			epochDBRow.PrevNumAttestations = int(customBState.GetAttNum())
			epochDBRow.PrevNumAttValidators = int(customBState.GetAttestingValNum())
			epochDBRow.TotalBalance = float32(customBState.GetTotalActiveBalance())
			epochDBRow.AttEffectiveBalance = float32(customBState.GetAttEffBalance())
			epochDBRow.TotalEffectiveBalance = float32(customBState.GetTotalActiveEffBalance())

			epochDBRow.MissingSource = int(customBState.GetMissingFlag(int(altair.TimelySourceFlagIndex)))
			epochDBRow.MissingTarget = int(customBState.GetMissingFlag(int(altair.TimelyTargetFlagIndex)))
			epochDBRow.MissingHead = int(customBState.GetMissingFlag(int(altair.TimelyHeadFlagIndex)))

			epochBatch.Queue(model.UpdateRow,
				epochDBRow.Slot-utils.SlotBase,
				epochDBRow.PrevNumAttestations,
				epochDBRow.PrevNumAttValidators,
				epochDBRow.TotalBalance,
				epochDBRow.AttEffectiveBalance,
				epochDBRow.TotalEffectiveBalance,
				epochDBRow.MissingSource,
				epochDBRow.MissingTarget,
				epochDBRow.MissingHead)

			if epochBatch.Len() >= postgresql.MAX_EPOCH_BATCH_QUEUE || (*downloadFinishedFlag && len(s.EpochTaskChan) == 0) {
				s.dbClient.WriteChan <- epochBatch
				epochBatch = pgx.Batch{}

			}
		}

	}
}
