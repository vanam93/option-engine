package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/delivery"
)

// DeliveryConfig controls the recommendation delivery engine.
type DeliveryConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	SubscriberBuffer int  `mapstructure:"subscriber_buffer"`
}

// DeliveryEngineSettings maps delivery settings from application config.
// Prefers quality.delivery when present; falls back to intelligence.delivery.
func (c *Config) DeliveryEngineSettings() DeliveryConfig {
	if hasDeliverySettings(c.Quality.Delivery) {
		return c.Quality.Delivery
	}
	return c.Intelligence.Delivery
}

func hasDeliverySettings(cfg DeliveryConfig) bool {
	return cfg.Enabled || cfg.SubscriberBuffer != 0
}

// BuildDeliveryEngineConfig maps application config into the delivery engine config.
func BuildDeliveryEngineConfig(cfg DeliveryConfig) (delivery.Config, error) {
	out := delivery.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return delivery.Config{}, fmt.Errorf("delivery config: %w", err)
	}
	return out, nil
}
