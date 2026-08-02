package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/portfolio"
)

// BuildPortfolioEngineConfig maps application config into the portfolio engine config.
func BuildPortfolioEngineConfig(cfg PortfolioEngineConfig) (portfolio.Config, error) {
	out := portfolio.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
	}
	if err := out.Validate(); err != nil {
		return portfolio.Config{}, fmt.Errorf("portfolio config: %w", err)
	}
	return out, nil
}
