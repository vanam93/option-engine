package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/quality"
)

// QualityConfig controls the recommendation quality engine.
type QualityConfig struct {
	Enabled                bool    `mapstructure:"enabled"`
	SubscriberBuffer       int     `mapstructure:"subscriber_buffer"`
	TrackingTimeoutMinutes int     `mapstructure:"tracking_timeout_minutes"`
	ExcellentThreshold     float64 `mapstructure:"excellent_threshold"`
	GoodThreshold          float64 `mapstructure:"good_threshold"`
	AverageThreshold       float64 `mapstructure:"average_threshold"`
}

// QualityEngineSettings maps intelligence quality settings.
func (c *Config) QualityEngineSettings() QualityConfig {
	return c.Intelligence.Quality
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
	}
	if err := out.Validate(); err != nil {
		return quality.Config{}, fmt.Errorf("quality config: %w", err)
	}
	return out, nil
}
