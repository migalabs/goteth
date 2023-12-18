package db

import (
	"strings"

	"github.com/migalabs/goteth/pkg/metrics"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
)

var (

	// List of metrics that we are going to export
	RowsPersisted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: strings.ToLower(utils.CliName),
			Subsystem: modName,
			Name:      "rows_persisted",
			Help:      "Rows persisted on the last insert",
		},
		[]string{
			// Which user has requested the operation?
			"table",
		},
	)
	TimePersisted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: strings.ToLower(utils.CliName),
			Subsystem: modName,
			Name:      "time_persisted",
			Help:      "Duration (seconds) of last insert",
		},
		[]string{
			// Which user has requested the operation?
			"table",
		},
	)
	RatePersisted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: strings.ToLower(utils.CliName),
			Subsystem: modName,
			Name:      "rows_s_persisted",
			Help:      "Rows per second persisted in the last insert",
		},
		[]string{
			// Which user has requested the operation?
			"table",
		},
	)
)

func (r *PostgresDBService) GetPrometheusMetrics() *metrics.MetricsModule {
	metricsMod := metrics.NewMetricsModule(
		modName,
		"metrics about the database",
	)
	// compose all the metrics
	metricsMod.AddIndvMetric(r.getWriteChanLength())
	metricsMod.AddIndvMetric(r.getBatcherMetrics())
	metricsMod.AddIndvMetric(r.getPersistMetrics())

	return metricsMod
}

func (r *PostgresDBService) getWriteChanLength() *metrics.IndvMetrics {
	initFn := func() error {
		return nil
	}
	updateFn := func() (interface{}, error) {
		chanLen := len(r.writeChan)
		return chanLen, nil
	}
	writeChanLen, err := metrics.NewIndvMetrics(
		"write_chan_length",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return writeChanLen
}

func (r *PostgresDBService) getBatcherMetrics() *metrics.IndvMetrics {
	initFn := func() error {
		return nil
	}
	updateFn := func() (interface{}, error) {
		listAvg := r.GetBatcherStats()
		return listAvg, nil
	}
	batchAverages, err := metrics.NewIndvMetrics(
		"last_persist",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return batchAverages
}

func (r *PostgresDBService) getPersistMetrics() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(RowsPersisted)
		prometheus.MustRegister(TimePersisted)
		prometheus.MustRegister(RatePersisted)
		return nil
	}
	updateFn := func() (interface{}, error) {
		for k, v := range r.metrics {
			RowsPersisted.WithLabelValues(k).Set(float64(v.Rows))
			TimePersisted.WithLabelValues(k).Set(v.PersistTime.Seconds())
			RatePersisted.WithLabelValues(k).Set(v.RatePersisted)
		}
		listAvg := r.GetBatcherStats()
		return listAvg, nil
	}
	batchAverages, err := metrics.NewIndvMetrics(
		"last_copy",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return batchAverages
}
