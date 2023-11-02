package db

import "github.com/migalabs/goteth/pkg/metrics"

func (r *PostgresDBService) GetPrometheusMetrics() *metrics.MetricsModule {
	metricsMod := metrics.NewMetricsModule(
		modName,
		"metrics about the database",
	)
	// compose all the metrics
	metricsMod.AddIndvMetric(r.getWriteChanLength())
	metricsMod.AddIndvMetric(r.getBatcherMetrics())

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
		"average_batch_time_ms",
		initFn,
		updateFn,
	)
	if err != nil {
		return nil
	}
	return batchAverages
}
