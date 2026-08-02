package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/aicontext"
)

// AIContextConfig controls the AI context builder engine.
type AIContextConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	ExecutivePrompt bool `mapstructure:"executive_prompt"`
	TechnicalPrompt bool `mapstructure:"technical_prompt"`
	JSONPrompt      bool `mapstructure:"json_prompt"`
}

// AIContextEngineSettings maps AI context settings.
func (c *Config) AIContextEngineSettings() AIContextConfig {
	return c.AIContext
}

// BuildAIContextEngineConfig maps application config into the AI context engine config.
func BuildAIContextEngineConfig(cfg AIContextConfig) (aicontext.Config, error) {
	out := aicontext.Config{
		Enabled:         cfg.Enabled,
		ExecutivePrompt: cfg.ExecutivePrompt,
		TechnicalPrompt: cfg.TechnicalPrompt,
		JSONPrompt:      cfg.JSONPrompt,
	}
	if err := out.Validate(); err != nil {
		return aicontext.Config{}, fmt.Errorf("aicontext config: %w", err)
	}
	return out, nil
}
