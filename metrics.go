package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pcm "github.com/prometheus/client_model/go"
)

// This is the main metrics config struct
type appMetrics struct {
	labels      map[string]string
	labelNames  []string
	labelValues []string

	// Gauges
	config *prometheus.GaugeVec

	// Counters
	httpRequestsTotal *prometheus.CounterVec
}

func initMetrics(registry *prometheus.Registry, labelNames, labelValues []string) appMetrics {

	am := appMetrics{}
	am.labelNames = labelNames
	am.labelValues = labelValues

	am.labels = make(map[string]string, 0)
	for i := range labelNames {
		am.labels[labelNames[i]] = labelValues[i]
	}

	am.config = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pr_notify",
			Name:      "config",
			Help:      "App config info",
		},
		append(am.labelNames, "version"),
	)

	am.httpRequestsTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pr_notify",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "The total number of requests processed",
		},
		am.labelNames,
	)

	am.config.WithLabelValues(append(labelValues, version)...).Set(1)
	am.httpRequestsTotal.WithLabelValues(labelValues...).Add(0)

	return am
}

// Get counter value
func getCounter(cp *prometheus.CounterVec, labels ...string) (float64, error) {

	c, err := cp.GetMetricWithLabelValues(labels...)
	if err != nil {
		return float64(0), err
	}

	m := &pcm.Metric{}
	c.Write(m)

	return m.GetCounter().GetValue(), nil
}
