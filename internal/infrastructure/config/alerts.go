package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/alerts"
)

// AlertsConfig controls the alert engine.
type AlertsConfig struct {
	Enabled                   bool    `mapstructure:"enabled"`
	SubscriberBuffer          int     `mapstructure:"subscriber_buffer"`
	ConfidenceChangeThreshold float64 `mapstructure:"confidence_change_threshold"`
	CooldownSeconds           int     `mapstructure:"cooldown_seconds"`
}

// AlertsEngineSettings maps intelligence alert settings.
func (c *Config) AlertsEngineSettings() AlertsConfig {
	return c.Intelligence.Alerts
}

// BuildAlertsEngineConfig maps application config into the alert engine config.
func BuildAlertsEngineConfig(cfg AlertsConfig) (alerts.Config, error) {
	out := alerts.Config{
		Enabled:                   cfg.Enabled,
		SubscriberBuffer:          cfg.SubscriberBuffer,
		ConfidenceChangeThreshold: cfg.ConfidenceChangeThreshold,
		CooldownSeconds:           cfg.CooldownSeconds,
	}
	if err := out.Validate(); err != nil {
		return alerts.Config{}, fmt.Errorf("alerts config: %w", err)
	}
	return out, nil
}
