package clientapi

import (
	"strings"
	"sync"

	"github.com/migalabs/goteth/pkg/metrics"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	clientAPIMetricsName    = "clientapi"
	clientAPIMetricsDetails = "metrics about API client interactions"
)

var (
	registerReceiptMetricsOnce sync.Once
	receiptFailureReasons      = []string{"not_found", "deadline_exceeded", "context_cancelled", "other", "unknown"}

	receiptRequestFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: strings.ToLower(utils.CliName),
			Subsystem: clientAPIMetricsName,
			Name:      "execution_receipt_request_failures_total",
			Help:      "Total number of execution receipt requests that ended in error.",
		},
		[]string{"reason"},
	)

	receiptRequestFailureAttempts = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: strings.ToLower(utils.CliName),
			Subsystem: clientAPIMetricsName,
			Name:      "execution_receipt_request_attempts",
			Help:      "Number of attempts issued when an execution receipt request ends in error.",
			Buckets:   []float64{1, 2, 3, 5, 8},
		},
		[]string{"reason"},
	)

	receiptRequestFailureTotals = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: strings.ToLower(utils.CliName),
			Subsystem: clientAPIMetricsName,
			Name:      "execution_receipt_request_failures",
			Help:      "Running total of execution receipt request failures grouped by reason.",
		},
		[]string{"reason"},
	)
)

type receiptMetrics struct {
	mu     sync.Mutex
	totals map[string]int64
}

func newReceiptMetrics() *receiptMetrics {
	return &receiptMetrics{
		totals: make(map[string]int64),
	}
}

func (m *receiptMetrics) recordFailure(reason string, attempts int) {
	if m == nil {
		return
	}
	if reason == "" {
		reason = "unknown"
	}
	if attempts <= 0 {
		attempts = 1
	}

	receiptRequestFailures.WithLabelValues(reason).Inc()
	receiptRequestFailureAttempts.WithLabelValues(reason).Observe(float64(attempts))

	m.mu.Lock()
	defer m.mu.Unlock()
	m.totals[reason]++
	receiptRequestFailureTotals.WithLabelValues(reason).Set(float64(m.totals[reason]))
}

func (m *receiptMetrics) snapshot() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make(map[string]int64, len(receiptFailureReasons))
	for _, reason := range receiptFailureReasons {
		out[reason] = m.totals[reason]
	}
	return out
}

func (m *receiptMetrics) getPrometheusMetrics() *metrics.MetricsModule {
	if m == nil {
		return nil
	}

	mod := metrics.NewMetricsModule(
		clientAPIMetricsName,
		clientAPIMetricsDetails,
	)

	initFn := func() error {
		registerReceiptMetricsOnce.Do(func() {
			prometheus.MustRegister(receiptRequestFailures)
			prometheus.MustRegister(receiptRequestFailureAttempts)
			prometheus.MustRegister(receiptRequestFailureTotals)
			for _, reason := range receiptFailureReasons {
				receiptRequestFailures.WithLabelValues(reason).Add(0)
				_, _ = receiptRequestFailureAttempts.GetMetricWithLabelValues(reason)
				receiptRequestFailureTotals.WithLabelValues(reason).Set(0)
			}
		})
		return nil
	}

	updateFn := func() (interface{}, error) {
		return m.snapshot(), nil
	}

	indvMetrics, err := metrics.NewIndvMetrics(
		"receipt_request_failures",
		initFn,
		updateFn,
	)
	if err != nil {
		log.Error(errors.Wrap(err, "unable to init receipt_request_failures metrics"))
		return nil
	}

	if err := mod.AddIndvMetric(indvMetrics); err != nil {
		log.Error(errors.Wrap(err, "unable to register receipt metrics module"))
		return nil
	}

	return mod
}
