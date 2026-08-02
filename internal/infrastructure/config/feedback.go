package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/feedback"
)

// FeedbackConfig controls the recommendation feedback engine.
type FeedbackConfig struct {
	Enabled           bool      `mapstructure:"enabled"`
	SubscriberBuffer  int       `mapstructure:"subscriber_buffer"`
	RollingWindows    []int     `mapstructure:"rolling_windows"`
	ConfidenceBuckets []float64 `mapstructure:"confidence_buckets"`
}

// FeedbackEngineSettings maps intelligence feedback settings.
func (c *Config) FeedbackEngineSettings() FeedbackConfig {
	return c.Intelligence.Feedback
}

// BuildFeedbackEngineConfig maps application config into the feedback engine config.
func BuildFeedbackEngineConfig(cfg FeedbackConfig) (feedback.Config, error) {
	out := feedback.Config{
		Enabled:           cfg.Enabled,
		SubscriberBuffer:  cfg.SubscriberBuffer,
		RollingWindows:    cfg.RollingWindows,
		ConfidenceBuckets: cfg.ConfidenceBuckets,
	}
	if err := out.Validate(); err != nil {
		return feedback.Config{}, fmt.Errorf("feedback config: %w", err)
	}
	return out, nil
}
