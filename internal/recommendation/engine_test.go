package recommendation_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/recommendation"
)

func testConfig() recommendation.Config {
	return recommendation.Config{
		Enabled:            true,
		SubscriberBuffer:   16,
		StrongBuyThreshold: 0.85,
		BuyThreshold:       0.70,
		WatchThreshold:     0.40,
	}
}

func richComponents() map[string]float64 {
	return map[string]float64{
		"signal":       0.90,
		"strategy":     0.88,
		"performance":  0.75,
		"optimization": 0.82,
		"walkforward":  0.80,
		"montecarlo":   0.78,
		"risk_factor":  1.0,
	}
}

func TestStrongBuyGeneration(t *testing.T) {
	builder := recommendation.NewBuilder(testConfig(), recommendation.NewFormatter())
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	rec := builder.Build(recommendation.InputOpportunity{
		Symbol: "NIFTY", Timeframe: "1m", Rank: 1, Confidence: 0.90,
		Components: richComponents(), Timestamp: at,
	}, at)

	require.Equal(t, recommendation.LevelStrongBuy, rec.Recommendation)
	require.NotEmpty(t, rec.Reasons)
	require.NotEmpty(t, rec.SupportingIndicators)
	require.NotEmpty(t, rec.SupportingStrategies)
	require.Contains(t, rec.OptimizationSummary, "Optimization score")
	require.Contains(t, rec.WalkForwardSummary, "Walk-forward")
	require.Contains(t, rec.MonteCarloSummary, "Monte Carlo")
}

func TestBuyGeneration(t *testing.T) {
	builder := recommendation.NewBuilder(testConfig(), recommendation.NewFormatter())
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	rec := builder.Build(recommendation.InputOpportunity{
		Symbol: "NIFTY", Timeframe: "1m", Rank: 2, Confidence: 0.75,
		Components: richComponents(), Timestamp: at,
	}, at)

	require.Equal(t, recommendation.LevelBuy, rec.Recommendation)
}

func TestWatchGeneration(t *testing.T) {
	builder := recommendation.NewBuilder(testConfig(), recommendation.NewFormatter())
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	rec := builder.Build(recommendation.InputOpportunity{
		Symbol: "NIFTY", Timeframe: "1m", Rank: 5, Confidence: 0.55,
		Components: map[string]float64{"signal": 0.5, "strategy": 0.5},
		Timestamp:  at,
	}, at)

	require.Equal(t, recommendation.LevelWatch, rec.Recommendation)
}

func TestRecommendationUpdatedPublished(t *testing.T) {
	bus := eventbus.New()
	engine, err := recommendation.New(testConfig(), bus, clock.NewSystem())
	require.NoError(t, err)

	updates := make(chan recommendation.RecommendationUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationUpdated
	})
	go func() {
		for evt := range sub.C {
			var update recommendation.RecommendationUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	oppEvt, err := events.NewEventWithTime(events.OpportunityUpdated, "opportunity", map[string]any{
		"symbol": "NIFTY", "timeframe": "1m", "rank": 1, "confidence": 0.92,
		"classification": "BUY", "score": 0.92, "components": richComponents(),
		"timestamp": at,
	}, at)
	require.NoError(t, err)
	bus.Publish(oppEvt)

	select {
	case update := <-updates:
		require.Equal(t, "NIFTY", update.Symbol)
		require.Equal(t, recommendation.LevelStrongBuy, update.Recommendation)
		require.Equal(t, 1, update.Rank)
		require.NotEmpty(t, update.Reasons)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for recommendation.updated")
	}

	latest, ok := engine.Latest("NIFTY", "1m")
	require.True(t, ok)
	require.Equal(t, recommendation.LevelStrongBuy, latest.Recommendation)

	report := engine.Health()
	require.Equal(t, "1", report.Details["recommendations_generated"])
	require.Equal(t, "1", report.Details["strong_buy"])

	require.NoError(t, engine.Close())
	sub.Close()
}
