package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec/metrics"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

func (s *StateAnalyzer) runProcessState(wgProcess *sync.WaitGroup) {
	defer wgProcess.Done()
	log.Info("Launching Beacon State Pre-Processer")
	ticker := time.NewTicker(utils.RoutineFlushTimeout)
loop:
	for {

		select {

		case task := <-s.epochTaskChan:

			log.Infof("epoch task received for slot %d, epoch: %d, analyzing...", task.State.Slot, task.State.Epoch)

			// returns the state in a custom struct for Phase0, Altair of Bellatrix
			stateMetrics, err := metrics.StateMetricsByForkVersion(task.NextState, task.State, task.PrevState, s.cli.Api)

			if err != nil {
				log.Errorf(err.Error())
				continue
			}
			if task.NextState.Slot <= s.finalSlot || task.Finalized {
				log.Debugf("Creating validator batches for slot %d...", task.State.Slot)
				// divide number of validators into number of workers equally

				var validatorBatches []utils.PoolKeys

				// first of all see if there user input any validator list
				// in case no validators provided, do all the existing ones in the next epoch
				valIdxs := stateMetrics.GetMetricsBase().NextState.GetAllVals()
				validatorBatches = utils.DivideValidatorsBatches(valIdxs, s.validatorWorkerNum)

				if len(s.poolValidators) > 0 { // in case the user introduces custom pools
					validatorBatches = s.poolValidators

					valMatrix := make([][]phase0.ValidatorIndex, len(validatorBatches))

					for i, item := range validatorBatches {
						valMatrix[i] = make([]phase0.ValidatorIndex, 0)
						valMatrix[i] = item.ValIdxs
					}

					if s.missingVals {
						othersMissingList := utils.ObtainMissing(len(valIdxs), valMatrix)
						// now allValList should contain those validators that do not belong to any pool
						// keep track of those in a separate pool
						validatorBatches = utils.AddOthersPool(validatorBatches, othersMissingList)
					}

				}

				for _, item := range validatorBatches {
					valTask := &ValTask{
						ValIdxs:         item.ValIdxs,
						StateMetricsObj: stateMetrics,
						PoolName:        item.PoolName,
						Finalized:       task.Finalized,
					}
					s.valTaskChan <- valTask
				}
			}
			if task.PrevState.Slot >= s.initSlot || task.Finalized { // only write epoch metrics inside the defined range

				log.Debugf("Writing epoch metrics to DB for slot %d...", task.State.Slot)
				// create a model to be inserted into the db, we only insert previous epoch metrics

				missedBlocks := stateMetrics.GetMetricsBase().CurrentState.MissedBlocks
				// take into accoutn epoch transition
				nextMissedBlock := stateMetrics.GetMetricsBase().NextState.TrackPrevMissingBlock()
				if nextMissedBlock != 0 {
					missedBlocks = append(missedBlocks, nextMissedBlock)
				}

				// TODO: send constructor to model package
				epochModel := stateMetrics.GetMetricsBase().ExportToEpoch()

				s.dbClient.Persist(epochModel)

				// Proposer Duties

				for _, item := range stateMetrics.GetMetricsBase().CurrentState.EpochStructs.ProposerDuties {

					newDuty := spec.ProposerDuty{
						ValIdx:       item.ValidatorIndex,
						ProposerSlot: item.Slot,
						Proposed:     true,
					}
					for _, item := range missedBlocks {
						if newDuty.ProposerSlot == item { // we found the proposer slot in the missed blocks
							newDuty.Proposed = false
						}
					}
					s.dbClient.Persist(newDuty)
				}

				// Process Blocks
				// send task to be processed
				for _, block := range stateMetrics.GetMetricsBase().CurrentState.BlockList {

					s.dbClient.Persist(block)

					for _, item := range block.ExecutionPayload.Withdrawals {
						s.dbClient.Persist(spec.Withdrawal{
							Slot:           block.Slot,
							Index:          item.Index,
							ValidatorIndex: item.ValidatorIndex,
							Address:        item.Address,
							Amount:         item.Amount,
						})

					}

					// store transactions if it has been enabled
					if s.metrics.Transaction {

						for _, tx := range spec.RequestTransactionDetails(block) {
							s.dbClient.Persist(tx)
						}
					}
				}

			}
		case <-ticker.C:
			// in case the downloads have finished, and there are no more tasks to execute
			if s.downloadFinished && len(s.epochTaskChan) == 0 {
				break loop
			}
		case <-s.ctx.Done():
			break loop
		}

	}
	log.Infof("Pre process routine finished...")
}
