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

type SlotRoot struct {
	Slot      phase0.Slot
	Epoch     phase0.Epoch
	StateRoot phase0.Root
}

type StateQueue struct {
	prevState       spec.AgnosticState
	currentState    spec.AgnosticState
	nextState       spec.AgnosticState
	Roots           map[phase0.Slot]SlotRoot // Here we will store stateroots from the blocks
	HeadRoot        SlotRoot
	LatestFinalized SlotRoot
}

func NewStateQueue(finalizedSlot phase0.Slot, finalizedRoot phase0.Root) StateQueue {
	return StateQueue{
		prevState:    spec.AgnosticState{},
		currentState: spec.AgnosticState{},
		nextState:    spec.AgnosticState{},
		Roots:        make(map[phase0.Slot]SlotRoot),
		LatestFinalized: SlotRoot{
			Slot:      finalizedSlot,
			Epoch:     phase0.Epoch(finalizedSlot / spec.SlotsPerEpoch),
			StateRoot: finalizedRoot,
		},
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
	if s.prevState.StateRoot != emptyRoot {
		return true
	}
	return false
}

func (s *StateQueue) AddRoot(iSlot phase0.Slot, iRoot phase0.Root) {
	s.Roots[iSlot] = SlotRoot{
		Slot:      iSlot,
		Epoch:     phase0.Epoch(iSlot / spec.SlotsPerEpoch),
		StateRoot: iRoot,
	}
}

func (s *StateQueue) Rewind(slot phase0.Slot) {

	for i := s.HeadRoot.Slot; i >= slot; i-- {
		delete(s.Roots, i)
		s.HeadRoot = s.Roots[i-1]
		if i%spec.SlotsPerEpoch == 31 { // end of epoch, remove the state
			s.nextState = s.currentState
			s.currentState = s.prevState
			s.prevState = spec.AgnosticState{} // epoch = 0
		}
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
