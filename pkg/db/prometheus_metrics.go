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

func (r *DBService) GetPrometheusMetrics() *metrics.MetricsModule {
	metricsMod := metrics.NewMetricsModule(
		modName,
		"metrics about the database",
	)
	// compose all the metrics
	metricsMod.AddIndvMetric(r.getPersistRows())
	metricsMod.AddIndvMetric(r.getPersistTime())
	metricsMod.AddIndvMetric(r.getPersistRate())
	return metricsMod
}

func (r *DBService) getPersistRows() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(RowsPersisted)
		return nil
	}
	updateFn := func() (interface{}, error) {
		var sumRows int
		for k, v := range r.monitorMetrics {
			metrics := v
			RowsPersisted.WithLabelValues(k).Set(float64(metrics.Rows))
			sumRows += metrics.Rows
		}

		return sumRows, nil
	}
	rowsPersisted, err := metrics.NewIndvMetrics(
		"rows_persisted",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return rowsPersisted
}

func (r *DBService) getPersistTime() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(TimePersisted)
		return nil
	}
	updateFn := func() (interface{}, error) {
		var sumTimes float64
		for k, v := range r.monitorMetrics {
			metrics := v
			TimePersisted.WithLabelValues(k).Set(float64(metrics.PersistTime.Seconds()))
			sumTimes += metrics.PersistTime.Seconds()
		}

		return sumTimes, nil
	}
	timePersisted, err := metrics.NewIndvMetrics(
		"time_persisted",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return timePersisted
}

func (r *DBService) getPersistRate() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(RatePersisted)
		return nil
	}
	updateFn := func() (interface{}, error) {
		var rates float64
		for k, v := range r.monitorMetrics {
			metrics := v
			RatePersisted.WithLabelValues(k).Set(float64(metrics.RowRate))
			rates += metrics.RowRate
		}
		avgRate := rates / float64(len(r.monitorMetrics))

		return avgRate, nil
	}
	ratePersisted, err := metrics.NewIndvMetrics(
		"rows_s_persisted",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return ratePersisted
}
