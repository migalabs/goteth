package analyzer

import (
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/clientapi"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/sirupsen/logrus"
)

const (
	ValidatorSetSize           = 500000                 // Estimation of current number of validators, used for channel length declaration
	maxWorkers                 = 50                     // maximum number of workers allowed in the tool
	minBlockReqTime            = 100 * time.Millisecond // max 10 queries per second, dont spam beacon node
	minStateReqTime            = 1 * time.Second        // max 1 query per second, dont spam beacon node
	epochsToFinalizedTentative = 3                      // usually, 2 full epochs before the head it is finalized
	waitMaxTimeout             = 60 * time.Second
)

var (
	log = logrus.WithField(
		"module", "analyzer",
	)
)

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

type BlocksMap struct {
	sync.Mutex

	m    map[phase0.Slot]spec.AgnosticBlock
	subs map[phase0.Slot][]chan spec.AgnosticBlock
}

func (m *BlocksMap) Set(key phase0.Slot, value spec.AgnosticBlock) {
	m.Lock()
	defer m.Unlock()

	m.m[key] = value

	// Send the new value to all waiting subscribers of the key
	for _, sub := range m.subs[key] {
		sub <- value
	}
	delete(m.subs, key)
}

func (m *BlocksMap) Wait(key phase0.Slot) spec.AgnosticBlock {
	m.Lock()
	// Unlock cannot be deferred so we can unblock Set() while waiting

	value, ok := m.m[key]
	if ok {
		m.Unlock()
		return value
	}

	ticker := time.NewTicker(waitMaxTimeout)

	// if there is no value yet, subscribe to any new values for this key
	ch := make(chan spec.AgnosticBlock)
	m.subs[key] = append(m.subs[key], ch)
	m.Unlock()

	select {
	case <-ticker.C:
		log.Fatalf("Waiting for too long for slot %d...", key)
		return spec.AgnosticBlock{}
	case block := <-ch:
		return block
	}
}

func (m *BlocksMap) Delete(key phase0.Slot) {
	m.Lock()
	delete(m.m, key)
	delete(m.subs, key)
	m.Unlock()
}

type StatesMap struct {
	sync.Mutex

	m    map[phase0.Epoch]spec.AgnosticState
	subs map[phase0.Epoch][]chan spec.AgnosticState
}

func (m *StatesMap) Set(key phase0.Epoch, value spec.AgnosticState) {
	m.Lock()
	defer m.Unlock()

	m.m[key] = value

	// Send the new value to all waiting subscribers of the key
	for _, sub := range m.subs[key] {
		sub <- value
	}
	delete(m.subs, key)
}

func (m *StatesMap) Wait(key phase0.Epoch) spec.AgnosticState {
	m.Lock()
	// Unlock cannot be deferred so we can unblock Set() while waiting

	value, ok := m.m[key]
	if ok {
		m.Unlock()
		return value
	}

	ticker := time.NewTicker(waitMaxTimeout)

	// if there is no value yet, subscribe to any new values for this key
	ch := make(chan spec.AgnosticState)
	m.subs[key] = append(m.subs[key], ch)
	m.Unlock()

	select {
	case <-ticker.C:
		log.Fatalf("Waiting for too long for state from epoch %d...", key)
		return spec.AgnosticState{}
	case state := <-ch:
		return state
	}
}

func (m *StatesMap) Delete(key phase0.Epoch) {
	delete(m.m, key)
	delete(m.subs, key)
}
