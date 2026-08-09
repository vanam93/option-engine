package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/performance"
)

// BuildPerformanceEngineConfig maps application config into the performance analytics engine config.
func BuildPerformanceEngineConfig(cfg PerformanceEngineConfig) (performance.Config, error) {
	out := performance.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return performance.Config{}, fmt.Errorf("performance config: %w", err)
	}
	return out, nil
}
