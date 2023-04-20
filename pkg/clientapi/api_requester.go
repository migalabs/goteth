package clientapi

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

var (
	moduleName = "API-Cli"
	log        = logrus.WithField(
		"module", "")
)

type APIClient struct {
	ctx context.Context
	Api *http.Service
}

func NewAPIClient(ctx context.Context, cliEndpoint string, timeout time.Duration) (*APIClient, error) {
	log.Debugf("generating http client at %s", cliEndpoint)
	httpCli, err := http.New(
		ctx,
		http.WithAddress(cliEndpoint),
		http.WithLogLevel(zerolog.WarnLevel),
		http.WithTimeout(timeout),
	)
	if err != nil {
		return &APIClient{}, err
	}

	hc, ok := httpCli.(*http.Service)
	if !ok {
		log.Error("gernerating the http api client")
	}
	return &APIClient{
		ctx: ctx,
		Api: hc,
	}, nil
}
