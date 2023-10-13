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

// --- Ethereum Type Converteres ----

func SlotTo[T uint64 | int64 | int](slot phase0.Slot) T {
	return T(slot)
}

func EpochTo[T uint64 | int64 | int](epoch phase0.Epoch) T {
	return T(epoch)
}

// --- Map ---

type AgnosticMapOption[T spec.AgnosticBlock | spec.AgnosticState] func(*AgnosticMap[T])

type AgnosticMap[T spec.AgnosticBlock | spec.AgnosticState] struct {
	sync.Mutex
	// both spec.Slot and spec.Epoch are uint64
	m    map[uint64]T
	subs map[uint64][]chan T

	setCollisionF func(T) // extra code we would like to do depending on an existing collision between an existing key and a new one
	deleteF       func(T) // extra code we want to run when deleting a key from the map
}

func NewAgnosticMap[T spec.AgnosticBlock | spec.AgnosticState](opts ...AgnosticMapOption[T]) *AgnosticMap[T] {
	// init by default with empty functions
	emptyF := func(_ T) {}
	agnosticMap := &AgnosticMap[T]{
		m:             make(map[uint64]T),
		subs:          make(map[uint64][]chan T),
		setCollisionF: emptyF,
		deleteF:       emptyF,
	}

	// apply options
	for _, opt := range opts {
		opt(agnosticMap)
	}
	return agnosticMap
}

func WithSetCollisionF[T spec.AgnosticBlock | spec.AgnosticState](f func(T)) AgnosticMapOption[T] {
	return func(m *AgnosticMap[T]) {
		m.setCollisionF = f
	}
}

func WithDeleteF[T spec.AgnosticBlock | spec.AgnosticState](f func(T)) AgnosticMapOption[T] {
	return func(m *AgnosticMap[T]) {
		m.deleteF = f
	}
}

func (m *AgnosticMap[T]) Set(key uint64, value T) {
	m.Lock()
	defer m.Unlock()

	prevItem, ok := m.m[key]
	if ok {
		m.setCollisionF(prevItem)
	}
	m.m[key] = value

	// Send the new value to all waiting subscribers of the key
	for _, sub := range m.subs[key] {
		sub <- value
	}
	delete(m.subs, key)
}

func (m *AgnosticMap[T]) Wait(key uint64) T {
	m.Lock()
	// Unlock cannot be deferred so we can unblock Set() while waiting

	value, ok := m.m[key]
	if ok {
		m.Unlock()
		return value
	}

	ticker := time.NewTicker(waitMaxTimeout)

	// if there is no value yet, subscribe to any new values for this key
	ch := make(chan T)
	m.subs[key] = append(m.subs[key], ch)
	m.Unlock()

	var item T
	select {
	case <-ticker.C:
		log.Fatalf("Waiting for too long for %T %d...", *new(T), key)
		return item
	case item = <-ch:
		return item
	}
}

func (m *AgnosticMap[T]) Delete(key uint64) {
	m.Lock()
	prevItem, ok := m.m[key]
	if ok {
		m.deleteF(prevItem)
	}
	delete(m.m, key)
	delete(m.subs, key)
	m.Unlock()

}
