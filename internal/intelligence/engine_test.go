package intelligence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/intelligence"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func testConfig() intelligence.Config {
	return intelligence.Config{
		Enabled:                    true,
		SubscriberBuffer:           16,
		IncludeTimeline:            true,
		IncludeResearch:            true,
		IncludeConfidenceBreakdown: true,
		StrongBuyThreshold:         0.85,
		BuyThreshold:               0.70,
		WatchThreshold:             0.40,
	}
}

func stateUpdatePayload(recID string, overrides map[string]any) map[string]any {
	payload := map[string]any{
		"recommendation_id": recID,
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"strategy":          "ema_cross",
		"recommendation":    "BUY",
		"current_status":    "ACTIVE",
		"confidence":        0.82,
		"latest_timeline_entry": map[string]any{
			"timestamp":      time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
			"event":          "Status Changed",
			"reason":         "initial activation from validated recommendation",
			"previous_value": "CREATED",
			"new_value":      "ACTIVE",
		},
		"summary": "NIFTY 1m ema_cross ACTIVE: Status Changed (confidence 0.82)",
		"components": map[string]float64{
			"signal":       0.88,
			"strategy":     0.80,
			"performance":  0.62,
			"optimization": 0.75,
			"walkforward":  0.71,
			"montecarlo":   0.68,
		},
		"supporting_indicators": []string{"Strong EMA crossover", "MACD bullish confirmation"},
		"supporting_strategies": []string{"Trend Following confirmed"},
		"optimization_summary":  "Optimization score 75% (strong parameter fit).",
		"walk_forward_summary":  "Walk-forward validation score 71% (strong out-of-sample robustness).",
		"monte_carlo_summary":   "Monte Carlo profit probability 68% (strong simulated robustness).",
	}
	for key, value := range overrides {
		payload[key] = value
	}
	return payload
}

