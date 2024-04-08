package relay

import (
	"context"
	"fmt"
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
	mevRelayTimeout = 3 * time.Minute
)

var (
	log = logrus.WithField(
		"module", moduleName)
)

type RelayBidOption func(*RelayClient) error

type RelayClient struct {
	ctx    context.Context
	client relayclient.Service
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
		ctx:    pCtx,
		client: client,
	}, nil
}

// Retrieves payloads for the given slot
// if the blocks array if provided, the list will be filstered
// if error, the map positions will have an empty bid
func (r RelayClient) GetDeliveredBidsPerSlotRange(slot phase0.Slot, limit int) ([]*v1.BidTrace, error) {

	bidsDelivered, err := r.client.(relayclient.DeliveredBidTraceProvider).DeliveredBulkBidTrace(r.ctx, slot, limit)
	if err != nil || bidsDelivered == nil {
		return bidsDelivered, fmt.Errorf("error obtaining delivered bid trace from %s: %s", r.client.Address(), err)
	}

	return bidsDelivered, nil

}

type RelaysMonitor struct {
	relays []RelayClient
}

func InitRelaysMonitorer(pCtx context.Context, genesisTime uint64) (*RelaysMonitor, error) {
	relayClients := make([]RelayClient, 0)
	relayList := getNetworkRelays(genesisTime)

	for _, item := range relayList {
		relayClient, err := New(pCtx, item)
		if err != nil {
			return nil, fmt.Errorf("relay client error: %s", err)
		}
		relayClients = append(relayClients, *relayClient)
	}

	return &RelaysMonitor{
		relays: relayClients,
	}, nil

}

// Returns a map of bids per slot
// Each slot contains an array of bids using the same order as relayList
// Returns results from slot-limit (not included) to slot (included)
func (m RelaysMonitor) GetDeliveredBidsPerSlotRange(slot phase0.Slot, limit int) (RelayBidsPerSlot, error) {
	bidsDelivered := newRelayBidsPerSlot()

	for _, relayClient := range m.relays {
		singleRelayBidsDelivered, err := relayClient.GetDeliveredBidsPerSlotRange(slot, limit)
		if err != nil || singleRelayBidsDelivered == nil {
			log.Errorf("%s", err)
			continue
		}

		for _, bid := range singleRelayBidsDelivered {
			if bid.Slot > (slot-phase0.Slot(limit)) && bid.Slot <= slot { // if the bid inside the requested slots
				bidsDelivered.addBid(relayClient.client.Address(), bid)
			}

		}
	}
	return bidsDelivered, nil
}

type RelayBidsPerSlot struct {
	bids map[phase0.Slot]map[string]*v1.BidTrace
}

func newRelayBidsPerSlot() RelayBidsPerSlot {
	return RelayBidsPerSlot{
		bids: make(map[phase0.Slot]map[string]*v1.BidTrace),
	}
}

func (r *RelayBidsPerSlot) addBid(address string, bid *v1.BidTrace) {
	slot := bid.Slot

	if r.bids[slot] == nil {
		r.bids[slot] = make(map[string]*v1.BidTrace)
	}
	slotBidList := r.bids[slot]
	slotBidList[address] = bid
}

func (r RelayBidsPerSlot) GetBidsAtSlot(slot phase0.Slot) map[string]v1.BidTrace {
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
	default:
		log.Errorf("could not find network. Genesis time: %d", genesisTime)
		return []string{}
	}

}
