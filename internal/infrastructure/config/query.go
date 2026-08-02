package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/query"
)

// QueryConfig controls the research query API.
type QueryConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	APIPrefix    string `mapstructure:"api_prefix"`
	DefaultLimit int    `mapstructure:"default_limit"`
	MaxLimit     int    `mapstructure:"max_limit"`
}

// QueryAPISettings maps query API settings.
func (c *Config) QueryAPISettings() QueryConfig {
	return c.Query
}

// BuildQueryAPIConfig maps application config into the query API config.
func BuildQueryAPIConfig(cfg QueryConfig) (query.Config, error) {
	out := query.Config{
		Enabled:      cfg.Enabled,
		APIPrefix:    cfg.APIPrefix,
		DefaultLimit: cfg.DefaultLimit,
		MaxLimit:     cfg.MaxLimit,
	}
	if err := out.Validate(); err != nil {
		return query.Config{}, fmt.Errorf("query config: %w", err)
	}
	return out, nil
}
