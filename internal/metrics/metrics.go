package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AppMetrics contains pointers to prometheus metrics objects
type AppMetrics struct {
	Listen   string
	Registry *prometheus.Registry

	// Gauges
	Config *prometheus.GaugeVec
}

func InitMetrics(version string) AppMetrics {

	am := AppMetrics{}
	am.Registry = prometheus.NewRegistry()

	// Config info
	am.Config = promauto.With(am.Registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pr_notify",
			Name:      "config",
			Help:      "App config info",
		},
		[]string{"version"},
	)

	am.Config.WithLabelValues(version).Set(1)
	am.Registry.MustRegister()
	return am
}
