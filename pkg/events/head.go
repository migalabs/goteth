package events

import api "github.com/attestantio/go-eth2-client/api/v1"

func (e Events) SubscribeToHeadEvents() {
	// subscribe to head event
	err := e.cli.Api.Events(e.ctx, []string{"head"}, e.HandleHeadEvent) // every new head
	if err != nil {
		log.Panicf("failed to subscribe to head events: %s", err)
	}
}

func (e *Events) HandleHeadEvent(event *api.Event) {
	log := log.WithField("routine", "head-event")
	if event.Data == nil {
		return
	}

	data := event.Data.(*api.HeadEvent) // cast to head event
	log.Infof("Received a new event: slot %d", data.Slot)
	e.HeadChan <- data.Slot
}
