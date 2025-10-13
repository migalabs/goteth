package events

import (
	eth2api "github.com/attestantio/go-eth2-client/api"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
)

func (e *Events) SubscribeToReorgsEvents() {
	// subscribe to head event
	err := e.cli.Api.Events(e.ctx, &eth2api.EventsOpts{
		Topics:  []string{"chain_reorg"},
		Handler: e.HandleReorgEvent,
	}) // every reorg
	if err != nil {
		log.Panicf("failed to subscribe to chain_reorg events: %s", err)
	}
	log.Infof("subscribed to chain_reorg events")
}

func (e *Events) HandleReorgEvent(event *apiv1.Event) {
	log := log.WithField("routine", "reorg-event")
	if event.Data == nil {
		return
	}

	data := event.Data.(*apiv1.ChainReorgEvent) // cast to head event
	log.Infof("New event: slot %d of depth %d", data.Slot, data.Depth)

	e.ReorgChan <- *data
}
