package db

import (
	"strings"

	"github.com/migalabs/goteth/pkg/metrics"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
)

var (

	// List of metrics that we are going to export
	LastProcessedEpoch = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: strings.ToLower(utils.CliName),
		Subsystem: modName,
		Name:      "last_processed_epoch",
		Help:      "Last epoch processed with metrics",
	})
	// List of metrics that we are going to export
	LastProcessedSlot = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: strings.ToLower(utils.CliName),
		Subsystem: modName,
		Name:      "last_processed_slot",
		Help:      "Last slot processed with metrics",
	})

	// List of metrics that we are going to export
	RowsPersisted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: strings.ToLower(utils.CliName),
			Subsystem: modName,
			Name:      "rows_persisted",
			Help:      "Rows persisted on the last insert",
		},
		[]string{
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
			"table",
		},
	)
	NumberPersisted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: strings.ToLower(utils.CliName),
			Subsystem: modName,
			Name:      "persist_count",
			Help:      "Number of batch persists done",
		},
		[]string{
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
	metricsMod.AddIndvMetric(r.getPersistCount())

	r.monitorMetrics = make(map[string][]DBMonitorMetrics)

	metricsMod.AddIndvMetric(r.lastProcessedSlotMetric())
	metricsMod.AddIndvMetric(r.lastProcessedEpochMetric())
	return metricsMod
}

func (r *DBService) getPersistRows() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(RowsPersisted)
		return nil
	}
	updateFn := func() (interface{}, error) {
		sumRows := make(map[string]float64)

		for k, v := range r.monitorMetrics {

			for _, persistMetrics := range v {
				sumRows[k] += float64(persistMetrics.Rows)
			}
			RowsPersisted.WithLabelValues(k).Set(sumRows[k])
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
		sumTimes := make(map[string]float64)

		for k, v := range r.monitorMetrics {

			for _, persistMetrics := range v {
				sumTimes[k] += persistMetrics.PersistTime.Seconds()
			}
			TimePersisted.WithLabelValues(k).Set(sumTimes[k])
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
		rate := make(map[string]float64)

		for k, v := range r.monitorMetrics {
			var sumTimes float64
			var sumRows float64

			for _, persistMetrics := range v {
				sumTimes += persistMetrics.PersistTime.Seconds()
				sumRows += float64(persistMetrics.Rows)
			}
			if sumTimes != 0 {
				rate[k] = sumRows / sumTimes
			}
			RatePersisted.WithLabelValues(k).Set(rate[k])
		}

		return rate, nil
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

func (r *DBService) getPersistCount() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(NumberPersisted)
		return nil
	}
	updateFn := func() (interface{}, error) {
		numberPersists := make(map[string]float64)

		for k, v := range r.monitorMetrics {
			numberPersists[k] += float64(len(v))
			NumberPersisted.WithLabelValues(k).Set(numberPersists[k])
		}

		return numberPersists, nil
	}
	ratePersisted, err := metrics.NewIndvMetrics(
		"persist_count",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return ratePersisted
}

func (r *DBService) lastProcessedEpochMetric() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(LastProcessedEpoch)
		return nil
	}
	updateFn := func() (interface{}, error) {
		epoch, err := r.RetrieveLastEpoch()
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

func (r *DBService) lastProcessedSlotMetric() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(LastProcessedSlot)
		return nil
	}
	updateFn := func() (interface{}, error) {
		slot, err := r.RetrieveLastSlot()
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
