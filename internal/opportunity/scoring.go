package opportunity

// ScoreComponents holds normalized factor contributions.
type ScoreComponents struct {
	Signal       float64
	Strategy     float64
	Performance  float64
	Optimization float64
	WalkForward  float64
	MonteCarlo   float64
	RiskFactor   float64
}

// ScoreResult is the computed opportunity confidence for a symbol.
type ScoreResult struct {
	Symbol         string
	Timeframe      string
	Confidence     float64
	Score          float64
	Classification Classification
	Components     map[string]float64
}

// Scorer computes weighted opportunity confidence from intelligence state.
type Scorer struct {
	cfg Config
}

// NewScorer creates a scorer from configuration.
func NewScorer(cfg Config) *Scorer {
	return &Scorer{cfg: cfg.withDefaults()}
}

// Compute derives confidence and classification for a symbol.
func (s *Scorer) Compute(state SymbolState, platform PlatformState) ScoreResult {
	weights := s.cfg.Weights
	weightSum := weights.Signal + weights.Strategy + weights.Performance +
		weights.Optimization + weights.WalkForward + weights.MonteCarlo

	signal := state.SignalConfidence
	if signal == 0 && state.ScannerConfidence > 0 {
		signal = state.ScannerConfidence
	}

	components := ScoreComponents{
		Signal:       signal,
		Strategy:     state.StrategyConfidence,
		Performance:  state.PerformanceScore,
		Optimization: state.OptimizationScore,
		WalkForward:  platform.WalkForwardScore,
		MonteCarlo:   platform.MonteCarloScore,
		RiskFactor:   1,
	}
	if !state.RiskApproved {
		components.RiskFactor = 0.6
	}

	score := (weights.Signal*components.Signal +
		weights.Strategy*components.Strategy +
		weights.Performance*components.Performance +
		weights.Optimization*components.Optimization +
		weights.WalkForward*components.WalkForward +
		weights.MonteCarlo*components.MonteCarlo) / weightSum

	score *= components.RiskFactor
	confidence := clamp01(score)

	return ScoreResult{
		Symbol:         state.Symbol,
		Timeframe:      state.Timeframe,
		Confidence:     confidence,
		Score:          confidence,
		Classification: classify(confidence, s.cfg.BuyThreshold, s.cfg.WatchThreshold),
		Components: map[string]float64{
			"signal":       components.Signal,
			"strategy":     components.Strategy,
			"performance":  components.Performance,
			"optimization": components.Optimization,
			"walkforward":  components.WalkForward,
			"montecarlo":   components.MonteCarlo,
			"risk_factor":  components.RiskFactor,
		},
	}
}

func classify(confidence, buyThreshold, watchThreshold float64) Classification {
	switch {
	case confidence >= buyThreshold:
		return ClassificationBuy
	case confidence >= watchThreshold:
		return ClassificationWatch
	default:
		return ClassificationIgnore
	}
}
