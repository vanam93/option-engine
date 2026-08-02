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
	ema emaSet
	sma smaSet
	rsi rsiSet
	atr atrSet
}

// Cache stores incremental indicator state per symbol and timeframe.
type Cache struct {
	mu     sync.RWMutex
	emaCfg []int
	smaCfg []int
	rsiCfg []int
	atrCfg []int
	series map[seriesKey]*seriesState
}

// NewCache creates indicator state storage from configuration.
func NewCache(cfg Config) *Cache {
	return &Cache{
		emaCfg: cfg.EMAPeriods(),
		smaCfg: cfg.SMAPeriods(),
		rsiCfg: cfg.RSIPeriods(),
		atrCfg: cfg.ATRPeriods(),
		series: make(map[seriesKey]*seriesState),
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
	return state
}

// ActiveSeries returns the number of tracked symbol/timeframe pairs.
func (c *Cache) ActiveSeries() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.series)
}
