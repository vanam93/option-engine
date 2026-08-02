package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator"
)

// BuildIndicatorEngineConfig maps application config into the indicator engine config.
func BuildIndicatorEngineConfig(cfg IndicatorEngineConfig) (indicator.Config, error) {
	out := indicator.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
	}
	if err := appendPeriods(&out.EMA, cfg.EMA, "ema"); err != nil {
		return indicator.Config{}, err
	}
	if err := appendPeriods(&out.SMA, cfg.SMA, "sma"); err != nil {
		return indicator.Config{}, err
	}
	if err := appendPeriods(&out.RSI, cfg.RSI, "rsi"); err != nil {
		return indicator.Config{}, err
	}
	if err := appendPeriods(&out.ATR, cfg.ATR, "atr"); err != nil {
		return indicator.Config{}, err
	}
	if cfg.MACD.FastPeriod > 0 || cfg.MACD.SlowPeriod > 0 || cfg.MACD.SignalPeriod > 0 {
		out.MACD = &indicator.MACDConfig{
			FastPeriod:   cfg.MACD.FastPeriod,
			SlowPeriod:   cfg.MACD.SlowPeriod,
			SignalPeriod: cfg.MACD.SignalPeriod,
		}
	}
	if cfg.Bollinger.Period > 0 || cfg.Bollinger.StdDev > 0 {
		out.Bollinger = &indicator.BollingerConfig{
			Period: cfg.Bollinger.Period,
			StdDev: cfg.Bollinger.StdDev,
		}
	}
	return out, nil
}

func appendPeriods(dst *[]indicator.PeriodConfig, src []IndicatorPeriodConfig, name string) error {
	for _, p := range src {
		if p.Period < 1 {
			return fmt.Errorf("indicator %s period must be >= 1", name)
		}
		*dst = append(*dst, indicator.PeriodConfig{Period: p.Period})
	}
	return nil
}
