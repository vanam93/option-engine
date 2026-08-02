package config

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/backtest"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// BacktestConfig controls historical replay mode.
type BacktestConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Speed     string   `mapstructure:"speed"`
	Symbols   []string `mapstructure:"symbols"`
	StartTime string   `mapstructure:"start_time"`
	EndTime   string   `mapstructure:"end_time"`
	DataPath  string   `mapstructure:"data_path"`
	Timeframe string   `mapstructure:"timeframe"`
}

// BuildBacktestConfig maps application config into the backtest engine config.
func BuildBacktestConfig(cfg BacktestConfig) (backtest.Config, error) {
	speed, err := backtest.ParseSpeed(cfg.Speed)
	if err != nil {
		return backtest.Config{}, fmt.Errorf("backtest speed: %w", err)
	}

	var start, end time.Time
	if cfg.StartTime != "" {
		start, err = time.Parse(time.RFC3339, cfg.StartTime)
		if err != nil {
			return backtest.Config{}, fmt.Errorf("backtest start_time: %w", err)
		}
	}
	if cfg.EndTime != "" {
		end, err = time.Parse(time.RFC3339, cfg.EndTime)
		if err != nil {
			return backtest.Config{}, fmt.Errorf("backtest end_time: %w", err)
		}
	}

	tf := market.Timeframe(cfg.Timeframe)
	if tf == "" {
		tf = market.TF1m
	}

	out := backtest.Config{
		Enabled:   cfg.Enabled,
		Speed:     speed,
		Symbols:   append([]string(nil), cfg.Symbols...),
		StartTime: start,
		EndTime:   end,
		DataPath:  cfg.DataPath,
		Timeframe: tf,
	}
	if err := out.Validate(); err != nil {
		return backtest.Config{}, fmt.Errorf("backtest config: %w", err)
	}
	return out, nil
}

// ToProviderConfig maps backtest settings for the backtest provider factory.
func (c BacktestConfig) ToProviderConfig() map[string]any {
	speed, _ := backtest.ParseSpeed(c.Speed)
	return map[string]any{
		"speed":      speed,
		"symbols":    append([]string(nil), c.Symbols...),
		"start_time": c.StartTime,
		"end_time":   c.EndTime,
		"data_path":  c.DataPath,
		"timeframe":  c.Timeframe,
	}
}
