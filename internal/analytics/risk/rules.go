package risk

import "time"

// InputDecision mirrors the StrategyDecision payload consumed by the risk engine.
type InputDecision struct {
	Symbol     string
	Timeframe  string
	Decision   string
	Strategy   string
	Confidence float64
	Timestamp  time.Time
	Reason     string
}

// Evaluator applies risk controls to strategy decisions.
type Evaluator struct {
	cfg Config
}

// NewEvaluator creates a risk evaluator from configuration.
func NewEvaluator(cfg Config) *Evaluator {
	return &Evaluator{cfg: cfg.WithDefaults()}
}

// Process evaluates a decision, updates cache state, and returns the trade intent.
func (e *Evaluator) Process(input InputDecision, cache *Cache) ApprovedTradeIntent {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.resetDayIfNeeded(input.Timestamp)
	intent := e.evaluate(input, cache)
	cache.apply(intent)
	return intent
}

func (e *Evaluator) evaluate(input InputDecision, cache *Cache) ApprovedTradeIntent {
	base := newTradeIntent(
		Rejected,
		input.Decision,
		input.Symbol,
		input.Timeframe,
		input.Strategy,
		0,
		input.Confidence,
		input.Timestamp,
		"",
	)

	if input.Decision == actionHold {
		base.Reason = "non-actionable decision"
		return base
	}

	if input.Confidence < e.cfg.MinConfidence {
		base.Reason = "confidence below threshold"
		return base
	}

	if cache.tradesToday >= e.cfg.MaxTradesPerDay {
		base.Reason = "daily trade limit exceeded"
		return base
	}

	key := seriesKey{symbol: input.Symbol, timeframe: input.Timeframe}
	state := cache.seriesState(key)

	switch input.Decision {
	case actionLongEntry:
		if state.position == positionLong {
			base.Reason = "duplicate long position"
			return base
		}
		if state.position == positionFlat && cache.activePositions() >= e.cfg.MaxPositions {
			base.Reason = "max positions exceeded"
			return base
		}
	case actionShortEntry:
		if state.position == positionShort {
			base.Reason = "duplicate short position"
			return base
		}
		if state.position == positionFlat && cache.activePositions() >= e.cfg.MaxPositions {
			base.Reason = "max positions exceeded"
			return base
		}
	case actionLongExit:
		if state.position != positionLong {
			base.Reason = "no open long position"
			return base
		}
	case actionShortExit:
		if state.position != positionShort {
			base.Reason = "no open short position"
			return base
		}
	default:
		base.Reason = "unsupported decision"
		return base
	}

	return newTradeIntent(
		Approved,
		input.Decision,
		input.Symbol,
		input.Timeframe,
		input.Strategy,
		e.cfg.DefaultQuantity,
		input.Confidence,
		input.Timestamp,
		input.Reason,
	)
}
