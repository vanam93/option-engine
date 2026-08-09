package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/montecarlo"
)

// MonteCarloConfig groups Stage 4 Monte Carlo simulation engine settings.
type MonteCarloConfig struct {
	Enabled          bool    `mapstructure:"enabled"`
	Simulations      int     `mapstructure:"simulations"`
	ConfidenceLevel  float64 `mapstructure:"confidence_level"`
	RandomSeed       *int64  `mapstructure:"random_seed"`
	SubscriberBuffer int     `mapstructure:"subscriber_buffer"`
	RuinDrawdownPct  float64 `mapstructure:"ruin_drawdown_pct"`
}

// MonteCarloEngineConfig is the validated Monte Carlo configuration used by DI wiring.
type MonteCarloEngineConfig struct {
	Enabled          bool
	Simulations      int
	ConfidenceLevel  float64
	RandomSeed       *int64
	SubscriberBuffer int
	RuinDrawdownPct  float64
}

// MonteCarloEngineSettings maps Monte Carlo settings.
func (c *Config) MonteCarloEngineSettings() MonteCarloEngineConfig {
	return MonteCarloEngineConfig{
		Enabled:          c.MonteCarlo.Enabled,
		Simulations:      c.MonteCarlo.Simulations,
		ConfidenceLevel:  c.MonteCarlo.ConfidenceLevel,
		RandomSeed:       c.MonteCarlo.RandomSeed,
		SubscriberBuffer: c.MonteCarlo.SubscriberBuffer,
		RuinDrawdownPct:  c.MonteCarlo.RuinDrawdownPct,
	}
}

// BuildMonteCarloEngineConfig maps application config into the Monte Carlo engine config.
func BuildMonteCarloEngineConfig(cfg MonteCarloEngineConfig) (montecarlo.Config, error) {
	out := montecarlo.Config{
		Enabled:          cfg.Enabled,
		Simulations:      cfg.Simulations,
		ConfidenceLevel:  cfg.ConfidenceLevel,
		RandomSeed:       cfg.RandomSeed,
		SubscriberBuffer: cfg.SubscriberBuffer,
		RuinDrawdownPct:  cfg.RuinDrawdownPct,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return montecarlo.Config{}, fmt.Errorf("montecarlo config: %w", err)
	}
	return out, nil
}
