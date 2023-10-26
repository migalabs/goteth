package clientapi

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

var (
	moduleName = "api-cli"
	log        = logrus.WithField(
		"module", moduleName)
	QueryTimeout     = time.Second * 90
	maxParallelConns = 3
	maxRetries       = 3
)

type APIClientOption func(*APIClient) error

type APIClient struct {
	ctx     context.Context
	Api     *http.Service     // Beacon Node
	ELApi   *ethclient.Client // Execution Node
	Metrics db.DBMetrics

	apiBook   *utils.RoutineBook // Book to track what is being downloaded through the CL API
	elApiBook *utils.RoutineBook // Book to track what is being downloaded through the EL API
}

func NewAPIClient(ctx context.Context, bnEndpoint string, options ...APIClientOption) (*APIClient, error) {
	log.Debugf("generating http client at %s", bnEndpoint)

	apiService := &APIClient{
		ctx:       ctx,
		apiBook:   utils.NewRoutineBook(maxParallelConns, "api-cli"),
		elApiBook: utils.NewRoutineBook(maxParallelConns, "api-cli"),
	}

	bnCli, err := http.New(
		ctx,
		http.WithAddress(bnEndpoint),
		http.WithLogLevel(zerolog.WarnLevel),
		http.WithTimeout(QueryTimeout),
	)
	if err != nil {
		return &APIClient{}, err
	}

	hc, ok := bnCli.(*http.Service)
	if !ok {
		log.Error("gernerating the http api client")
	}

	apiService.Api = hc

	for _, o := range options {
		err := o(apiService)
		if err != nil {
			log.Warnf(err.Error()) // these are optional, show error and continue
		}
	}

	return apiService, nil
}

func WithELEndpoint(url string) APIClientOption {
	return func(s *APIClient) error {
		if url == "" {
			return fmt.Errorf("empty execution address, skipping...")
		}
		client, err := ethclient.DialContext(s.ctx, url)
		if err != nil {
			return err
		}
		s.ELApi = client
		return nil
	}
}

func WithDBMetrics(metrics db.DBMetrics) APIClientOption {
	return func(s *APIClient) error {
		s.Metrics = metrics
		return nil
	}
}

func (s APIClient) ActiveReqNum() int {

	return s.apiBook.ActivePages() + s.elApiBook.ActivePages()
}
