package analyzer

import (
	"fmt"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	local_spec "github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec/metrics"
	"github.com/sirupsen/logrus"
)

const (
	ValidatorSetSize = 500000 // Estimation of current number of validators, used for channel length declaration
	maxWorkers       = 50
	minBlockReqTime  = 100 * time.Millisecond // max 10 queries per second, dont spam beacon node
	minStateReqTime  = 1 * time.Second        // max 1 query per second, dont spam beacon node
)

var (
	log = logrus.WithField(
		"module", "analyzer",
	)
)

type StateQueue struct {
	FirstState  local_spec.AgnosticState
	SecondState local_spec.AgnosticState
	ThirdState  local_spec.AgnosticState
	FourthState local_spec.AgnosticState
	Finalized   bool
}

func NewStateQueue() StateQueue {
	return StateQueue{
		FirstState:  local_spec.AgnosticState{},
		SecondState: local_spec.AgnosticState{},
		ThirdState:  local_spec.AgnosticState{},
		FourthState: local_spec.AgnosticState{},
	}
}

// Shift states
func (s *StateQueue) AddNewState(input local_spec.AgnosticState) error {

	if s.FourthState.Slot > 0 &&
		input.Slot != s.FourthState.Slot+local_spec.SlotsPerEpoch {
		return fmt.Errorf("slot not in next epoch to nextState")
	}
	s.FirstState = s.SecondState
	s.SecondState = s.ThirdState
	s.ThirdState = s.FourthState
	s.FourthState = input

	return nil
}

// used to see if we already have 4 states in the queue
func (s StateQueue) Complete() bool {
	if s.FirstState.BlockList == nil { // we always download blocks
		return false
	}
	return true
}

func (s *StateAnalyzer) persistEpochData(stateMetrics metrics.StateMetrics) {

	if !s.metrics.Epoch {
		return // Only persist when metric activated
	}

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

func (s *StateAnalyzer) persistBlockData(block spec.AgnosticBlock) {

	if s.metrics.Block {
		s.dbClient.Persist(block)
	}

	if s.metrics.Withdrawals {
		for _, item := range block.ExecutionPayload.Withdrawals {
			s.dbClient.Persist(spec.Withdrawal{
				Slot:           block.Slot,
				Index:          item.Index,
				ValidatorIndex: item.ValidatorIndex,
				Address:        item.Address,
				Amount:         item.Amount,
			})

		}
	}

	// store transactions if it has been enabled
	if s.metrics.Transaction {

		for _, tx := range spec.RequestTransactionDetails(block) {
			s.dbClient.Persist(tx)
		}
	}
}
