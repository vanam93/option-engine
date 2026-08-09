package config

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/console"
)

// ConsoleConfig controls the recommendation console renderer.
type ConsoleConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
}

// ConsoleEngineSettings maps console settings.
func (c *Config) ConsoleEngineSettings() ConsoleConfig {
	return c.Console
}

// BuildConsoleEngineConfig maps application config into the console engine config.
func BuildConsoleEngineConfig(cfg ConsoleConfig, subscriberBuffer int) (console.Config, error) {
	out := console.Config{
		Enabled:          cfg.Enabled,
		RefreshInterval:  cfg.RefreshInterval,
		SubscriberBuffer: subscriberBuffer,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return console.Config{}, fmt.Errorf("console config: %w", err)
	}
	return out, nil
}
