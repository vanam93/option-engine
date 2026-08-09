package indicatorstore

import (
	"fmt"
	"sync"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

type macdKey struct {
	fast, slow, signal int
}

// Store lazily computes and caches indicator series over an immutable candle slice.
type Store struct {
	candles []market.Candle
	mu      sync.RWMutex

	ema          map[int][]indicators.Result
	sma          map[int][]indicators.Result
	rsi          map[int][]indicators.Result
	atr          map[int][]indicators.Result
	macd         map[macdKey][]indicators.MACDResult
	bollinger    map[string][]indicators.BollingerResult
	adx          map[int][]indicators.ADXResult
	donchian     map[int][]indicators.DonchianResult
	supertrend   map[string][]indicators.SuperTrendResult
	openingRange map[int][]indicators.OpeningRangeResult
	sessionVWAP  []indicators.VWAPResult
}

// New creates an indicator store over a shared candle slice.
func New(candles []market.Candle) *Store {
	return &Store{
		candles:      candles,
		ema:          make(map[int][]indicators.Result),
		sma:          make(map[int][]indicators.Result),
		rsi:          make(map[int][]indicators.Result),
		atr:          make(map[int][]indicators.Result),
		macd:         make(map[macdKey][]indicators.MACDResult),
		bollinger:    make(map[string][]indicators.BollingerResult),
		adx:          make(map[int][]indicators.ADXResult),
		donchian:     make(map[int][]indicators.DonchianResult),
		supertrend:   make(map[string][]indicators.SuperTrendResult),
		openingRange: make(map[int][]indicators.OpeningRangeResult),
	}
}

// Len returns the number of candles in the dataset.
func (s *Store) Len() int {
	return len(s.candles)
}

// Candles returns the immutable candle slice backing this store.
func (s *Store) Candles() []market.Candle {
	return s.candles
}

func (s *Store) EMA(period, bar int) (indicators.Result, bool) {
	series := s.ensureEMA(period)
	return atBar(series, bar)
}

func (s *Store) SMA(period, bar int) (indicators.Result, bool) {
	series := s.ensureSMA(period)
	return atBar(series, bar)
}

func (s *Store) RSI(period, bar int) (indicators.Result, bool) {
	series := s.ensureRSI(period)
	return atBar(series, bar)
}

func (s *Store) ATR(period, bar int) (indicators.Result, bool) {
	series := s.ensureATR(period)
	return atBar(series, bar)
}

func (s *Store) MACD(fast, slow, signal, bar int) (indicators.MACDResult, bool) {
	key := macdKey{fast: fast, slow: slow, signal: signal}
	series := s.ensureMACD(key)
	return atBarMACD(series, bar)
}

func (s *Store) Bollinger(period int, stddev float64, bar int) (indicators.BollingerResult, bool) {
	key := bollingerKey(period, stddev)
	series := s.ensureBollinger(key)
	return atBarBollinger(series, bar)
}

func (s *Store) ADX(period, bar int) (indicators.ADXResult, bool) {
	series := s.ensureADX(period)
	return atBarADX(series, bar)
}

func (s *Store) Donchian(period, bar int) (indicators.DonchianResult, bool) {
	series := s.ensureDonchian(period)
	return atBarDonchian(series, bar)
}

func (s *Store) SuperTrend(atrPeriod int, multiplier float64, bar int) (indicators.SuperTrendResult, bool) {
	key := supertrendKey(atrPeriod, multiplier)
	series := s.ensureSuperTrend(key)
	return atBarSuperTrend(series, bar)
}

func (s *Store) OpeningRange(windowMinutes, bar int) (indicators.OpeningRangeResult, bool) {
	series := s.ensureOpeningRange(windowMinutes)
	return atBarOpeningRange(series, bar)
}

func (s *Store) SessionVWAP(bar int) (indicators.VWAPResult, bool) {
	series := s.ensureSessionVWAP()
	return atBarVWAP(series, bar)
}

func (s *Store) ensureEMA(period int) []indicators.Result {
	s.mu.RLock()
	if series, ok := s.ema[period]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.ema[period]; ok {
		return series
	}
	series := computeEMASeries(s.candles, period)
	s.ema[period] = series
	return series
}

func (s *Store) ensureSMA(period int) []indicators.Result {
	s.mu.RLock()
	if series, ok := s.sma[period]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.sma[period]; ok {
		return series
	}
	series := computeSMASeries(s.candles, period)
	s.sma[period] = series
	return series
}

func (s *Store) ensureRSI(period int) []indicators.Result {
	s.mu.RLock()
	if series, ok := s.rsi[period]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.rsi[period]; ok {
		return series
	}
	series := computeRSISeries(s.candles, period)
	s.rsi[period] = series
	return series
}

func (s *Store) ensureATR(period int) []indicators.Result {
	s.mu.RLock()
	if series, ok := s.atr[period]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.atr[period]; ok {
		return series
	}
	series := computeATRSeries(s.candles, period)
	s.atr[period] = series
	return series
}

func (s *Store) ensureMACD(key macdKey) []indicators.MACDResult {
	s.mu.RLock()
	if series, ok := s.macd[key]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.macd[key]; ok {
		return series
	}
	series := computeMACDSeries(s.candles, key.fast, key.slow, key.signal)
	s.macd[key] = series
	return series
}

func (s *Store) ensureBollinger(key string) []indicators.BollingerResult {
	s.mu.RLock()
	if series, ok := s.bollinger[key]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.bollinger[key]; ok {
		return series
	}
	period, stddev := parseBollingerKey(key)
	series := computeBollingerSeries(s.candles, period, stddev)
	s.bollinger[key] = series
	return series
}

func (s *Store) ensureADX(period int) []indicators.ADXResult {
	s.mu.RLock()
	if series, ok := s.adx[period]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.adx[period]; ok {
		return series
	}
	series := computeADXSeries(s.candles, period)
	s.adx[period] = series
	return series
}

func (s *Store) ensureDonchian(period int) []indicators.DonchianResult {
	s.mu.RLock()
	if series, ok := s.donchian[period]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.donchian[period]; ok {
		return series
	}
	series := computeDonchianSeries(s.candles, period)
	s.donchian[period] = series
	return series
}

func (s *Store) ensureSuperTrend(key string) []indicators.SuperTrendResult {
	s.mu.RLock()
	if series, ok := s.supertrend[key]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.supertrend[key]; ok {
		return series
	}
	atrPeriod, multiplier := parseSuperTrendKey(key)
	series := computeSuperTrendSeries(s.candles, atrPeriod, multiplier)
	s.supertrend[key] = series
	return series
}

func (s *Store) ensureOpeningRange(windowMinutes int) []indicators.OpeningRangeResult {
	s.mu.RLock()
	if series, ok := s.openingRange[windowMinutes]; ok {
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.openingRange[windowMinutes]; ok {
		return series
	}
	series := computeOpeningRangeSeries(s.candles, windowMinutes)
	s.openingRange[windowMinutes] = series
	return series
}

func (s *Store) ensureSessionVWAP() []indicators.VWAPResult {
	s.mu.RLock()
	if s.sessionVWAP != nil {
		series := s.sessionVWAP
		s.mu.RUnlock()
		return series
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessionVWAP != nil {
		return s.sessionVWAP
	}
	s.sessionVWAP = computeSessionVWAPSeries(s.candles)
	return s.sessionVWAP
}

func computeEMASeries(candles []market.Candle, period int) []indicators.Result {
	e := indicators.NewEMA(period)
	out := make([]indicators.Result, len(candles))
	for i, c := range candles {
		out[i] = e.Update(c.Close)
	}
	return out
}

func computeSMASeries(candles []market.Candle, period int) []indicators.Result {
	sma := indicators.NewSMA(period)
	out := make([]indicators.Result, len(candles))
	for i, c := range candles {
		out[i] = sma.Update(c.Close)
	}
	return out
}

func computeRSISeries(candles []market.Candle, period int) []indicators.Result {
	r := indicators.NewRSI(period)
	out := make([]indicators.Result, len(candles))
	for i, c := range candles {
		out[i] = r.Update(c.Close)
	}
	return out
}

func computeATRSeries(candles []market.Candle, period int) []indicators.Result {
	a := indicators.NewATR(period)
	out := make([]indicators.Result, len(candles))
	for i, c := range candles {
		out[i] = a.Update(c.High, c.Low, c.Close)
	}
	return out
}

func computeMACDSeries(candles []market.Candle, fast, slow, signal int) []indicators.MACDResult {
	m := indicators.NewMACD(fast, slow, signal)
	out := make([]indicators.MACDResult, len(candles))
	for i, c := range candles {
		out[i] = m.Update(c.Close)
	}
	return out
}

func computeBollingerSeries(candles []market.Candle, period int, stddev float64) []indicators.BollingerResult {
	b := indicators.NewBollinger(period, stddev)
	out := make([]indicators.BollingerResult, len(candles))
	for i, c := range candles {
		out[i] = b.Update(c.Close)
	}
	return out
}

func computeADXSeries(candles []market.Candle, period int) []indicators.ADXResult {
	a := indicators.NewADX(period)
	out := make([]indicators.ADXResult, len(candles))
	for i, c := range candles {
		out[i] = a.Update(c.High, c.Low, c.Close)
	}
	return out
}

func computeDonchianSeries(candles []market.Candle, period int) []indicators.DonchianResult {
	d := indicators.NewDonchianChannel(period)
	out := make([]indicators.DonchianResult, len(candles))
	for i, c := range candles {
		out[i] = d.Update(c.High, c.Low)
	}
	return out
}

func computeSuperTrendSeries(candles []market.Candle, atrPeriod int, multiplier float64) []indicators.SuperTrendResult {
	st := indicators.NewSuperTrend(atrPeriod, multiplier)
	out := make([]indicators.SuperTrendResult, len(candles))
	for i, c := range candles {
		out[i] = st.Update(c.High, c.Low, c.Close)
	}
	return out
}

func computeOpeningRangeSeries(candles []market.Candle, windowMinutes int) []indicators.OpeningRangeResult {
	or := indicators.NewOpeningRange(windowMinutes)
	out := make([]indicators.OpeningRangeResult, len(candles))
	for i, c := range candles {
		out[i] = or.Update(c.OpenTime, c.High, c.Low)
	}
	return out
}

func computeSessionVWAPSeries(candles []market.Candle) []indicators.VWAPResult {
	v := indicators.NewSessionVWAP()
	out := make([]indicators.VWAPResult, len(candles))
	for i, c := range candles {
		out[i] = v.Update(c.OpenTime, c.High, c.Low, c.Close, c.Volume)
	}
	return out
}

func bollingerKey(period int, stddev float64) string {
	return fmt.Sprintf("%d:%.6f", period, stddev)
}

func parseBollingerKey(key string) (period int, stddev float64) {
	var p int
	var sd float64
	fmt.Sscanf(key, "%d:%f", &p, &sd)
	return p, sd
}

func supertrendKey(atrPeriod int, multiplier float64) string {
	return fmt.Sprintf("%d:%.6f", atrPeriod, multiplier)
}

func parseSuperTrendKey(key string) (atrPeriod int, multiplier float64) {
	var p int
	var m float64
	fmt.Sscanf(key, "%d:%f", &p, &m)
	return p, m
}

func atBar(series []indicators.Result, bar int) (indicators.Result, bool) {
	if bar < 0 || bar >= len(series) {
		return indicators.Result{}, false
	}
	return series[bar], true
}

func atBarMACD(series []indicators.MACDResult, bar int) (indicators.MACDResult, bool) {
	if bar < 0 || bar >= len(series) {
		return indicators.MACDResult{}, false
	}
	return series[bar], true
}

func atBarBollinger(series []indicators.BollingerResult, bar int) (indicators.BollingerResult, bool) {
	if bar < 0 || bar >= len(series) {
		return indicators.BollingerResult{}, false
	}
	return series[bar], true
}

func atBarADX(series []indicators.ADXResult, bar int) (indicators.ADXResult, bool) {
	if bar < 0 || bar >= len(series) {
		return indicators.ADXResult{}, false
	}
	return series[bar], true
}

func atBarDonchian(series []indicators.DonchianResult, bar int) (indicators.DonchianResult, bool) {
	if bar < 0 || bar >= len(series) {
		return indicators.DonchianResult{}, false
	}
	return series[bar], true
}

func atBarSuperTrend(series []indicators.SuperTrendResult, bar int) (indicators.SuperTrendResult, bool) {
	if bar < 0 || bar >= len(series) {
		return indicators.SuperTrendResult{}, false
	}
	return series[bar], true
}

func atBarOpeningRange(series []indicators.OpeningRangeResult, bar int) (indicators.OpeningRangeResult, bool) {
	if bar < 0 || bar >= len(series) {
		return indicators.OpeningRangeResult{}, false
	}
	return series[bar], true
}

func atBarVWAP(series []indicators.VWAPResult, bar int) (indicators.VWAPResult, bool) {
	if bar < 0 || bar >= len(series) {
		return indicators.VWAPResult{}, false
	}
	return series[bar], true
}

// Ensure Store implements strategylib.IndicatorSource.
var _ strategylib.IndicatorSource = (*Store)(nil)
