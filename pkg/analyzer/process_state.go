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

			log.Infof("epoch task received for slot %d, epoch: %d, analyzing...", task.ThirdState.Slot, task.ThirdState.Epoch)

			// returns the state in a custom struct for Phase0, Altair of Bellatrix
			stateMetrics, err := metrics.StateMetricsByForkVersion(task.FirstState, task.SecondState, task.ThirdState, task.FourthState, s.cli.Api)

			if err != nil {
				log.Errorf(err.Error())
				continue
			}
			log.Debugf("Creating validator batches for slot %d...", task.ThirdState.Slot)
			// divide number of validators into number of workers equally

			var validatorBatches []utils.PoolKeys

			// first of all see if there user input any validator list
			// in case no validators provided, do all the existing ones in the next epoch
			valIdxs := stateMetrics.GetMetricsBase().ThirdState.GetAllVals()
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

			s.PersistEpochData(stateMetrics)
			s.PersistBlockData(stateMetrics)

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

func (s *StateAnalyzer) PersistEpochData(stateMetrics metrics.StateMetrics) {
	log.Debugf("Writing epoch metrics to DB for epoch %d...", stateMetrics.GetMetricsBase().ThirdState.Epoch)
	missedBlocks := stateMetrics.GetMetricsBase().ThirdState.GetMissingBlocks()

	epochModel := stateMetrics.GetMetricsBase().ExportToEpoch()

	s.dbClient.Persist(epochModel)

	// Proposer Duties

	// TODO: this should be done by the statemetrics directly
	for _, item := range stateMetrics.GetMetricsBase().ThirdState.EpochStructs.ProposerDuties {

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
}

func (s *StateAnalyzer) PersistBlockData(stateMetrics metrics.StateMetrics) {
	for _, block := range stateMetrics.GetMetricsBase().ThirdState.BlockList {

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
