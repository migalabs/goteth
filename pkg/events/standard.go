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

	SubscribedFinalized bool
	FinalizedChan       chan apiv1.FinalizedCheckpointEvent
	ReorgChan           chan apiv1.ChainReorgEvent
	BlobSidecarChan     chan spec.BlobSideCarEventWraper
}

func NewEventsObj(iCtx context.Context, iCli *clientapi.APIClient) Events {
	return Events{
		ctx:                 iCtx,
		cli:                 iCli,
		SubscribedHead:      false,
		HeadChan:            make(chan db.HeadEvent),
		SubscribedFinalized: false,
		FinalizedChan:       make(chan apiv1.FinalizedCheckpointEvent),
		ReorgChan:           make(chan apiv1.ChainReorgEvent),
		BlobSidecarChan:     make(chan spec.BlobSideCarEventWraper),
	}
}
