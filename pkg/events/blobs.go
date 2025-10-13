package events

import (
	"time"

	eth2api "github.com/attestantio/go-eth2-client/api"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/migalabs/goteth/pkg/spec"
)

func (e *Events) SubscribeToBlobSidecarsEvents() {
	// subscribe to head event
	err := e.cli.Api.Events(e.ctx, &eth2api.EventsOpts{
		Topics:  []string{"blob_sidecar"},
		Handler: e.HandleBlobSidecarEvent,
	}) // every reorg
	if err != nil {
		log.Panicf("failed to subscribe to blob_sidecar events: %s", err)
	}
	log.Infof("subscribed to blob_sidecar events")
}

func (e *Events) HandleBlobSidecarEvent(event *apiv1.Event) {
	timestamp := time.Now()
	if event.Data == nil {
		return
	}

	data := spec.BlobSideCarEventWraper{
		Timestamp:        timestamp,
		BlobSidecarEvent: *event.Data.(*apiv1.BlobSidecarEvent),
	}

	e.BlobSidecarChan <- data
}
