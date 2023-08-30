package analyzer

import (
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/clientapi"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db"
	"github.com/cortze/eth-cl-state-analyzer/pkg/spec"
	"github.com/sirupsen/logrus"
)

const (
	ValidatorSetSize           = 500000                 // Estimation of current number of validators, used for channel length declaration
	maxWorkers                 = 50                     // maximum number of workers allowed in the tool
	minBlockReqTime            = 100 * time.Millisecond // max 10 queries per second, dont spam beacon node
	minStateReqTime            = 1 * time.Second        // max 1 query per second, dont spam beacon node
	epochsToFinalizedTentative = 3                      // usually, 2 full epochs before the head it is finalized
)

var (
	log = logrus.WithField(
		"module", "analyzer",
	)
)

type StateQueue struct {
	prevState       spec.AgnosticState
	currentState    spec.AgnosticState
	nextState       spec.AgnosticState
	BlockHistory    map[phase0.Slot]spec.AgnosticBlock // Here we will store stateroots from the blocks
	HeadBlock       spec.AgnosticBlock
	LatestFinalized spec.AgnosticBlock
}

func NewStateQueue(finalizedBlock spec.AgnosticBlock) StateQueue {
	return StateQueue{
		prevState:       spec.AgnosticState{},
		currentState:    spec.AgnosticState{},
		nextState:       spec.AgnosticState{},
		BlockHistory:    make(map[phase0.Slot]spec.AgnosticBlock),
		LatestFinalized: finalizedBlock,
	}
}

func (s *StateQueue) AddNewState(newState spec.AgnosticState) {

	if s.nextState.Epoch != phase0.Epoch(0) && newState.Epoch != s.nextState.Epoch+1 {
		log.Panicf("state at epoch %d is not consecutive to %d...", newState.Epoch, s.nextState.Epoch)
	}

	s.prevState = s.currentState
	s.currentState = s.nextState
	s.nextState = newState
}

func (s StateQueue) Complete() bool {
	emptyRoot := phase0.Root{}
	if s.currentState.StateRoot != emptyRoot { // if we have currentState we can already write epoch metrics
		return true
	}
	return false
}

func (s *StateQueue) AddNewBlock(block spec.AgnosticBlock) {

	// check previous slot exists

	_, ok := s.BlockHistory[block.Slot-1]

	if !ok && len(s.BlockHistory) > 0 { // if there are roots and we did not find the previous one
		log.Panicf("root at slot %d:%s is not consecutive to %d", block.Slot, block.StateRoot, block.Slot-1)
	}

	s.BlockHistory[block.Slot] = block
	s.HeadBlock = block
}

// Advances the finalized checkpoint to the given slot
func (s *StateQueue) AdvanceFinalized(slot phase0.Slot) {

	for i := s.LatestFinalized.Slot; i < slot; i++ {
		delete(s.BlockHistory, i)
		s.LatestFinalized = s.BlockHistory[i+1] // we assume all roots exist in the array
	}
}

func InitGenesis(dbClient *db.PostgresDBService, apiClient *clientapi.APIClient) {
	// Get genesis from the API
	apiGenesis := apiClient.RequestGenesis()

	// Insert into db, this does nothing if there was a genesis before
	dbClient.InsertGenesis(apiGenesis.Unix())

	dbGenesis := dbClient.ObtainGenesis()

	if apiGenesis.Unix() != dbGenesis {
		log.Panicf("the genesis time in the database does not match the API, is the beacon node in the correct network?")
	}

}