func publishStateUpdate(t *testing.T, bus *eventbus.Bus, payload map[string]any, at time.Time) {
	t.Helper()
	evt, err := events.NewEventWithTime(events.RecommendationStateUpdated, "recommendation_state_engine", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func TestRecommendationExplanation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := intelligence.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan intelligence.RecommendationIntelligenceUpdated, 2)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationIntelligenceUpdated
	})
	go func() {
		for evt := range sub.C {
			var update intelligence.RecommendationIntelligenceUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))
	publishStateUpdate(t, bus, stateUpdatePayload("REC-1", nil), at)

	select {
	case update := <-updates:
		require.Equal(t, "REC-1", update.RecommendationID)
		require.NotEmpty(t, update.Document.Explanation)
		require.Equal(t, intelligence.LevelBuy, update.Document.RecommendationLevel)
		require.Contains(t, update.Document.DecisionSummary, "NIFTY")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for recommendation.intelligence.updated")
	}

	doc, ok := engine.Get("REC-1")
	require.True(t, ok)
	require.Equal(t, "REC-1", doc.RecommendationID)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestResearchSummaryGeneration(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := intelligence.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan intelligence.RecommendationIntelligenceUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationIntelligenceUpdated
	})
	go func() {
		for evt := range sub.C {
			var update intelligence.RecommendationIntelligenceUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))
	publishStateUpdate(t, bus, stateUpdatePayload("REC-2", nil), at)

	select {
	case update := <-updates:
		require.NotEmpty(t, update.Document.ResearchSummary)
		require.NotEmpty(t, update.Document.ResearchEvidence.Signal)
		require.NotEmpty(t, update.Document.ResearchEvidence.Optimization)
		require.NotEmpty(t, update.Document.ResearchEvidence.WalkForward)
		require.NotEmpty(t, update.Document.ResearchEvidence.MonteCarlo)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for intelligence update")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestConfidenceBreakdown(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := intelligence.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan intelligence.RecommendationIntelligenceUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationIntelligenceUpdated
	})
	go func() {
		for evt := range sub.C {
			var update intelligence.RecommendationIntelligenceUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))
	publishStateUpdate(t, bus, stateUpdatePayload("REC-3", nil), at)

	select {
	case update := <-updates:
		breakdown := update.Document.ConfidenceBreakdown
		require.InDelta(t, 0.82, breakdown.Overall, 0.0001)
		require.NotNil(t, breakdown.Signal)
		require.NotNil(t, breakdown.Strategy)
		require.NotNil(t, breakdown.Optimization)
		require.NotNil(t, breakdown.WalkForward)
		require.NotNil(t, breakdown.MonteCarlo)
		require.NotNil(t, breakdown.Validation)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for intelligence update")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestUpgradeExplanation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := intelligence.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan intelligence.RecommendationIntelligenceUpdated, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationIntelligenceUpdated
	})
	go func() {
		for evt := range sub.C {
			var update intelligence.RecommendationIntelligenceUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	watchPayload := stateUpdatePayload("REC-4", map[string]any{
		"recommendation": "WATCH",
		"current_status": "WATCH",
		"confidence":     0.55,
		"latest_timeline_entry": map[string]any{
			"timestamp":      at,
			"event":          "Status Changed",
			"reason":         "validated recommendation status changed",
			"previous_value": "CREATED",
			"new_value":      "WATCH",
		},
	})
	publishStateUpdate(t, bus, watchPayload, at)
	select {
	case <-updates:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout on watch state update")
	}

	buyPayload := stateUpdatePayload("REC-4", map[string]any{
		"recommendation": "BUY",
		"current_status": "ACTIVE",
		"confidence":     0.82,
		"latest_timeline_entry": map[string]any{
			"timestamp":      at.Add(time.Minute),
			"event":          "Status Changed",
			"reason":         "validated recommendation status changed",
			"previous_value": "WATCH",
			"new_value":      "ACTIVE",
		},
	})
	publishStateUpdate(t, bus, buyPayload, at.Add(time.Minute))

	select {
	case update := <-updates:
		require.Contains(t, update.Document.ReasonForUpgrade, "upgraded")
		require.Contains(t, update.Document.Explanation, "upgraded")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for upgrade explanation")
	}

	report := engine.Health()
	require.Equal(t, "1", report.Details["upgrade_explanations"])

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestDowngradeExplanation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := intelligence.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan intelligence.RecommendationIntelligenceUpdated, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationIntelligenceUpdated
	})
	go func() {
		for evt := range sub.C {
			var update intelligence.RecommendationIntelligenceUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	buyPayload := stateUpdatePayload("REC-5", map[string]any{
		"recommendation": "BUY",
		"current_status": "ACTIVE",
		"confidence":     0.82,
	})
	publishStateUpdate(t, bus, buyPayload, at)
	select {
	case <-updates:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout on initial buy update")
	}

	watchPayload := stateUpdatePayload("REC-5", map[string]any{
		"recommendation": "WATCH",
		"current_status": "WATCH",
		"confidence":     0.55,
		"latest_timeline_entry": map[string]any{
			"timestamp":      at.Add(time.Minute),
			"event":          "Confidence Decreased",
			"reason":         "performance deteriorated",
			"previous_value": "0.8200",
			"new_value":      "0.5500",
		},
	})
	publishStateUpdate(t, bus, watchPayload, at.Add(time.Minute))

	select {
	case update := <-updates:
		require.Contains(t, update.Document.ReasonForDowngrade, "downgraded")
		require.NotEmpty(t, update.Document.RiskFactors)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for downgrade explanation")
	}

	report := engine.Health()
	require.Equal(t, "1", report.Details["downgrade_explanations"])

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestTimelineSummary(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := intelligence.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan intelligence.RecommendationIntelligenceUpdated, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationIntelligenceUpdated
	})
	go func() {
		for evt := range sub.C {
			var update intelligence.RecommendationIntelligenceUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	first := stateUpdatePayload("REC-6", nil)
	publishStateUpdate(t, bus, first, at)
	select {
	case <-updates:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout on first update")
	}

	second := stateUpdatePayload("REC-6", map[string]any{
		"latest_timeline_entry": map[string]any{
			"timestamp":      at.Add(time.Minute),
			"event":          "Confidence Increased",
			"reason":         "validated recommendation confidence increased",
			"previous_value": "0.8200",
			"new_value":      "0.9000",
		},
		"confidence": 0.90,
	})
	publishStateUpdate(t, bus, second, at.Add(time.Minute))

	select {
	case update := <-updates:
		require.NotEmpty(t, update.Document.TimelineSummary)
		require.Contains(t, update.Document.TimelineSummary, "Confidence Increased")
		require.GreaterOrEqual(t, len(update.Document.RecommendationHistory), 2)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for timeline summary")
	}

	report := engine.Health()
	require.Equal(t, "2", report.Details["timeline_summaries"])

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestIntelligenceUpdatedPublished(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := intelligence.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	eventsCh := make(chan events.Event, 2)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationIntelligenceUpdated
	})
	go func() {
		for evt := range sub.C {
			eventsCh <- evt
		}
	}()

	require.NoError(t, engine.Start(context.Background()))
	publishStateUpdate(t, bus, stateUpdatePayload("REC-7", nil), at)

	select {
	case evt := <-eventsCh:
		require.Equal(t, events.RecommendationIntelligenceUpdated, evt.Type)
		require.Equal(t, "recommendation_intelligence_engine", evt.Source)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for recommendation.intelligence.updated event")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestHealthMetrics(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := intelligence.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationIntelligenceUpdated
	})

	require.NoError(t, engine.Start(context.Background()))
	publishStateUpdate(t, bus, stateUpdatePayload("REC-8", nil), at)
	time.Sleep(100 * time.Millisecond)

	report := engine.Health()
	require.Equal(t, "recommendation_intelligence_engine", report.Component)
	require.Equal(t, "1", report.Details["documents_generated"])
	require.NotEmpty(t, report.Details["average_confidence"])
	require.Equal(t, "1", report.Details["research_summaries"])

	require.NoError(t, engine.Close())
	sub.Close()
}
