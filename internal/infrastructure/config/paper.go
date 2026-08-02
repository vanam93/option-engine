package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/execution/paper"
)

// BuildPaperExecutionConfig maps application config into the paper execution engine config.
func BuildPaperExecutionConfig(cfg PaperExecutionEngineConfig) (paper.Config, error) {
	out := paper.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
		SlippagePercent:  cfg.SlippagePercent,
		DefaultPrice:     cfg.DefaultPrice,
	}
	if err := out.Validate(); err != nil {
		return paper.Config{}, fmt.Errorf("paper execution config: %w", err)
	}
	return out, nil
}
