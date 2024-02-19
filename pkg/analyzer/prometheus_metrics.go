package analyzer

import (
	"strings"

	"github.com/migalabs/goteth/pkg/metrics"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	modName    = "analyzer"
	modDetails = "general metrics about the analyzer"

	StateQueueLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: strings.ToLower(utils.CliName),
		Subsystem: modName,
		Name:      "state_queue_length",
		Help:      "The number of states int the history queue",
	})
	BlockQueueLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: strings.ToLower(utils.CliName),
		Subsystem: modName,
		Name:      "block_queue_length",
		Help:      "The number of blocks int the history queue",
	})
)

func (c *ChainAnalyzer) GetPrometheusMetrics() *metrics.MetricsModule {
	metricsMod := metrics.NewMetricsModule(
		modName,
		modDetails,
	)
	// compose all the metrics

	metricsMod.AddIndvMetric(c.getStateHistoryLength())
	metricsMod.AddIndvMetric(c.getBlockHistoryLength())

	return metricsMod
}

func (p *ChainAnalyzer) getStateHistoryLength() *metrics.IndvMetrics {

	initFn := func() error {
		prometheus.MustRegister(StateQueueLength)
		return nil
	}

	updateFn := func() (interface{}, error) {
		numberStates := len(p.downloadCache.StateHistory.GetKeyList())
		StateQueueLength.Set(float64(numberStates))
		return numberStates, nil
	}

	indvMetr, err := metrics.NewIndvMetrics(
		"state_queue_length",
		initFn,
		updateFn,
	)
	if err != nil {
		log.Error(errors.Wrap(err, "unable to init state_queue_length"))
		return nil
	}

	return indvMetr
}

func (p *ChainAnalyzer) getBlockHistoryLength() *metrics.IndvMetrics {

	initFn := func() error {
		prometheus.MustRegister(BlockQueueLength)
		return nil
	}

	updateFn := func() (interface{}, error) {
		numberBlocks := len(p.downloadCache.BlockHistory.GetKeyList())
		BlockQueueLength.Set(float64(numberBlocks))
		return numberBlocks, nil
	}

	indvMetr, err := metrics.NewIndvMetrics(
		"blocks_queue_length",
		initFn,
		updateFn,
	)
	if err != nil {
		log.Error(errors.Wrap(err, "unable to init blocks_queue_length"))
		return nil
	}

	return indvMetr
}
