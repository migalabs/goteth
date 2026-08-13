package events

import (
	"context"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/migalabs/goteth/pkg/clientapi"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "Events",
	)
)

type Events struct {
	ctx            context.Context
	cli            *clientapi.APIClient
	SubscribedHead bool
	HeadChan       chan db.HeadEvent

	SubscribedFinalized   bool
	FinalizedChan         chan apiv1.FinalizedCheckpointEvent
	ReorgChan             chan apiv1.ChainReorgEvent
	DataColumnSidecarChan chan spec.DataColumnSidecarEventWrapper
}

func NewEventsObj(iCtx context.Context, iCli *clientapi.APIClient) Events {
	return Events{
		ctx:                 iCtx,
		cli:                 iCli,
		SubscribedHead:      false,
		HeadChan:            make(chan db.HeadEvent, 32),
		SubscribedFinalized: false,
		FinalizedChan:       make(chan apiv1.FinalizedCheckpointEvent),
		ReorgChan:           make(chan apiv1.ChainReorgEvent),
		// Buffered like HeadChan: HandleDataColumnSidecarEvent does a blocking send, so
		// a full channel stalls the SSE reader and with it head ingestion. Sized much
		// larger than the old blob channel on purpose: PeerDAS emits one event per data
		// *column*, so a single block can produce up to 128 events (NUMBER_OF_COLUMNS)
		// instead of the <=6 blob events it used to, all arriving in a burst.
		DataColumnSidecarChan: make(chan spec.DataColumnSidecarEventWrapper, 1024),
	}
}
