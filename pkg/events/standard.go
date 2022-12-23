package events

import (
	"context"

	"github.com/cortze/eth2-state-analyzer/pkg/clientapi"
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
	HeadChan       chan struct{}
}

func NewEventsObj(iCtx context.Context, iCli *clientapi.APIClient) Events {
	return Events{
		ctx:            iCtx,
		cli:            iCli,
		SubscribedHead: false,
		HeadChan:       make(chan struct{}),
	}
}
