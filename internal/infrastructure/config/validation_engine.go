package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/validation"
)

// ValidationEngineConfig controls the recommendation validation engine.
type ValidationEngineConfig struct {
	Enabled              bool    `mapstructure:"enabled"`
	SubscriberBuffer     int     `mapstructure:"subscriber_buffer"`
	MinConfidence        float64 `mapstructure:"min_confidence"`
	MinOptimizationScore float64 `mapstructure:"min_optimization_score"`
	MinWalkforwardScore  float64 `mapstructure:"min_walkforward_score"`
	MinMonteCarloScore   float64 `mapstructure:"min_montecarlo_score"`
	MinWinRate           float64 `mapstructure:"min_win_rate"`
	MaxDrawdown          float64 `mapstructure:"max_drawdown"`
	FreshnessSeconds     int     `mapstructure:"freshness_seconds"`
	SuppressDuplicates   bool    `mapstructure:"suppress_duplicates"`
}

// ValidationEngineSettings maps intelligence validation settings.
func (c *Config) ValidationEngineSettings() ValidationEngineConfig {
	return c.Intelligence.Validation
}

// BuildValidationEngineConfig maps application config into the validation engine config.
func BuildValidationEngineConfig(cfg ValidationEngineConfig) (validation.Config, error) {
	out := validation.Config{
		Enabled:              cfg.Enabled,
		SubscriberBuffer:     cfg.SubscriberBuffer,
		MinConfidence:        cfg.MinConfidence,
		MinOptimizationScore: cfg.MinOptimizationScore,
		MinWalkforwardScore:  cfg.MinWalkforwardScore,
		MinMonteCarloScore:   cfg.MinMonteCarloScore,
		MinWinRate:           cfg.MinWinRate,
		MaxDrawdown:          cfg.MaxDrawdown,
		FreshnessSeconds:     cfg.FreshnessSeconds,
		SuppressDuplicates:   cfg.SuppressDuplicates,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return validation.Config{}, fmt.Errorf("validation config: %w", err)
	}
	return out, nil
}
