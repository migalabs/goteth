package clientapi

import (
	"context"
	"fmt"
	"time"

	clhttp "github.com/attestantio/go-eth2-client/http"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/migalabs/goteth/pkg/db"
	prom_metrics "github.com/migalabs/goteth/pkg/metrics"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

var (
	moduleName = "api-cli"
	log        = logrus.WithField(
		"module", moduleName)
	QueryTimeout     = 3 * time.Minute
	maxParallelConns = 3
)

type APIClientOption func(*APIClient) error

type APIClient struct {
	ctx        context.Context
	Api        *clhttp.Service   // Beacon Node
	ELApi      *ethclient.Client // Execution Node
	Metrics    db.DBMetrics
	maxRetries int
	statesBook *utils.RoutineBook // Book to track what is being downloaded through the CL API: states
	blocksBook *utils.RoutineBook // Book to track what is being downloaded through the CL API: blocks
	txBook     *utils.RoutineBook // Book to track what is being downloaded through the EL API: transactions
}

func NewAPIClient(ctx context.Context, bnEndpoint string, bnApiKey string, cfAccessClientID string, cfAccessClientSecret string, maxRequestRetries int, options ...APIClientOption) (*APIClient, error) {
	log.Debugf("generating http client at %s", bnEndpoint)

	apiService := &APIClient{
		ctx:        ctx,
		statesBook: utils.NewRoutineBook(1, "api-cli-states"),
		blocksBook: utils.NewRoutineBook(1, "api-cli-blocks"),
		txBook:     utils.NewRoutineBook(maxParallelConns, "api-cli-tx"),
	}

	clientBuildingOpts := []clhttp.Parameter{
		clhttp.WithAddress(bnEndpoint),
		clhttp.WithLogLevel(zerolog.WarnLevel),
		clhttp.WithTimeout(QueryTimeout),
	}

	extraHeadersMap := make(map[string]string)
	if bnApiKey != "" {
		extraHeadersMap["X-goog-api-key"] = bnApiKey
	}

	if cfAccessClientID != "" {
		extraHeadersMap["CF-Access-Client-Id"] = cfAccessClientID
	}

	if cfAccessClientSecret != "" {
		extraHeadersMap["CF-Access-Client-Secret"] = cfAccessClientSecret
	}

	if len(extraHeadersMap) > 0 {
		clientBuildingOpts = append(clientBuildingOpts, clhttp.WithExtraHeaders(extraHeadersMap))
	}

	bnCli, err := clhttp.New(
		ctx,
		clientBuildingOpts...,
	)
	if err != nil {
		return &APIClient{}, err
	}

	hc, ok := bnCli.(*clhttp.Service)
	if !ok {
		log.Error("generating the http api client")
	}

	apiService.Api = hc
	apiService.maxRetries = maxRequestRetries
	for _, o := range options {
		err := o(apiService)
		if err != nil {
			log.Warn(err.Error()) // these are optional, show error and continue
		}
	}

	return apiService, nil
}

func WithELEndpoint(url string) APIClientOption {
	return func(s *APIClient) error {
		if url == "" {
			return fmt.Errorf("empty execution address, skipping. Beware transactions data might not be complete")
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

func WithPromMetrics(metrics *prom_metrics.PrometheusMetrics) APIClientOption {
	return func(s *APIClient) error {

		metrics.AddMeticsModule(s.statesBook.GetPrometheusMetrics())
		metrics.AddMeticsModule(s.blocksBook.GetPrometheusMetrics())
		metrics.AddMeticsModule(s.txBook.GetPrometheusMetrics())

		return nil
	}
}

func (s APIClient) ActiveReqNum() int {

	return s.blocksBook.ActivePages() + s.statesBook.ActivePages() + s.txBook.ActivePages()
}
