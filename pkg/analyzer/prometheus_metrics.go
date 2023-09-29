package analyzer

import (
	"github.com/migalabs/goteth/pkg/metrics"
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
)

func (c *ChainAnalyzer) GetPrometheusMetrics() *metrics.MetricsModule {
	metricsMod := metrics.NewMetricsModule(
		modName,
		modDetails,
	)
	// compose all the metrics
	metricsMod.AddIndvMetric(c.lastProcessedSlotMetric())
	metricsMod.AddIndvMetric(c.lastProcessedEpochMetric())

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
