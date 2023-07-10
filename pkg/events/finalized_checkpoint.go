package events

import api "github.com/attestantio/go-eth2-client/api/v1"

func (e *Events) SubscribeToFinalizedCheckpointEvents() {
	// subscribe to head event
	err := e.cli.Api.Events(e.ctx, []string{"finalized_checkpoint"}, e.HandleCheckpointEvent) // every new checkpoint
	if err != nil {
		log.Panicf("failed to subscribe to finalized checkpoint events: %s", err)
	}
	log.Infof("subscribed to finalized checkpoint events")
}

func (e *Events) HandleCheckpointEvent(event *api.Event) {
	log := log.WithField("routine", "checkpoint-event")
	if event.Data == nil {
		return
	}

	data := event.Data.(*api.FinalizedCheckpointEvent) // cast to head event
	log.Infof("Received a new event: epoch %d, state root: %s", data.Epoch, data.State.String())

	select { // only notify if we can
	case e.FinalizedChan <- *data:
	default:
	}
}
