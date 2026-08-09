package opportunity

import "fmt"

// WeightsConfig holds configurable scoring weights.
type WeightsConfig struct {
	Signal       float64
	Strategy     float64
	Performance  float64
	Optimization float64
	WalkForward  float64
	MonteCarlo   float64
}

// Config controls the opportunity ranking engine.
type Config struct {
	Enabled          bool
	TopN             int
	SubscriberBuffer int
	BuyThreshold     float64
	WatchThreshold   float64
	Weights          WeightsConfig
}

func (c Config) WithDefaults() Config {
	out := c
	if out.TopN <= 0 {
		out.TopN = 20
	}
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.BuyThreshold <= 0 {
		out.BuyThreshold = 0.70
	}
	if out.WatchThreshold <= 0 {
		out.WatchThreshold = 0.40
	}
	if out.Weights.Signal == 0 && out.Weights.Strategy == 0 && out.Weights.Performance == 0 &&
		out.Weights.Optimization == 0 && out.Weights.WalkForward == 0 && out.Weights.MonteCarlo == 0 {
		out.Weights = WeightsConfig{
			Signal:       0.20,
			Strategy:     0.20,
			Performance:  0.15,
			Optimization: 0.15,
			WalkForward:  0.15,
			MonteCarlo:   0.15,
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
		return fmt.Errorf("opportunity: subscriber_buffer must be >= 1")
	}
	if c.TopN < 1 {
		return fmt.Errorf("opportunity: top_n must be >= 1")
	}
	if c.BuyThreshold <= c.WatchThreshold {
		return fmt.Errorf("opportunity: buy_threshold must be greater than watch_threshold")
	}
	total := c.Weights.Signal + c.Weights.Strategy + c.Weights.Performance +
		c.Weights.Optimization + c.Weights.WalkForward + c.Weights.MonteCarlo
	if total <= 0 {
		return fmt.Errorf("opportunity: weights must sum to a positive value")
	}
	return nil
}
