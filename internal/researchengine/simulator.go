package researchengine

import (
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// Simulator executes strategy signals against historical candles.
type Simulator struct {
	cfg SimulatorConfig
}

// NewSimulator creates a trade simulator.
func NewSimulator(cfg SimulatorConfig) *Simulator {
	return &Simulator{cfg: cfg.withDefaults()}
}

type openPosition struct {
	direction   Direction
	entryPrice  float64
	entryMid    float64
	entryTime   time.Time
	entryBar    int
	quantity    int
	entrySignal strategylib.Signal
	mfe         float64
	mae         float64
	regime      string
}

// Run simulates a single strategy across candles and returns a trade journal.
func (s *Simulator) Run(strategy strategylib.Strategy, candles []market.Candle) *Journal {
	return s.RunWithStore(strategy, candles, nil)
}

// RunWithStore simulates a strategy using an optional shared indicator store.
func (s *Simulator) RunWithStore(strategy strategylib.Strategy, candles []market.Candle, store strategylib.IndicatorSource) *Journal {
	return s.RunWithMetrics(strategy, candles, store, nil)
}

// RunWithMetrics simulates a strategy and optionally records execution metrics.
func (s *Simulator) RunWithMetrics(strategy strategylib.Strategy, candles []market.Candle, store strategylib.IndicatorSource, metrics *SimulationMetrics) *Journal {
	journal := NewJournal()
	if strategy == nil || len(candles) == 0 {
		return journal
	}
	if metrics != nil {
		metrics.Reset()
	}

	meta := strategy.Metadata()
	strategyName := strategy.Name()
	warmup := strategy.WarmupBars()
	useStore := store != nil

	historyCap := warmup * 3
	if historyCap < 64 {
		historyCap = 64
	}
	var history []market.Candle
	var pos openPosition
	hasPos := false

	for i, candle := range candles {
		ctx := strategylib.Context{
			Symbol:         candle.Symbol,
			Timeframe:      string(candle.Timeframe),
			Candle:         candle,
			Position:       positionFromOpen(hasPos, pos.direction),
			Timestamp:      candle.CloseTime,
			BarIndex:       i,
			IndicatorStore: store,
		}
		if !useStore {
			ctx.History = append([]market.Candle(nil), history...)
		}
		if ctx.Timestamp.IsZero() {
			ctx.Timestamp = candle.OpenTime
		}

		if i < warmup {
			_ = strategy.Evaluate(ctx)
			if !useStore {
				history = append(history, candle)
				if len(history) > historyCap {
					history = history[len(history)-historyCap:]
				}
			}
			continue
		}

		if hasPos {
			s.updateExcursion(&pos, candle)
		}

		sig := strategy.Evaluate(ctx)
		if !useStore {
			history = append(history, candle)
			if len(history) > historyCap {
				history = history[len(history)-historyCap:]
			}
		}

		if !sig.IsAction() {
			continue
		}

		switch sig.Decision {
		case strategylib.DecisionBuy:
			if metrics != nil {
				metrics.BuySignals++
			}
			if hasPos && pos.direction == DirectionLong {
				if metrics != nil {
					metrics.IgnoredBuyLong++
				}
				continue
			}
			if hasPos && pos.direction == DirectionShort {
				journal.Add(s.closePosition(strategyName, &pos, candle, i, sig, meta.Version))
				if metrics != nil {
					metrics.Closes++
				}
				hasPos = false
			}
			if !hasPos && s.canOpen(journal) {
				pos = s.openPosition(DirectionLong, candle, i, sig, ctx.MarketRegime)
				hasPos = true
				if metrics != nil {
					metrics.OpensLong++
				}
			}
		case strategylib.DecisionSell:
			if metrics != nil {
				metrics.SellSignals++
			}
			if hasPos && pos.direction == DirectionShort {
				if metrics != nil {
					metrics.IgnoredSellShort++
				}
				continue
			}
			if hasPos && pos.direction == DirectionLong {
				journal.Add(s.closePosition(strategyName, &pos, candle, i, sig, meta.Version))
				if metrics != nil {
					metrics.Closes++
				}
				hasPos = false
			}
			if !hasPos && s.canOpen(journal) {
				pos = s.openPosition(DirectionShort, candle, i, sig, ctx.MarketRegime)
				hasPos = true
				if metrics != nil {
					metrics.OpensShort++
				}
			}
		case strategylib.DecisionExit:
			if metrics != nil {
				metrics.ExitSignals++
			}
			if !hasPos {
				if metrics != nil {
					metrics.IgnoredExitFlat++
				}
				continue
			}
			journal.Add(s.closePosition(strategyName, &pos, candle, i, sig, meta.Version))
			if metrics != nil {
				metrics.Closes++
			}
			hasPos = false
		}
	}

	if hasPos {
		last := candles[len(candles)-1]
		exitSig := strategylib.Signal{
			Decision:    strategylib.DecisionExit,
			Reasons:     []string{"end of data"},
			Parameters:  strategy.Parameters(),
			GeneratedAt: last.CloseTime,
		}
		journal.Add(s.closePosition(strategyName, &pos, last, len(candles)-1, exitSig, meta.Version))
		if metrics != nil {
			metrics.EndOfDataCloses++
			metrics.Closes++
		}
	}

	return journal
}

func positionFromOpen(hasPos bool, dir Direction) strategylib.Position {
	if !hasPos {
		return strategylib.PositionFlat
	}
	if dir == DirectionLong {
		return strategylib.PositionLong
	}
	return strategylib.PositionShort
}

func (s *Simulator) openPosition(dir Direction, candle market.Candle, bar int, sig strategylib.Signal, regime string) openPosition {
	price := s.fillPrice(candle.Close, dir, true)
	return openPosition{
		direction:   dir,
		entryPrice:  price,
		entryMid:    candle.Close,
		entryTime:   candle.CloseTime,
		entryBar:    bar,
		quantity:    s.cfg.Quantity,
		entrySignal: sig,
		mfe:         0,
		mae:         0,
		regime:      regime,
	}
}

func (s *Simulator) canOpen(journal *Journal) bool {
	equity := s.cfg.InitialCapital
	for _, t := range journal.Trades {
		equity += t.NetProfit
	}
	return equity > 0
}

func (s *Simulator) closePosition(strategyName string, pos *openPosition, candle market.Candle, bar int, exitSig strategylib.Signal, version string) SimulatedTrade {
	exitPrice := s.fillPrice(candle.Close, pos.direction, false)
	barsHeld := bar - pos.entryBar
	if barsHeld < 1 {
		barsHeld = 1
	}
	hold := candle.CloseTime.Sub(pos.entryTime)
	if hold < 0 {
		hold = 0
	}

	gross := grossPnL(pos.direction, pos.entryPrice, exitPrice, pos.quantity)
	slipEntry := slippageFromFill(pos.entryPrice, pos.entryMid, pos.quantity)
	slipExit := slippageFromFill(exitPrice, candle.Close, pos.quantity)
	slippage := slipEntry + slipExit
	commission := s.cfg.Commission * 2
	taxes := gross * s.cfg.TaxRate
	if gross < 0 {
		taxes = 0
	}
	// Slippage is already embedded in entry/exit fill prices; do not subtract again.
	net := gross - commission - taxes
	entryValue := pos.entryPrice * float64(pos.quantity)
	retPct := 0.0
	if entryValue > 0 {
		retPct = (net / entryValue) * 100
	}

	exitReason := "signal"
	if len(exitSig.Reasons) > 0 {
		exitReason = exitSig.Reasons[0]
	}

	risk := pos.mae
	reward := pos.mfe
	riskReward := 0.0
	if risk > 0 {
		riskReward = reward / risk
	}

	return SimulatedTrade{
		TradeID:                uuid.New(),
		Strategy:               strategyName,
		StrategyVersion:        version,
		ParameterSet:           strategylib.CloneParams(pos.entrySignal.Parameters),
		EntrySignal:            pos.entrySignal,
		ExitSignal:             exitSig,
		Symbol:                 candle.Symbol,
		Timeframe:              string(candle.Timeframe),
		EntryTime:              pos.entryTime,
		ExitTime:               candle.CloseTime,
		EntryPrice:             pos.entryPrice,
		ExitPrice:              exitPrice,
		Quantity:               pos.quantity,
		Direction:              pos.direction,
		Commission:             commission,
		Taxes:                  taxes,
		Slippage:               slippage,
		GrossProfit:            gross,
		NetProfit:              net,
		ReturnPercent:          retPct,
		MaxFavorableExcursion:  pos.mfe,
		MaxAdverseExcursion:    pos.mae,
		BarsHeld:               barsHeld,
		HoldingDuration:        hold,
		MarketRegime:           pos.regime,
		ExitReason:             exitReason,
		RiskReward:             riskReward,
		ExpectancyContribution: net,
	}
}

func (s *Simulator) updateExcursion(pos *openPosition, candle market.Candle) {
	fav, adv := excursion(pos.direction, pos.entryPrice, candle.High, candle.Low, pos.quantity)
	if fav > pos.mfe {
		pos.mfe = fav
	}
	if adv > pos.mae {
		pos.mae = adv
	}
}

func (s *Simulator) fillPrice(close float64, dir Direction, isEntry bool) float64 {
	pct := s.cfg.SlippagePct / 100
	switch {
	case dir == DirectionLong && isEntry:
		return close * (1 + pct)
	case dir == DirectionLong && !isEntry:
		return close * (1 - pct)
	case dir == DirectionShort && isEntry:
		return close * (1 - pct)
	default:
		return close * (1 + pct)
	}
}

func grossPnL(dir Direction, entry, exit float64, qty int) float64 {
	q := float64(qty)
	if dir == DirectionLong {
		return (exit - entry) * q
	}
	return (entry - exit) * q
}

func slippageFromFill(fill, mid float64, qty int) float64 {
	diff := fill - mid
	if diff < 0 {
		diff = -diff
	}
	return diff * float64(qty)
}

func slippageCost(price float64, qty int, pct float64) float64 {
	return price * float64(qty) * (pct / 100)
}

func excursion(dir Direction, entry, high, low float64, qty int) (favorable, adverse float64) {
	q := float64(qty)
	if dir == DirectionLong {
		return (high - entry) * q, (entry - low) * q
	}
	return (entry - low) * q, (high - entry) * q
}
