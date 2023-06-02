package analyzer

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
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

func (s StateAnalyzer) DownloadBeaconStateAndBlocks(epoch phase0.Epoch) (local_spec.AgnosticState, error) {
	newState, err := s.downloadNewState(epoch)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return local_spec.AgnosticState{}, fmt.Errorf("unable to retrieve Beacon State at %d: %s", phase0.Slot((epoch+1)*local_spec.SlotsPerEpoch-1), err.Error())

	}
	blockList := make([]local_spec.AgnosticBlock, 0)
	for i := phase0.Slot((epoch) * local_spec.SlotsPerEpoch); i < phase0.Slot((epoch+1)*local_spec.SlotsPerEpoch-1); i++ {
		block, err := s.downloadNewBlock(i)
		if err != nil {
			return local_spec.AgnosticState{}, fmt.Errorf("unable to retrieve Beacon State at %d: %s", phase0.Slot((epoch+1)*local_spec.SlotsPerEpoch-1), err.Error())
		}
		blockList = append(blockList, block)
	}
	newState.BlockList = blockList

	return newState, nil
}

func (s StateAnalyzer) downloadNewBlock(slot phase0.Slot) (local_spec.AgnosticBlock, error) {

	// log.Infof("requesting Beacon Block from endpoint: slot %d", slot)
	newBlock, err := s.cli.RequestBeaconBlock(slot)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return local_spec.AgnosticBlock{}, fmt.Errorf("unable to retrieve beacon state from the beacon node, closing requester routine. %s", err.Error())

	}

	return newBlock, nil
}

func (s StateAnalyzer) downloadNewState(epoch phase0.Epoch) (local_spec.AgnosticState, error) {

	log.Debugf("requesting Beacon State from endpoint: epoch %d", epoch)
	newState, err := s.cli.RequestBeaconState(epoch)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return local_spec.AgnosticState{}, fmt.Errorf("unable to retrieve beacon state from the beacon node, closing requester routine. %s", err.Error())

	}
	epochDuties := s.cli.NewEpochDuties(epoch)

	resultState, err := local_spec.GetCustomState(*newState, epochDuties)
	if err != nil {
		// close the channel (to tell other routines to stop processing and end)
		return local_spec.AgnosticState{}, fmt.Errorf("unable to open beacon state, closing requester routine. %s", err.Error())
	}

	return resultState, nil

}

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
