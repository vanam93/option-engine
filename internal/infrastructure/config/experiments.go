package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/experiments"
)

// ExperimentsConfig groups Stage 4 experiment engine settings.
type ExperimentsConfig struct {
	Enabled           bool                        `mapstructure:"enabled"`
	ParallelWorkers   int                         `mapstructure:"parallel_workers"`
	MaxConcurrentRuns int                         `mapstructure:"max_concurrent_runs"`
	SubscriberBuffer  int                         `mapstructure:"subscriber_buffer"`
	Symbols           []string                    `mapstructure:"symbols"`
	Timeframes        []string                    `mapstructure:"timeframes"`
	Strategy          string                      `mapstructure:"strategy"`
	ParameterRanges   experiments.ParameterRanges `mapstructure:"parameter_ranges"`
}

// ExperimentsEngineConfig is the validated experiment configuration used by DI wiring.
type ExperimentsEngineConfig struct {
	Enabled           bool
	ParallelWorkers   int
	MaxConcurrentRuns int
	SubscriberBuffer  int
	Symbols           []string
	Timeframes        []string
	Strategy          string
	ParameterRanges   experiments.ParameterRanges
}

// ExperimentsEngineSettings maps experiment settings.
func (c *Config) ExperimentsEngineSettings() ExperimentsEngineConfig {
	return ExperimentsEngineConfig{
		Enabled:           c.Experiments.Enabled,
		ParallelWorkers:   c.Experiments.ParallelWorkers,
		MaxConcurrentRuns: c.Experiments.MaxConcurrentRuns,
		SubscriberBuffer:  c.Experiments.SubscriberBuffer,
		Symbols:           append([]string(nil), c.Experiments.Symbols...),
		Timeframes:        append([]string(nil), c.Experiments.Timeframes...),
		Strategy:          c.Experiments.Strategy,
		ParameterRanges:   c.Experiments.ParameterRanges,
	}
}

// BuildExperimentsEngineConfig maps application config into the experiment engine config.
func BuildExperimentsEngineConfig(cfg ExperimentsEngineConfig) (experiments.Config, error) {
	out := experiments.Config{
		Enabled:           cfg.Enabled,
		ParallelWorkers:   cfg.ParallelWorkers,
		MaxConcurrentRuns: cfg.MaxConcurrentRuns,
		SubscriberBuffer:  cfg.SubscriberBuffer,
		Symbols:           cfg.Symbols,
		Timeframes:        cfg.Timeframes,
		Strategy:          cfg.Strategy,
		ParameterRanges:   cfg.ParameterRanges,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return experiments.Config{}, fmt.Errorf("experiments config: %w", err)
	}
	return out, nil
}
