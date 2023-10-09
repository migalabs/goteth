package analyzer

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/spec/metrics"
	"github.com/pkg/errors"
)

// We always provide the epoch we transition to
// To process the transition from epoch 9 to 10, we provide 10 and we retrieve 8, 9, 10
func (s ChainAnalyzer) ProcessStateTransitionMetrics(epoch phase0.Epoch) error {

	if !s.metrics.Epoch {
		return nil
	}

	// Retrieve states to process metrics

	prevState := spec.AgnosticState{}
	currentState := spec.AgnosticState{}
	nextState := spec.AgnosticState{}

	if epoch-2 >= 0 {
		prevState = s.queue.StateHistory.Wait(epoch - 2)
	}
	if epoch-1 >= 0 {
		currentState = s.queue.StateHistory.Wait(epoch - 1)
	}
	if epoch >= 0 {
		nextState = s.queue.StateHistory.Wait(epoch)
	}

	bundle, err := metrics.StateMetricsByForkVersion(nextState, currentState, prevState, s.cli.Api)
	if err != nil {
		return errors.Wrap(err, "could not parse bundle metrics at epoch")
	}

	// For Epoch metrics we only need current and nextState
	emptyRoot := phase0.Root{}

	// If nextState is filled, we can process proposer duties
	if nextState.StateRoot != emptyRoot {
		s.processEpochDuties(bundle)
		s.processValLastStatus(bundle)

		// If currentState and nextState are filled, we can process epoch metrics
		if currentState.StateRoot != emptyRoot {
			s.processEpochMetrics(bundle)

			// If prevState, currentState and nextState are filled, we can process validator rewards
			if prevState.StateRoot != emptyRoot && s.metrics.ValidatorRewards {
				s.processEpochValRewards(bundle)
			}
		}
	}

	return nil

}

func (s *ChainAnalyzer) processEpochMetrics(bundle metrics.StateMetrics) {
	// we need sameEpoch and nextEpoch

	epochModel := bundle.GetMetricsBase().ExportToEpoch()

	log.Debugf("persisting epoch metrics: epoch %d", epochModel.Epoch)
	s.dbClient.Persist(epochModel)

}

func (s *ChainAnalyzer) processEpochDuties(bundle metrics.StateMetrics) {

	missedBlocks := bundle.GetMetricsBase().NextState.MissedBlocks

	for _, item := range bundle.GetMetricsBase().NextState.EpochStructs.ProposerDuties {

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

func (s *ChainAnalyzer) processValLastStatus(bundle metrics.StateMetrics) {

	if s.downloadMode == "finalized" {
		for valIdx, validator := range bundle.GetMetricsBase().NextState.Validators {

			s.dbClient.Persist(spec.ValidatorLastStatus{
				ValIdx:          phase0.ValidatorIndex(valIdx),
				Epoch:           bundle.GetMetricsBase().NextState.Epoch,
				CurrentBalance:  bundle.GetMetricsBase().NextState.Balances[valIdx],
				CurrentStatus:   bundle.GetMetricsBase().NextState.GetValStatus(phase0.ValidatorIndex(valIdx)),
				Slashed:         validator.Slashed,
				ActivationEpoch: validator.ActivationEpoch,
				WithdrawalEpoch: validator.WithdrawableEpoch,
				ExitEpoch:       validator.ExitEpoch,
				PublicKey:       validator.PublicKey,
			})
		}

	}
}

func (s *ChainAnalyzer) processEpochValRewards(bundle metrics.StateMetrics) {

	if s.metrics.ValidatorRewards { // only if flag is activated
		log.Debugf("persising validator metrics: epoch %d", bundle.GetMetricsBase().NextState.Epoch)

		// process each validator
		for valIdx := range bundle.GetMetricsBase().NextState.Validators {

			if valIdx >= len(bundle.GetMetricsBase().NextState.Validators) {
				continue // validator is not in the chain yet
			}
			// get max reward at given epoch using the formulas
			maxRewards, err := bundle.GetMaxReward(phase0.ValidatorIndex(valIdx))

			if err != nil {
				log.Errorf("Error obtaining max reward: %s", err.Error())
				continue
			}

			if s.metrics.ValidatorRewards { // only if flag is activated
				s.dbClient.Persist(maxRewards)
			}

		}
	}
}
