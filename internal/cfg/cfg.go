package cfg

import (
	"github.com/impossiblecloud/pr-notify/internal/metrics"
)

// Config is the main app config struct
type AppConfig struct {
	Metrics metrics.AppMetrics
}
