package paper

// Executor simulates order fills with configurable slippage.
type Executor struct {
	cfg Config
}

// NewExecutor creates a paper order executor from configuration.
func NewExecutor(cfg Config) *Executor {
	return &Executor{cfg: cfg.WithDefaults()}
}

// Execute simulates an approved trade intent and returns an execution report.
func (e *Executor) Execute(intent InputIntent, cache *Cache) ExecutionReport {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	orderID := cache.nextOrderID()
	base := ExecutionReport{
		OrderID:   orderID,
		Symbol:    intent.Symbol,
		Timeframe: intent.Timeframe,
		Action:    intent.Action,
		Quantity:  intent.Quantity,
		Strategy:  intent.Strategy,
		Timestamp: intent.Timestamp,
	}

	refPrice := intent.ReferencePrice
	if refPrice <= 0 {
		if fallback, ok := e.cfg.defaultPriceValue(); ok {
			refPrice = fallback
		} else if e.cfg.requiresMarketPrice() {
			base.Status = Rejected
			base.RejectionReason = "market reference price unavailable"
			cache.record(base)
			return base
		}
	}

	if refPrice <= 0 {
		base.Status = Rejected
		base.RejectionReason = "invalid execution price"
		cache.record(base)
		return base
	}

	base.ExecutionPrice = applySlippage(refPrice, intent.Action, e.cfg.SlippagePercent)
	base.Status = Filled
	cache.record(base)
	cache.apply(base)
	return base
}

func applySlippage(price float64, action string, slippagePercent float64) float64 {
	if slippagePercent == 0 {
		return price
	}
	factor := slippagePercent / 100
	switch action {
	case actionLongEntry, actionShortExit:
		return price * (1 + factor)
	case actionShortEntry, actionLongExit:
		return price * (1 - factor)
	default:
		return price
	}
}
