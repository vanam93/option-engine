package walkforward

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/experiments"
)

// Config controls the walk-forward analysis engine.
type Config struct {
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

func (c Config) withDefaults() Config {
	out := c
	if out.TrainWindowDays <= 0 {
		out.TrainWindowDays = 30
	}
	if out.ValidationWindowDays <= 0 {
		out.ValidationWindowDays = 10
	}
	if out.StepDays <= 0 {
		out.StepDays = 10
	}
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.ParallelWorkers <= 0 {
		out.ParallelWorkers = 2
	}
	if out.MaxConcurrentRuns <= 0 {
		out.MaxConcurrentRuns = out.ParallelWorkers
	}
	out.Experiments = applyExperimentDefaults(out.Experiments)
	return out
}

func applyExperimentDefaults(cfg experiments.Config) experiments.Config {
	out := cfg
	if out.ParallelWorkers <= 0 {
		out.ParallelWorkers = 4
	}
	if out.MaxConcurrentRuns <= 0 {
		out.MaxConcurrentRuns = out.ParallelWorkers
	}
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if len(out.Symbols) == 0 {
		out.Symbols = []string{"NIFTY"}
	}
	if len(out.Timeframes) == 0 {
		out.Timeframes = []string{"5m"}
	}
	if out.Strategy == "" {
		out.Strategy = "trend_following"
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.TrainWindowDays < 1 {
		return fmt.Errorf("walkforward: train_window_days must be >= 1")
	}
	if c.ValidationWindowDays < 1 {
		return fmt.Errorf("walkforward: validation_window_days must be >= 1")
	}
	if c.StepDays < 1 {
		return fmt.Errorf("walkforward: step_days must be >= 1")
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("walkforward: subscriber_buffer must be >= 1")
	}
	if c.ParallelWorkers < 1 {
		return fmt.Errorf("walkforward: parallel_workers must be >= 1")
	}
	if c.MaxConcurrentRuns < 1 {
		return fmt.Errorf("walkforward: max_concurrent_runs must be >= 1")
	}
	if c.DataStart.IsZero() || c.DataEnd.IsZero() {
		return fmt.Errorf("walkforward: data_start and data_end are required")
	}
	if !c.DataEnd.After(c.DataStart) {
		return fmt.Errorf("walkforward: data_end must be after data_start")
	}
	minSpan := c.TrainWindowDays + c.ValidationWindowDays
	if daysBetween(c.DataStart, c.DataEnd) < minSpan {
		return fmt.Errorf("walkforward: data range spans %d days, need at least %d", daysBetween(c.DataStart, c.DataEnd), minSpan)
	}
	return c.Experiments.Validate()
}

func daysBetween(start, end time.Time) int {
	if end.Before(start) {
		return 0
	}
	return int(end.Sub(start).Hours() / 24)
}
