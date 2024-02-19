package events

import (
	"time"

	"github.com/migalabs/goteth/pkg/db"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (e Events) SubscribeToHeadEvents() {
	// subscribe to head event
	err := e.cli.Api.Events(e.ctx, []string{"head"}, e.HandleHeadEvent) // every new head
	if err != nil {
		log.Panicf("failed to subscribe to head events: %s", err)
	}
	log.Infof("subscribed to head events")
}

func (e *Events) HandleHeadEvent(event *api.Event) {
	timestamp := time.Now().UnixNano() / 1000000
	log := log.WithField("routine", "head-event")
	if event.Data == nil {
		return
	}
	data := event.Data.(*api.HeadEvent) // cast to head event
	headEpoch := phase0.Epoch(data.Slot) / spec.SlotsPerEpoch

	log.Infof("New event: slot %d, epoch %d. %d pending slots for new epoch",
		data.Slot,
		data.Slot/spec.SlotsPerEpoch,
		(int(headEpoch+1)*spec.EpochSlots)-int(data.Slot))

	select { // only notify if we can
	case e.HeadChan <- db.HeadEvent{
		HeadEvent:        *data,
		ArrivalTimestamp: timestamp}:
	default:
	}

}
