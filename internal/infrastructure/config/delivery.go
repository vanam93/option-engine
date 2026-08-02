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

// DeliveryEngineSettings maps intelligence delivery settings.
func (c *Config) DeliveryEngineSettings() DeliveryConfig {
	return c.Intelligence.Delivery
}

// BuildDeliveryEngineConfig maps application config into the delivery engine config.
func BuildDeliveryEngineConfig(cfg DeliveryConfig) (delivery.Config, error) {
	out := delivery.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
	}
	if err := out.Validate(); err != nil {
		return delivery.Config{}, fmt.Errorf("delivery config: %w", err)
	}
	return out, nil
}
