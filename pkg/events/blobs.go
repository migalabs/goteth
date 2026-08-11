package events

import (
	"time"

	eth2api "github.com/attestantio/go-eth2-client/api"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/migalabs/goteth/pkg/spec"
)

// SubscribeToDataColumnSidecarsEvents subscribes to the p2p arrival of data columns.
//
// This replaces the old `blob_sidecar` subscription. Since the Fulu fork (PeerDAS)
// blobs are erasure-coded into columns and gossiped per column, so beacon nodes no
// longer emit `blob_sidecar` at all: the topic is still accepted (HTTP 200) but
// never fires. That is why blob arrival timings silently stopped being recorded at
// the fork while every other blob table kept working.
//
// A subscription failure is logged and skipped rather than crashing the process:
// the analyzer can keep indexing everything else without this data source. Note
// that Events() only returns errors synchronously for client-side conditions;
// node-side rejections surface asynchronously inside go-eth2-client's own
// goroutine, which logs and retries every second.
func (e *Events) SubscribeToDataColumnSidecarsEvents() {
	err := e.cli.Api.Events(e.ctx, &eth2api.EventsOpts{
		Topics:  []string{"data_column_sidecar"},
		Handler: e.HandleDataColumnSidecarEvent,
	})
	if err != nil {
		log.Errorf("failed to subscribe to data_column_sidecar events, blob arrival timings will not be recorded: %s", err)
		return
	}
	e.SubscribedDataColumns.Store(true)
	log.Infof("subscribed to data_column_sidecar events")
}

func (e *Events) HandleDataColumnSidecarEvent(event *apiv1.Event) {
	timestamp := time.Now()
	if event.Data == nil {
		return
	}

	data := spec.DataColumnSidecarEventWrapper{
		Timestamp:              timestamp,
		DataColumnSidecarEvent: *event.Data.(*apiv1.DataColumnSidecarEvent),
	}

	e.DataColumnSidecarChan <- data
}
