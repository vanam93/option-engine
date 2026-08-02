package config

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/backtest"
	"github.com/vanam-gangireddy/option-engine/internal/backtestrunner"
)

// BacktestRunnerConfig controls the historical backtest runner orchestrator.
type BacktestRunnerConfig struct {
	Enabled            bool `mapstructure:"enabled"`
	AutoStart          bool `mapstructure:"auto_start"`
	ConcurrentSessions int  `mapstructure:"concurrent_sessions"`
}

// BacktestRunnerEngineSettings maps backtest runner settings.
func (c *Config) BacktestRunnerEngineSettings() BacktestRunnerConfig {
	return c.BacktestRunner
}

// BuildBacktestRunnerEngineConfig maps application config into the runner engine config.
func BuildBacktestRunnerEngineConfig(cfg BacktestRunnerConfig, subscriberBuffer int) (backtestrunner.Config, error) {
	out := backtestrunner.Config{
		Enabled:            cfg.Enabled,
		AutoStart:          cfg.AutoStart,
		ConcurrentSessions: cfg.ConcurrentSessions,
		SubscriberBuffer:   subscriberBuffer,
	}
	if err := out.Validate(); err != nil {
		return backtestrunner.Config{}, fmt.Errorf("backtest_runner config: %w", err)
	}
	return out, nil
}

// DefaultBacktestSessionRequest builds a session request from backtest configuration.
func (c *Config) DefaultBacktestSessionRequest() (backtestrunner.SessionRequest, error) {
	bt := c.Backtest
	var start, end time.Time
	var err error
	if bt.StartTime != "" {
		start, err = time.Parse(time.RFC3339, bt.StartTime)
		if err != nil {
			return backtestrunner.SessionRequest{}, fmt.Errorf("backtest start_time: %w", err)
		}
	}
	if bt.EndTime != "" {
		end, err = time.Parse(time.RFC3339, bt.EndTime)
		if err != nil {
			return backtestrunner.SessionRequest{}, fmt.Errorf("backtest end_time: %w", err)
		}
	}
	speed, err := parseBacktestSpeed(bt.Speed)
	if err != nil {
		return backtestrunner.SessionRequest{}, err
	}
	return backtestrunner.SessionRequest{
		StartTime: start,
		EndTime:   end,
		Symbols:   append([]string(nil), bt.Symbols...),
		Speed:     speed,
		DataPath:  bt.DataPath,
	}, nil
}

func parseBacktestSpeed(raw string) (float64, error) {
	if raw == "" {
		return 1.0, nil
	}
	speed, err := backtest.ParseSpeed(raw)
	if err != nil {
		return 0, err
	}
	return speed, nil
}
