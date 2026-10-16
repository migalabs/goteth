package relay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	relayclient "github.com/attestantio/go-relay-client"
	v1 "github.com/attestantio/go-relay-client/api/v1"
	"github.com/attestantio/go-relay-client/http"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

const (
	moduleName      = "relays"
	mevRelayTimeout = 10 * time.Second

	// Circuit breaker: after this many consecutive failures, skip the
	// relay for a cooldown period instead of waiting for timeout every time.
	cbFailureThreshold = 3
	cbCooldown         = 2 * time.Minute

	// A failed request is retried before the relay's bids for the epoch are
	// given up: they are fetched once per epoch and nothing fetches them again.
	// The first retry waits relayRetryBackoff, each next one twice as long.
	relayRequestAttempts = 3
	relayRetryBackoff    = 5 * time.Second
)

// waitBeforeRetry waits d, or less if ctx is done. A variable so the tests can
// record the waits instead of sleeping.
var waitBeforeRetry = func(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

var (
	log = logrus.WithField(
		"module", moduleName)
)

type RelayBidOption func(*RelayClient) error

type RelayClient struct {
	ctx     context.Context
	client  relayclient.Service
	address string

	// Circuit breaker state
	mu              sync.Mutex
	consecutiveFail int
	openUntil       time.Time
}

func New(pCtx context.Context,
	address string,
) (*RelayClient, error) {

	client, err := http.New(
		pCtx,
		http.WithAddress(address),
		http.WithLogLevel(zerolog.WarnLevel),
		http.WithTimeout(mevRelayTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate relay client %s: %s", address, err)
	}

	return &RelayClient{
		ctx:     pCtx,
		client:  client,
		address: address,
	}, nil
}

// isOpen returns true if the circuit breaker is open (relay should be skipped).
func (r *RelayClient) isOpen() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.consecutiveFail < cbFailureThreshold {
		return false
	}
	if time.Now().After(r.openUntil) {
		// Cooldown expired — allow one probe attempt (half-open)
		r.consecutiveFail = cbFailureThreshold - 1
		return false
	}
	return true
}

// failing tells whether the relay's last query failed.
func (r *RelayClient) failing() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.consecutiveFail > 0
}

// recordResult updates the circuit breaker after a query.
func (r *RelayClient) recordResult(failed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if failed {
		r.consecutiveFail++
		if r.consecutiveFail >= cbFailureThreshold {
			r.openUntil = time.Now().Add(cbCooldown)
			log.Warnf("circuit breaker open for %s, skipping for %s", r.address, cbCooldown)
		}
	} else {
		r.consecutiveFail = 0
	}
}

// Retrieves payloads for the given slot
// if the blocks array if provided, the list will be filstered
// if error, the map positions will have an empty bid
func (r *RelayClient) GetDeliveredBidsPerSlotRange(slot phase0.Slot, limit int) ([]*v1.BidTrace, error) {

	if r.isOpen() {
		return nil, fmt.Errorf("circuit breaker open for %s, skipping", r.address)
	}

	// A relay whose last query failed gets a single attempt: retrying one that
	// is down only delays the epoch, and the circuit breaker deals with it.
	attempts := relayRequestAttempts
	if r.failing() {
		attempts = 1
	}
	backoff := relayRetryBackoff
	var err error
	for attempt := 1; ; attempt++ {
		// An empty answer (nil, nil) is an answer: the relay delivered nothing.
		var bidsDelivered []*v1.BidTrace
		bidsDelivered, err = r.client.(relayclient.DeliveredBidTraceProvider).DeliveredBulkBidTrace(r.ctx, slot, limit)
		if err == nil {
			r.recordResult(false)
			return bidsDelivered, nil
		}
		if attempt == attempts || r.ctx.Err() != nil {
			break
		}
		log.Warnf("retrying %s at slot %d in %s (attempt %d of %d): %s", r.address, slot, backoff, attempt+1, attempts, err)
		waitBeforeRetry(r.ctx, backoff)
		if r.ctx.Err() != nil {
			break
		}
		backoff *= 2
	}

	r.recordResult(true)
	return nil, fmt.Errorf("error obtaining delivered bid trace from %s: %s", r.address, err)
}

type RelaysMonitor struct {
	relays []*RelayClient
}

func InitRelaysMonitorer(pCtx context.Context, genesisTime uint64) (*RelaysMonitor, error) {
	relayClients := make([]*RelayClient, 0)
	relayList := getNetworkRelays(genesisTime)

	for _, item := range relayList {
		relayClient, err := New(pCtx, item)
		if err != nil {
			return nil, fmt.Errorf("relay client error: %s", err)
		}
		relayClients = append(relayClients, relayClient)
	}

	return &RelaysMonitor{
		relays: relayClients,
	}, nil

}

// Returns a map of bids per slot
// Each slot contains an array of bids using the same order as relayList
// Returns results from slot-limit (not included) to slot (included)
func (m RelaysMonitor) GetDeliveredBidsPerSlotRange(slot phase0.Slot, limit int) (*RelayBidsPerSlot, error) {
	bidsDelivered := newRelayBidsPerSlot()

	var wg sync.WaitGroup

	for _, relayClient := range m.relays {
		wg.Add(1)
		go func(rc *RelayClient) {
			defer wg.Done()

			singleRelayBidsDelivered, err := rc.GetDeliveredBidsPerSlotRange(slot, limit)
			if err != nil {
				log.Errorf("%s", err)
				return
			}

			for _, bid := range singleRelayBidsDelivered {
				if bid.Slot > (slot-phase0.Slot(limit)) && bid.Slot <= slot { // if the bid inside the requested slots
					bidsDelivered.addBid(rc.address, bid)
				}
			}
		}(relayClient)
	}

	wg.Wait()

	return bidsDelivered, nil
}

type RelayBidsPerSlot struct {
	mu   sync.Mutex
	bids map[phase0.Slot]map[string]*v1.BidTrace
}

func newRelayBidsPerSlot() *RelayBidsPerSlot {
	return &RelayBidsPerSlot{
		bids: make(map[phase0.Slot]map[string]*v1.BidTrace),
	}
}

func (r *RelayBidsPerSlot) addBid(address string, bid *v1.BidTrace) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slot := bid.Slot

	if r.bids[slot] == nil {
		r.bids[slot] = make(map[string]*v1.BidTrace)
	}
	slotBidList := r.bids[slot]
	slotBidList[address] = bid
}

func (r *RelayBidsPerSlot) GetBidsAtSlot(slot phase0.Slot) map[string]v1.BidTrace {
	if r == nil {
		return nil
	}
	bids := make(map[string]v1.BidTrace)

	for address, bid := range r.bids[slot] {
		bids[address] = *bid
	}
	return bids
}

func getNetworkRelays(genesisTime uint64) []string {

	switch genesisTime {
	case spec.MainnetGenesis:
		return mainnetRelayList

	case spec.HoleskyGenesis:
		return holeskyRelayList
	case spec.HoodiGenesis:
		return hoodiRelayList
	case spec.SepoliaGenesis:
		return sepoliaRelayList
	default:
		log.Errorf("could not find network. Genesis time: %d", genesisTime)
		return []string{}
	}

}
