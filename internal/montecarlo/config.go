package montecarlo

import "fmt"

// Config controls the Monte Carlo simulation engine.
type Config struct {
	Enabled          bool
	Simulations      int
	ConfidenceLevel  float64
	RandomSeed       *int64
	SubscriberBuffer int
	RuinDrawdownPct  float64
}

func (c Config) WithDefaults() Config {
	out := c
	if out.Simulations <= 0 {
		out.Simulations = 1000
	}
	if out.ConfidenceLevel <= 0 || out.ConfidenceLevel >= 1 {
		out.ConfidenceLevel = 0.95
	}
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.RuinDrawdownPct <= 0 {
		out.RuinDrawdownPct = 1.0
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Simulations < 1 {
		return fmt.Errorf("montecarlo: simulations must be >= 1")
	}
	if c.ConfidenceLevel <= 0 || c.ConfidenceLevel >= 1 {
		return fmt.Errorf("montecarlo: confidence_level must be in (0, 1)")
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("montecarlo: subscriber_buffer must be >= 1")
	}
	if c.RuinDrawdownPct <= 0 {
		return fmt.Errorf("montecarlo: ruin_drawdown_pct must be > 0")
	}
	return nil
}
