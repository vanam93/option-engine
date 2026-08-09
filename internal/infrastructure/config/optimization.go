package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// OptimizationConfig groups Stage 4 optimization engine settings.
type OptimizationConfig struct {
	Enabled          bool                      `mapstructure:"enabled"`
	SubscriberBuffer int                       `mapstructure:"subscriber_buffer"`
	Scoring          OptimizationScoringConfig `mapstructure:"scoring"`
}

// OptimizationScoringConfig holds scoring weight configuration.
type OptimizationScoringConfig struct {
	ProfitFactorWeight float64 `mapstructure:"profit_factor_weight"`
	WinRateWeight      float64 `mapstructure:"win_rate_weight"`
	ExpectancyWeight   float64 `mapstructure:"expectancy_weight"`
	DrawdownPenalty    float64 `mapstructure:"drawdown_penalty"`
}

// OptimizationEngineConfig is the validated optimization configuration used by DI wiring.
type OptimizationEngineConfig struct {
	Enabled          bool
	SubscriberBuffer int
	Scoring          optimization.ScoringConfig
}

// OptimizationEngineSettings maps optimization settings.
func (c *Config) OptimizationEngineSettings() OptimizationEngineConfig {
	return OptimizationEngineConfig{
		Enabled:          c.Optimization.Enabled,
		SubscriberBuffer: c.Optimization.SubscriberBuffer,
		Scoring: optimization.ScoringConfig{
			ProfitFactorWeight: c.Optimization.Scoring.ProfitFactorWeight,
			WinRateWeight:      c.Optimization.Scoring.WinRateWeight,
			ExpectancyWeight:   c.Optimization.Scoring.ExpectancyWeight,
			DrawdownPenalty:    c.Optimization.Scoring.DrawdownPenalty,
		},
	}
}

// BuildOptimizationEngineConfig maps application config into the optimization engine config.
func BuildOptimizationEngineConfig(cfg OptimizationEngineConfig) (optimization.Config, error) {
	out := optimization.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
		Scoring:          cfg.Scoring,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return optimization.Config{}, fmt.Errorf("optimization config: %w", err)
	}
	return out, nil
}
