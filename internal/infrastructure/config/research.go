package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/research"
)

// ResearchConfig groups Stage 4 research repository and reporting settings.
type ResearchConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	ExportDirectory  string   `mapstructure:"export_directory"`
	Formats          []string `mapstructure:"formats"`
	SubscriberBuffer int      `mapstructure:"subscriber_buffer"`
}

// ResearchEngineConfig is the validated research configuration used by DI wiring.
type ResearchEngineConfig struct {
	Enabled          bool
	ExportDirectory  string
	Formats          []string
	SubscriberBuffer int
}

// ResearchEngineSettings maps research settings.
func (c *Config) ResearchEngineSettings() ResearchEngineConfig {
	return ResearchEngineConfig{
		Enabled:          c.Research.Enabled,
		ExportDirectory:  c.Research.ExportDirectory,
		Formats:          append([]string(nil), c.Research.Formats...),
		SubscriberBuffer: c.Research.SubscriberBuffer,
	}
}

// BuildResearchEngineConfig maps application config into the research engine config.
func BuildResearchEngineConfig(cfg ResearchEngineConfig) (research.Config, error) {
	out := research.Config{
		Enabled:          cfg.Enabled,
		ExportDirectory:  cfg.ExportDirectory,
		Formats:          cfg.Formats,
		SubscriberBuffer: cfg.SubscriberBuffer,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return research.Config{}, fmt.Errorf("research config: %w", err)
	}
	return out, nil
}
