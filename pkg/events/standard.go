package events

import (
	"context"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/migalabs/goteth/pkg/clientapi"
	"github.com/migalabs/goteth/pkg/db"
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
	HeadChan       chan db.HeadEventType

	SubscribedFinalized bool
	FinalizedChan       chan api.FinalizedCheckpointEvent
	ReorgChan           chan api.ChainReorgEvent
}

func NewEventsObj(iCtx context.Context, iCli *clientapi.APIClient) Events {
	return Events{
		ctx:                 iCtx,
		cli:                 iCli,
		SubscribedHead:      false,
		HeadChan:            make(chan db.HeadEventType),
		SubscribedFinalized: false,
		FinalizedChan:       make(chan api.FinalizedCheckpointEvent),
		ReorgChan:           make(chan api.ChainReorgEvent),
	}
}
