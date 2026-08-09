package airesearch

import "fmt"

const (
	AnalyzerRuleBased = "rule_based"
)

// Config controls the AI research engine.
type Config struct {
	Enabled          bool
	Analyzer         string
	SubscriberBuffer int
}

func (c Config) WithDefaults() Config {
	out := c
	if out.Analyzer == "" {
		out.Analyzer = AnalyzerRuleBased
	}
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 512
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("airesearch: subscriber_buffer must be >= 1")
	}
	switch c.Analyzer {
	case AnalyzerRuleBased:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrUnknownAnalyzer, c.Analyzer)
	}
}
