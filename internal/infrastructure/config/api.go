package config

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/api"
)

// APIConfig controls the Intelligence API layer.
type APIConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	Prefix       string        `mapstructure:"prefix"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	DefaultLimit int           `mapstructure:"default_limit"`
	MaxLimit     int           `mapstructure:"max_limit"`
}

// APISettings maps Intelligence API settings.
func (c *Config) APISettings() APIConfig {
	return c.API
}

// BuildAPIConfig maps application config into the Intelligence API config.
func BuildAPIConfig(cfg APIConfig) (api.Config, error) {
	out := api.Config{
		Enabled:      cfg.Enabled,
		Prefix:       cfg.Prefix,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		DefaultLimit: cfg.DefaultLimit,
		MaxLimit:     cfg.MaxLimit,
	}
	if err := out.Validate(); err != nil {
		return api.Config{}, fmt.Errorf("api config: %w", err)
	}
	return out, nil
}
