package intelligence

import "fmt"

// Config controls the recommendation intelligence engine.
type Config struct {
	Enabled                    bool
	SubscriberBuffer           int
	IncludeTimeline            bool
	IncludeResearch            bool
	IncludeConfidenceBreakdown bool
	StrongBuyThreshold         float64
	BuyThreshold               float64
	WatchThreshold             float64
}

func (c Config) withDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.StrongBuyThreshold <= 0 {
		out.StrongBuyThreshold = 0.85
	}
	if out.BuyThreshold <= 0 {
		out.BuyThreshold = 0.70
	}
	if out.WatchThreshold <= 0 {
		out.WatchThreshold = 0.40
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("intelligence: subscriber_buffer must be >= 1")
	}
	if c.StrongBuyThreshold <= c.BuyThreshold || c.BuyThreshold <= c.WatchThreshold {
		return fmt.Errorf("intelligence: recommendation thresholds must satisfy strong_buy > buy > watch")
	}
	return nil
}
