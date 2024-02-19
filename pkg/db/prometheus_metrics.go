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

func (r *DBService) initMonitorMetrics() {

	tablesArr := []string{
		blocksTable,
		epochsTable,
		finalizedTable,
		genesisTable,
		headEventsTable,
		orphansTable,
		poolsTables,
		proposerDutiesTable,
		reorgsTable,
		transactionsTable,
		valLastStatusTable,
		valRewardsTable,
		withdrawalsTable}

	for _, tableName := range tablesArr {
		r.monitorMetrics[tableName] = &DBMonitorMetrics{}
	}

}

func (r *DBService) GetPrometheusMetrics() *metrics.MetricsModule {
	metricsMod := metrics.NewMetricsModule(
		modName,
		"metrics about the database",
	)
	// compose all the metrics
	metricsMod.AddIndvMetric(r.getPersistMetrics())

	metricsMod.AddIndvMetric(r.lastProcessedSlotMetric())
	metricsMod.AddIndvMetric(r.lastProcessedEpochMetric())
	return metricsMod
}

func (r *DBService) getPersistMetrics() *metrics.IndvMetrics {
	initFn := func() error {
		prometheus.MustRegister(RowsPersisted)
		prometheus.MustRegister(TimePersisted)
		prometheus.MustRegister(RatePersisted)
		return nil
	}
	updateFn := func() (interface{}, error) {
		ratePersisted := make(map[string]float64)

		copyMonitorMetrics := r.getMonitorMetrics()

		for k, v := range copyMonitorMetrics {
			var rate float64
			secondsTime := v.PersistTime.Seconds()

			if secondsTime != 0 {
				rate = float64(v.Rows) / secondsTime
			}

			ratePersisted[k] = rate

			RowsPersisted.WithLabelValues(k).Set(float64(v.Rows))
			TimePersisted.WithLabelValues(k).Set(secondsTime)
			RatePersisted.WithLabelValues(k).Set(rate)
		}

		return ratePersisted, nil
	}
	persistingMetrics, err := metrics.NewIndvMetrics(
		"persisiting_metrics",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return persistingMetrics
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

func (r *DBService) getMonitorMetrics() map[string]DBMonitorMetrics {
	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()

	copyMonitorMetrics := make(map[string]DBMonitorMetrics, len(r.monitorMetrics))

	for table, metrics := range r.monitorMetrics {
		copyMonitorMetrics[table] = *metrics
	}

	return copyMonitorMetrics
}
