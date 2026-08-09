package montecarlo

import (
	"math"
	"math/rand"
)

// Simulator runs bootstrap and shuffle Monte Carlo paths.
type Simulator struct {
	cfg Config
	rng *rand.Rand
}

// NewSimulator creates a Monte Carlo path generator.
func NewSimulator(cfg Config) *Simulator {
	cfg = cfg.WithDefaults()
	var rng *rand.Rand
	if cfg.RandomSeed != nil {
		rng = rand.New(rand.NewSource(*cfg.RandomSeed))
	} else {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	return &Simulator{cfg: cfg, rng: rng}
}

// Run executes Monte Carlo simulations on trade returns.
func (s *Simulator) Run(trades []float64) ([]SimulationOutcome, error) {
	if len(trades) == 0 {
		return nil, ErrNoTrades
	}

	outcomes := make([]SimulationOutcome, 0, s.cfg.Simulations)
	sampleSize := len(trades)

	for i := 0; i < s.cfg.Simulations; i++ {
		var path []float64
		if i%2 == 0 {
			path = BootstrapSample(trades, sampleSize, s.rng)
		} else {
			path = ShuffleOrder(trades, s.rng)
		}
		outcomes = append(outcomes, SimulationOutcome{
			TotalReturn: TotalReturn(path),
			MaxDrawdown: MaxDrawdown(path),
		})
	}
	return outcomes, nil
}

// Summarize converts simulation outcomes into a completed result.
func (s *Simulator) Summarize(
	simulationID, walkForwardID, experimentID string,
	outcomes []SimulationOutcome,
	startingCapital float64,
) SimulationResult {
	returns := make([]float64, len(outcomes))
	drawdowns := make([]float64, len(outcomes))
	for i, o := range outcomes {
		returns[i] = o.TotalReturn
		drawdowns[i] = o.MaxDrawdown
	}

	if startingCapital <= 0 {
		startingCapital = math.Max(1, mean(returns))
	}

	return SimulationResult{
		SimulationID:        simulationID,
		WalkForwardID:       walkForwardID,
		ExperimentID:        experimentID,
		Simulations:         len(outcomes),
		ConfidenceInterval:  ComputeConfidenceInterval(returns, s.cfg.ConfidenceLevel),
		ProbabilityOfProfit: ProbabilityOfProfit(returns),
		ProbabilityOfLoss:   ProbabilityOfLoss(returns),
		RiskOfRuin:          RiskOfRuin(drawdowns, startingCapital, s.cfg.RuinDrawdownPct),
		DistributionSummary: ComputeDistributionSummary(returns, drawdowns),
	}
}

// StartingCapital estimates initial equity from trade magnitudes.
func StartingCapital(trades []float64) float64 {
	if len(trades) == 0 {
		return 1
	}
	var sumAbs float64
	for _, t := range trades {
		sumAbs += math.Abs(t)
	}
	if sumAbs <= 0 {
		return 1
	}
	return sumAbs
}
