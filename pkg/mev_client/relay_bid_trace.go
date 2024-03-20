package mev_client

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	relayclient "github.com/attestantio/go-relay-client"
	v1 "github.com/attestantio/go-relay-client/api/v1"
	"github.com/attestantio/go-relay-client/http"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

const (
	ultraSoundRelay         string = "https://relay.ultrasound.money/"
	bloxRouteMaxProfitRelay string = "https://0x8b5d2e73e2a3a55c6c87b8b6eb92e0149a125c852751db1422fa951e42a09b82c142c3ea98d0d9930b056a3bc9896b8f@bloxroute.max-profit.blxrbdn.com"
	agnosticRelay           string = "https://agnostic-relay.net/"
	flashbotsRelay          string = "https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net"
	bloxRouteRegulatedRelay string = "https://0xb0b07cd0abef743db4260b0ed50619cf6ad4d82064cb4fbec9d3ec530f7c5e6793d9f286c4e082c0244ffb9f2658fe88@bloxroute.regulated.blxrbdn.com"
	aestusRelay             string = "https://0xa15b52576bcbf1072f4a011c0f99f9fb6c66f3e1ff321f11f461d15e31b1cb359caa092c71bbded0bae5b5ea401aab7e@aestus.live"
	manifoldRelay           string = "https://mainnet-relay.securerpc.com"
	edenNetworkRelay        string = "https://0xb3ee7afcf27f1f1259ac1787876318c6584ee353097a50ed84f51a1f21a323b3736f271a895c7ce918c038e4265918be@relay.edennetwork.io"
)

var (
	relayList []string = []string{
		ultraSoundRelay,
		bloxRouteMaxProfitRelay,
		agnosticRelay,
		flashbotsRelay,
		bloxRouteRegulatedRelay,
		aestusRelay,
		manifoldRelay,
		edenNetworkRelay,
	}
)

const (
	moduleName      = "mev_client"
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

func InitRelaysMonitorer(pCtx context.Context) (*RelaysMonitor, error) {
	relayClients := make([]RelayClient, 0)

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
func (m RelaysMonitor) GetDeliveredBidsPerSlotRange(slot phase0.Slot, limit int) (RelayBidsPerSlot, error) {
	bidsDelivered := newRelayBidsPerSlot()

	for _, relayClient := range m.relays {
		singleRelayBidsDelivered, err := relayClient.GetDeliveredBidsPerSlotRange(slot, limit)
		if err != nil || singleRelayBidsDelivered == nil {
			log.Errorf("%s", err)
			continue
		}

		for _, bid := range singleRelayBidsDelivered {
			bidsDelivered.addBid(relayClient.client.Address(), bid)
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
