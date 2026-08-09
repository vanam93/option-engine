package experiments

import "fmt"

// ParameterRanges defines discrete sweep values per dimension.
type ParameterRanges struct {
	EMAFast       []int     `mapstructure:"ema_fast"`
	EMASlow       []int     `mapstructure:"ema_slow"`
	RSIPeriod     []int     `mapstructure:"rsi_period"`
	RSIOverbought []float64 `mapstructure:"rsi_overbought"`
	RSIOversold   []float64 `mapstructure:"rsi_oversold"`
	MACDFast      []int     `mapstructure:"macd_fast"`
	MACDSlow      []int     `mapstructure:"macd_slow"`
	MACDSignal    []int     `mapstructure:"macd_signal"`
	MinConfidence []float64 `mapstructure:"min_confidence"`
	MaxPositions  []int     `mapstructure:"max_positions"`
}

// Config controls the experiment and parameter sweep engine.
type Config struct {
	Enabled           bool
	ParallelWorkers   int
	MaxConcurrentRuns int
	SubscriberBuffer  int
	Symbols           []string
	Timeframes        []string
	ParameterRanges   ParameterRanges
	Strategy          string
}

func (c Config) WithDefaults() Config {
	out := c
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
	if c.ParallelWorkers < 1 {
		return fmt.Errorf("experiments: parallel_workers must be >= 1")
	}
	if c.MaxConcurrentRuns < 1 {
		return fmt.Errorf("experiments: max_concurrent_runs must be >= 1")
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("experiments: subscriber_buffer must be >= 1")
	}
	if len(c.Symbols) == 0 {
		return fmt.Errorf("experiments: at least one symbol required")
	}
	if len(c.Timeframes) == 0 {
		return fmt.Errorf("experiments: at least one timeframe required")
	}
	return nil
}
