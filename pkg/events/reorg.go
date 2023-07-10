package events

import api "github.com/attestantio/go-eth2-client/api/v1"

func (e *Events) SubscribeToReorgsEvents() {
	// subscribe to head event
	err := e.cli.Api.Events(e.ctx, []string{"chain_reorg"}, e.HandleReorgEvent) // every reorg
	if err != nil {
		log.Panicf("failed to subscribe to chain_reorg events: %s", err)
	}
	log.Infof("subscribed to chain_reorg events")
}

func (e *Events) HandleReorgEvent(event *api.Event) {
	log := log.WithField("routine", "reorg-event")
	if event.Data == nil {
		return
	}

	data := event.Data.(*api.ChainReorgEvent) // cast to head event
	log.Infof("Received a new event: slot %d of depth %d", data.Slot, data.Depth)

	e.ReorgChan <- *data
}
