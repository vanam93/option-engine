package config

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/experiments"
	"github.com/vanam-gangireddy/option-engine/internal/walkforward"
)

// WalkForwardConfig groups Stage 4 walk-forward analysis engine settings.
type WalkForwardConfig struct {
	Enabled              bool      `mapstructure:"enabled"`
	TrainWindowDays      int       `mapstructure:"train_window_days"`
	ValidationWindowDays int       `mapstructure:"validation_window_days"`
	StepDays             int       `mapstructure:"step_days"`
	DataStart            time.Time `mapstructure:"data_start"`
	DataEnd              time.Time `mapstructure:"data_end"`
	SubscriberBuffer     int       `mapstructure:"subscriber_buffer"`
	ParallelWorkers      int       `mapstructure:"parallel_workers"`
	MaxConcurrentRuns    int       `mapstructure:"max_concurrent_runs"`
}

// WalkForwardEngineConfig is the validated walk-forward configuration used by DI wiring.
type WalkForwardEngineConfig struct {
	Enabled              bool
	TrainWindowDays      int
	ValidationWindowDays int
	StepDays             int
	DataStart            time.Time
	DataEnd              time.Time
	SubscriberBuffer     int
	ParallelWorkers      int
	MaxConcurrentRuns    int
	Experiments          experiments.Config
}

// WalkForwardEngineSettings maps walk-forward settings.
func (c *Config) WalkForwardEngineSettings() WalkForwardEngineConfig {
	return WalkForwardEngineConfig{
		Enabled:              c.WalkForward.Enabled,
		TrainWindowDays:      c.WalkForward.TrainWindowDays,
		ValidationWindowDays: c.WalkForward.ValidationWindowDays,
		StepDays:             c.WalkForward.StepDays,
		DataStart:            c.WalkForward.DataStart,
		DataEnd:              c.WalkForward.DataEnd,
		SubscriberBuffer:     c.WalkForward.SubscriberBuffer,
		ParallelWorkers:      c.WalkForward.ParallelWorkers,
		MaxConcurrentRuns:    c.WalkForward.MaxConcurrentRuns,
		Experiments:          c.experimentsConfig(),
	}
}

func (c *Config) experimentsConfig() experiments.Config {
	return experiments.Config{
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

// BuildWalkForwardEngineConfig maps application config into the walk-forward engine config.
func BuildWalkForwardEngineConfig(cfg WalkForwardEngineConfig) (walkforward.Config, error) {
	out := walkforward.Config{
		Enabled:              cfg.Enabled,
		TrainWindowDays:      cfg.TrainWindowDays,
		ValidationWindowDays: cfg.ValidationWindowDays,
		StepDays:             cfg.StepDays,
		DataStart:            cfg.DataStart,
		DataEnd:              cfg.DataEnd,
		SubscriberBuffer:     cfg.SubscriberBuffer,
		ParallelWorkers:      cfg.ParallelWorkers,
		MaxConcurrentRuns:    cfg.MaxConcurrentRuns,
		Experiments:          cfg.Experiments,
	}
	if err := out.Validate(); err != nil {
		return walkforward.Config{}, fmt.Errorf("walkforward config: %w", err)
	}
	return out, nil
}
