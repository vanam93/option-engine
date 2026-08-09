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

// FeedbackEngineSettings maps feedback settings from application config.
// Prefers quality.feedback when present; falls back to intelligence.feedback.
func (c *Config) FeedbackEngineSettings() FeedbackConfig {
	if hasFeedbackSettings(c.Quality.Feedback) {
		return c.Quality.Feedback
	}
	return c.Intelligence.Feedback
}

func hasFeedbackSettings(cfg FeedbackConfig) bool {
	return cfg.Enabled || cfg.SubscriberBuffer != 0 || len(cfg.RollingWindows) > 0 || len(cfg.ConfidenceBuckets) > 0
}

// BuildFeedbackEngineConfig maps application config into the feedback engine config.
func BuildFeedbackEngineConfig(cfg FeedbackConfig) (feedback.Config, error) {
	out := feedback.Config{
		Enabled:           cfg.Enabled,
		SubscriberBuffer:  cfg.SubscriberBuffer,
		RollingWindows:    cfg.RollingWindows,
		ConfidenceBuckets: cfg.ConfidenceBuckets,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return feedback.Config{}, fmt.Errorf("feedback config: %w", err)
	}
	return out, nil
}
