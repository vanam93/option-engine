package optimization

import "fmt"

// ScoringConfig holds configurable weights for the optimization score formula.
type ScoringConfig struct {
	ProfitFactorWeight float64
	WinRateWeight      float64
	ExpectancyWeight   float64
	DrawdownPenalty    float64
}

// Config controls the optimization engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
	Scoring          ScoringConfig
}

func (c Config) withDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.Scoring.ProfitFactorWeight == 0 &&
		out.Scoring.WinRateWeight == 0 &&
		out.Scoring.ExpectancyWeight == 0 &&
		out.Scoring.DrawdownPenalty == 0 {
		out.Scoring = ScoringConfig{
			ProfitFactorWeight: 0.40,
			WinRateWeight:      0.30,
			ExpectancyWeight:   0.20,
			DrawdownPenalty:    0.10,
		}
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("optimization: subscriber_buffer must be >= 1")
	}
	total := c.Scoring.ProfitFactorWeight + c.Scoring.WinRateWeight +
		c.Scoring.ExpectancyWeight + c.Scoring.DrawdownPenalty
	if total <= 0 {
		return fmt.Errorf("optimization: scoring weights must sum to a positive value")
	}
	return nil
}
