package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
)

// LaboratoryConfig controls the strategy laboratory orchestrator.
type LaboratoryConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	AutoVersion       bool `mapstructure:"auto_version"`
	ConcurrentStudies int  `mapstructure:"concurrent_studies"`
}

// LaboratoryEngineSettings maps laboratory settings.
func (c *Config) LaboratoryEngineSettings() LaboratoryConfig {
	return c.Laboratory
}

// BuildLaboratoryEngineConfig maps application config into the laboratory engine config.
func BuildLaboratoryEngineConfig(cfg LaboratoryConfig) (laboratory.Config, error) {
	out := laboratory.Config{
		Enabled:           cfg.Enabled,
		AutoVersion:       cfg.AutoVersion,
		ConcurrentStudies: cfg.ConcurrentStudies,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return laboratory.Config{}, fmt.Errorf("laboratory config: %w", err)
	}
	return out, nil
}
