package opportunity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
)

func testConfig() opportunity.Config {
	return opportunity.Config{
		Enabled:        true,
		TopN:           10,
		BuyThreshold:   0.70,
		WatchThreshold: 0.40,
		Weights: opportunity.WeightsConfig{
			Signal:       0.20,
			Strategy:     0.20,
			Performance:  0.15,
			Optimization: 0.15,
			WalkForward:  0.15,
			MonteCarlo:   0.15,
		},
	}
}

func TestConfidenceCalculation(t *testing.T) {
	scorer := opportunity.NewScorer(testConfig())
	state := opportunity.SymbolState{
		Symbol:             "NIFTY",
		Timeframe:          "1m",
		SignalConfidence:   0.8,
		StrategyConfidence: 0.7,
		RiskApproved:       true,
		PerformanceScore:   0.6,
		OptimizationScore:  0.5,
	}
	platform := opportunity.PlatformState{
		WalkForwardScore: 0.9,
		MonteCarloScore:  0.85,
	}

	result := scorer.Compute(state, platform)
	require.InDelta(t, 0.7275, result.Confidence, 0.01)
	require.NotEmpty(t, result.Components)
}

func TestRankingOrder(t *testing.T) {
	scorer := opportunity.NewScorer(testConfig())
	ranker := opportunity.NewRanker(testConfig())
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	platform := opportunity.PlatformState{WalkForwardScore: 0.5, MonteCarloScore: 0.5}

	states := []opportunity.SymbolState{
		{Symbol: "NIFTY", Timeframe: "1m", SignalConfidence: 0.9, StrategyConfidence: 0.9, RiskApproved: true, PerformanceScore: 0.8, OptimizationScore: 0.8},
		{Symbol: "BANKNIFTY", Timeframe: "1m", SignalConfidence: 0.5, StrategyConfidence: 0.5, RiskApproved: true, PerformanceScore: 0.4, OptimizationScore: 0.4},
	}

	ranked := ranker.Rank(states, platform, scorer)
	require.Len(t, ranked, 2)
	require.Equal(t, "NIFTY", ranked[0].Symbol)
	require.Equal(t, 1, ranked[0].Rank)
	require.Equal(t, "BANKNIFTY", ranked[1].Symbol)
	require.Equal(t, 2, ranked[1].Rank)
}

func TestBuyClassification(t *testing.T) {
	scorer := opportunity.NewScorer(testConfig())
	state := opportunity.SymbolState{
		Symbol:             "NIFTY",
		Timeframe:          "1m",
		SignalConfidence:   0.95,
		StrategyConfidence: 0.90,
		RiskApproved:       true,
		PerformanceScore:   0.85,
		OptimizationScore:  0.80,
	}
	platform := opportunity.PlatformState{
		WalkForwardScore: 0.90,
		MonteCarloScore:  0.88,
	}

	result := scorer.Compute(state, platform)
	require.Equal(t, opportunity.ClassificationBuy, result.Classification)
	require.GreaterOrEqual(t, result.Confidence, 0.70)
}

func TestWatchClassification(t *testing.T) {
	scorer := opportunity.NewScorer(testConfig())
	state := opportunity.SymbolState{
		Symbol:             "NIFTY",
		Timeframe:          "1m",
		SignalConfidence:   0.65,
		StrategyConfidence: 0.60,
		RiskApproved:       true,
		PerformanceScore:   0.55,
		OptimizationScore:  0.50,
	}
	platform := opportunity.PlatformState{
		WalkForwardScore: 0.60,
		MonteCarloScore:  0.55,
	}

	result := scorer.Compute(state, platform)
	require.Equal(t, opportunity.ClassificationWatch, result.Classification)
	require.GreaterOrEqual(t, result.Confidence, 0.40)
	require.Less(t, result.Confidence, 0.70)
}
