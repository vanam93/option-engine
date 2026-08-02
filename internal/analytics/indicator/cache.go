package indicator

import (
	"sync"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	domainindicator "github.com/vanam-gangireddy/option-engine/internal/domain/indicator"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

type seriesKey struct {
	symbol    string
	timeframe market.Timeframe
}

type emaSet map[int]*indicators.EMA
type smaSet map[int]*indicators.SMA
type rsiSet map[int]*indicators.RSI
type atrSet map[int]*indicators.ATR

type seriesState struct {
	ema       emaSet
	sma       smaSet
	rsi       rsiSet
	atr       atrSet
	macd      *indicators.MACD
	bollinger *indicators.Bollinger
}

// CacheStats holds runtime indicator instance counters.
type CacheStats struct {
	MACDInstances      int
	BollingerInstances int
	WarmedInstances    int
}

// Cache stores incremental indicator state per symbol and timeframe.
type Cache struct {
	mu           sync.RWMutex
	emaCfg       []int
	smaCfg       []int
	rsiCfg       []int
	atrCfg       []int
	macdCfg      *MACDConfig
	bollingerCfg *BollingerConfig
	series       map[seriesKey]*seriesState
}

// NewCache creates indicator state storage from configuration.
func NewCache(cfg Config) *Cache {
	return &Cache{
		emaCfg:       cfg.EMAPeriods(),
		smaCfg:       cfg.SMAPeriods(),
		rsiCfg:       cfg.RSIPeriods(),
		atrCfg:       cfg.ATRPeriods(),
		macdCfg:      cfg.MACD,
		bollingerCfg: cfg.Bollinger,
		series:       make(map[seriesKey]*seriesState),
	}
}

// Update processes a closed candle and returns warmed indicator values.
func (c *Cache) Update(candle market.Candle) []domainindicator.IndicatorValue {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := seriesKey{symbol: candle.Symbol, timeframe: candle.Timeframe}
	state := c.series[key]
	if state == nil {
		state = c.newSeriesState()
		c.series[key] = state
	}

	var out []domainindicator.IndicatorValue
	for period, ema := range state.ema {
		result := ema.Update(candle.Close)
		if !result.WarmedUp {
			continue
		}
		out = append(out, newIndicatorValue(domainindicator.EMA, candle, period, result))
	}
	for period, sma := range state.sma {
		result := sma.Update(candle.Close)
		if !result.WarmedUp {
			continue
		}
		out = append(out, newIndicatorValue(domainindicator.SMA, candle, period, result))
	}
	for period, rsi := range state.rsi {
		result := rsi.Update(candle.Close)
		if !result.WarmedUp {
			continue
		}
		out = append(out, newIndicatorValue(domainindicator.RSI, candle, period, result))
	}
	for period, atr := range state.atr {
		result := atr.Update(candle.High, candle.Low, candle.Close)
		if !result.WarmedUp {
			continue
		}
		out = append(out, newIndicatorValue(domainindicator.ATR, candle, period, result))
	}
	if state.macd != nil {
		result := state.macd.Update(candle.Close)
		if result.WarmedUp {
			out = append(out, newMACDIndicatorValue(candle, c.macdCfg, result))
		}
	}
	if state.bollinger != nil {
		result := state.bollinger.Update(candle.Close)
		if result.WarmedUp {
			out = append(out, newBollingerIndicatorValue(candle, c.bollingerCfg, result))
		}
	}
	return out
}

func (c *Cache) newSeriesState() *seriesState {
	state := &seriesState{
		ema: make(emaSet, len(c.emaCfg)),
		sma: make(smaSet, len(c.smaCfg)),
		rsi: make(rsiSet, len(c.rsiCfg)),
		atr: make(atrSet, len(c.atrCfg)),
	}
	for _, period := range c.emaCfg {
		state.ema[period] = indicators.NewEMA(period)
	}
	for _, period := range c.smaCfg {
		state.sma[period] = indicators.NewSMA(period)
	}
	for _, period := range c.rsiCfg {
		state.rsi[period] = indicators.NewRSI(period)
	}
	for _, period := range c.atrCfg {
		state.atr[period] = indicators.NewATR(period)
	}
	if c.macdCfg != nil {
		state.macd = indicators.NewMACD(c.macdCfg.FastPeriod, c.macdCfg.SlowPeriod, c.macdCfg.SignalPeriod)
	}
	if c.bollingerCfg != nil {
		state.bollinger = indicators.NewBollinger(c.bollingerCfg.Period, c.bollingerCfg.StdDev)
	}
	return state
}

// ActiveSeries returns the number of tracked symbol/timeframe pairs.
func (c *Cache) ActiveSeries() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.series)
}

// Stats returns runtime instance counters across all tracked series.
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var stats CacheStats
	for _, state := range c.series {
		for _, ema := range state.ema {
			if _, warmed := ema.Value(); warmed {
				stats.WarmedInstances++
			}
		}
		for _, sma := range state.sma {
			if _, warmed := sma.Value(); warmed {
				stats.WarmedInstances++
			}
		}
		for _, rsi := range state.rsi {
			if _, warmed := rsi.Value(); warmed {
				stats.WarmedInstances++
			}
		}
		for _, atr := range state.atr {
			if _, warmed := atr.Value(); warmed {
				stats.WarmedInstances++
			}
		}
		if state.macd != nil {
			stats.MACDInstances++
			if _, _, _, warmed := state.macd.Value(); warmed {
				stats.WarmedInstances++
			}
		}
		if state.bollinger != nil {
			stats.BollingerInstances++
			if _, _, _, _, _, warmed := state.bollinger.Value(); warmed {
				stats.WarmedInstances++
			}
		}
	}
	return stats
}
