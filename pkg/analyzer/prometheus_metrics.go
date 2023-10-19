package analyzer

import (
	"github.com/migalabs/goteth/pkg/metrics"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	modName    = "analyzer"
	modDetails = "general metrics about the analyzer"

	// List of metrics that we are going to export
	LastProcessedEpoch = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: modName,
		Name:      "last_processed_epoch",
		Help:      "Last epoch processed with metrics",
	})
	// List of metrics that we are going to export
	LastProcessedSlot = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: modName,
		Name:      "last_processed_slot",
		Help:      "Last slot processed with metrics",
	})
	StateQueueLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "goteth",
		Name:      "state_queue_length",
		Help:      "The number of states int the history queue",
	})
	BlockQueueLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "goteth",
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
	metricsMod.AddIndvMetric(c.lastProcessedSlotMetric())
	metricsMod.AddIndvMetric(c.lastProcessedEpochMetric())
	metricsMod.AddIndvMetric(c.getStateHistoryLength())
	metricsMod.AddIndvMetric(c.getBlockHistoryLength())

	return metricsMod
}

func (c *ChainAnalyzer) lastProcessedEpochMetric() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(LastProcessedEpoch)
		return nil
	}
	updateFn := func() (interface{}, error) {
		epoch, err := c.dbClient.ObtainLastEpoch()
		if err != nil {
			return nil, err
		}
		LastProcessedEpoch.Set(float64(epoch))
		return epoch, nil
	}
	lastEpoch, err := metrics.NewIndvMetrics(
		"last_processed_epoch",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return lastEpoch
}

func (c *ChainAnalyzer) lastProcessedSlotMetric() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(LastProcessedSlot)
		return nil
	}
	updateFn := func() (interface{}, error) {
		slot, err := c.dbClient.ObtainLastSlot()
		if err != nil {
			return nil, err
		}
		LastProcessedSlot.Set(float64(slot))
		return slot, nil
	}
	lastSlot, err := metrics.NewIndvMetrics(
		"last_processed_slot",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return lastSlot
}

func (p *ChainAnalyzer) getStateHistoryLength() *metrics.IndvMetrics {

	initFn := func() error {
		prometheus.MustRegister(StateQueueLength)
		return nil
	}

	updateFn := func() (interface{}, error) {
		numberStates := len(p.queue.StateHistory.GetKeyList())
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
		numberBlocks := len(p.queue.BlockHistory.GetKeyList())
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
