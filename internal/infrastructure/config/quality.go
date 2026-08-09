package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/quality"
)

// QualityConfig controls the recommendation quality engine.
type QualityConfig struct {
	Enabled                bool           `mapstructure:"enabled"`
	SubscriberBuffer       int            `mapstructure:"subscriber_buffer"`
	TrackingTimeoutMinutes int            `mapstructure:"tracking_timeout_minutes"`
	ExcellentThreshold     float64        `mapstructure:"excellent_threshold"`
	GoodThreshold          float64        `mapstructure:"good_threshold"`
	AverageThreshold       float64        `mapstructure:"average_threshold"`
	SuccessReturnPct       float64        `mapstructure:"success_return_pct"`
	FailureReturnPct       float64        `mapstructure:"failure_return_pct"`
	Feedback               FeedbackConfig `mapstructure:"feedback"`
	Delivery               DeliveryConfig `mapstructure:"delivery"`
}

// QualityEngineSettings maps quality settings from application config.
// Prefers top-level quality: when present; falls back to intelligence.quality.
func (c *Config) QualityEngineSettings() QualityConfig {
	if hasQualitySettings(c.Quality) {
		return c.Quality
	}
	return c.Intelligence.Quality
}

func hasQualitySettings(cfg QualityConfig) bool {
	return cfg.Enabled ||
		cfg.SubscriberBuffer != 0 ||
		cfg.TrackingTimeoutMinutes != 0 ||
		cfg.ExcellentThreshold != 0 ||
		cfg.GoodThreshold != 0 ||
		cfg.AverageThreshold != 0 ||
		cfg.SuccessReturnPct != 0 ||
		cfg.FailureReturnPct != 0
}

// BuildQualityEngineConfig maps application config into the quality engine config.
func BuildQualityEngineConfig(cfg QualityConfig) (quality.Config, error) {
	out := quality.Config{
		Enabled:                cfg.Enabled,
		SubscriberBuffer:       cfg.SubscriberBuffer,
		TrackingTimeoutMinutes: cfg.TrackingTimeoutMinutes,
		ExcellentThreshold:     cfg.ExcellentThreshold,
		GoodThreshold:          cfg.GoodThreshold,
		AverageThreshold:       cfg.AverageThreshold,
		SuccessReturnPct:       cfg.SuccessReturnPct,
		FailureReturnPct:       cfg.FailureReturnPct,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return quality.Config{}, fmt.Errorf("quality config: %w", err)
	}
	return out, nil
}
