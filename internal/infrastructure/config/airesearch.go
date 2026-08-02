package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/airesearch"
)

// AIResearchConfig controls the AI research engine.
type AIResearchConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Analyzer string `mapstructure:"analyzer"`
}

// AIResearchEngineSettings maps AI research settings.
func (c *Config) AIResearchEngineSettings() AIResearchConfig {
	return c.AIResearch
}

// BuildAIResearchEngineConfig maps application config into the AI research engine config.
func BuildAIResearchEngineConfig(cfg AIResearchConfig) (airesearch.Config, error) {
	out := airesearch.Config{
		Enabled:  cfg.Enabled,
		Analyzer: cfg.Analyzer,
	}
	if err := out.Validate(); err != nil {
		return airesearch.Config{}, fmt.Errorf("airesearch config: %w", err)
	}
	return out, nil
}
