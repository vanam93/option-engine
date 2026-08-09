package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/scanner"
)

// ScannerConfig groups Stage 5 market scanner settings.
type ScannerConfig struct {
	Enabled          bool               `mapstructure:"enabled"`
	Symbols          []string           `mapstructure:"symbols"`
	Scanners         ScannerRulesConfig `mapstructure:"scanners"`
	SubscriberBuffer int                `mapstructure:"subscriber_buffer"`
	MinConfidence    float64            `mapstructure:"min_confidence"`
}

// ScannerRulesConfig toggles individual scanner rules.
type ScannerRulesConfig struct {
	EMA     bool `mapstructure:"ema"`
	RSI     bool `mapstructure:"rsi"`
	MACD    bool `mapstructure:"macd"`
	Trend   bool `mapstructure:"trend"`
	Ranking bool `mapstructure:"ranking"`
}

// ScannerEngineConfig is the validated scanner configuration used by DI wiring.
type ScannerEngineConfig struct {
	Enabled          bool
	Symbols          []string
	Scanners         ScannerRulesConfig
	SubscriberBuffer int
	MinConfidence    float64
}

// ScannerEngineSettings maps scanner settings.
func (c *Config) ScannerEngineSettings() ScannerEngineConfig {
	return ScannerEngineConfig{
		Enabled:          c.Scanner.Enabled,
		Symbols:          append([]string(nil), c.Scanner.Symbols...),
		Scanners:         c.Scanner.Scanners,
		SubscriberBuffer: c.Scanner.SubscriberBuffer,
		MinConfidence:    c.Scanner.MinConfidence,
	}
}

// BuildScannerEngineConfig maps application config into the scanner engine config.
func BuildScannerEngineConfig(cfg ScannerEngineConfig) (scanner.Config, error) {
	out := scanner.Config{
		Enabled:          cfg.Enabled,
		Symbols:          cfg.Symbols,
		SubscriberBuffer: cfg.SubscriberBuffer,
		MinConfidence:    cfg.MinConfidence,
		Scanners: scanner.ScannersConfig{
			EMA:     cfg.Scanners.EMA,
			RSI:     cfg.Scanners.RSI,
			MACD:    cfg.Scanners.MACD,
			Trend:   cfg.Scanners.Trend,
			Ranking: cfg.Scanners.Ranking,
		},
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return scanner.Config{}, fmt.Errorf("scanner config: %w", err)
	}
	return out, nil
}
