package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/risk"
)

// BuildRiskEngineConfig maps application config into the risk engine config.
func BuildRiskEngineConfig(cfg RiskEngineConfig) (risk.Config, error) {
	out := risk.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
		MinConfidence:    cfg.MinConfidence,
		MaxPositions:     cfg.MaxPositions,
		MaxTradesPerDay:  cfg.MaxTradesPerDay,
		DefaultQuantity:  cfg.DefaultQuantity,
		DayResetTimezone: cfg.DayResetTimezone,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return risk.Config{}, fmt.Errorf("risk config: %w", err)
	}
	return out, nil
}
