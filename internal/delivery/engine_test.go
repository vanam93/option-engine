package delivery_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/delivery"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func testConfig() delivery.Config {
	return delivery.Config{
		Enabled:          true,
		SubscriberBuffer: 32,
	}
}

func statePayload(recID string, overrides map[string]any) map[string]any {
	payload := map[string]any{
		"recommendation_id": recID,
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"strategy":          "ema_cross",
		"recommendation":    "BUY",
		"current_status":    "ACTIVE",
		"confidence":        0.82,
		"components": map[string]float64{
			"optimization": 0.75,
			"walkforward":  0.71,
			"montecarlo":   0.68,
		},
		"scanner_matches":  []string{"ema_cross", "trend"},
		"opportunity_rank": 3,
		"latest_timeline_entry": map[string]any{
			"timestamp":      time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
			"event":          "Recommendation Created",
			"reason":         "validated buy recommendation",
			"previous_value": "",
			"new_value":      "ACTIVE",
		},
	}
	for key, value := range overrides {
		payload[key] = value
	}
	return payload
}

func intelligencePayload(recID string) map[string]any {
	return map[string]any{
		"recommendation_id": recID,
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"strategy":          "ema_cross",
		"generated_at":      time.Date(2026, 8, 2, 10, 1, 0, 0, time.UTC),
		"document": map[string]any{
			"recommendation_level":         "BUY",
			"confidence":                   0.82,
			"current_status":               "ACTIVE",
			"current_recommendation_state": "Active",
			"research_summary":             "Strong optimization and walk-forward support.",
			"decision_summary":             "NIFTY 1m BUY with 0.82 confidence.",
			"explanation":                  "Momentum and research alignment support BUY.",
			"research_evidence": map[string]any{
				"optimization": "Optimization score 75%",
				"walk_forward": "Walk-forward score 71%",
				"monte_carlo":  "Monte Carlo probability 68%",
			},
			"confidence_breakdown": map[string]any{
				"optimization_contribution": 0.75,
				"walk_forward_contribution": 0.71,
				"monte_carlo_contribution":  0.68,
			},
		},
	}
}

func qualityPayload(recID string, completed bool) map[string]any {
	return map[string]any{
		"recommendation_id": recID,
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"strategy":          "ema_cross",
		"generated_at":      time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC),
		"report": map[string]any{
			"recommendation_id":    recID,
			"recommendation_level": "BUY",
			"confidence":           0.82,
			"current_status":       "ACTIVE",
			"outcome":              "SUCCESS",
			"classification":       "GOOD",
			"quality_score":        0.78,
			"completed":            completed,
			"evaluated_at":         time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC),
			"price_statistics": map[string]any{
				"entry_price":       100.0,
				"latest_price":      101.5,
				"highest_price":     102.0,
				"lowest_price":      99.5,
				"absolute_return":   1.5,
				"percentage_return": 0.015,
				"holding_duration":  int64(30 * time.Minute),
			},
			"quality_metrics": map[string]any{
				"mfe":                 0.02,
				"mae":                 0.01,
				"return_pct":          0.015,
				"holding_duration_ms": int64(30 * time.Minute / time.Millisecond),
			},
		},
	}
}

func feedbackPayload() map[string]any {
	return map[string]any{
		"overall": map[string]any{
			"success_rate":        0.65,
			"average_return":      0.012,
			"average_quality":     0.74,
			"confidence_accuracy": 0.71,
		},
		"strategies": []map[string]any{
			{
				"strategy":       "ema_cross",
				"success_rate":   0.70,
				"win_rate":       0.68,
				"average_return": 0.014,
			},
		},
		"symbols": []map[string]any{
			{
				"symbol":         "NIFTY",
				"average_return": 0.011,
			},
		},
		"timeframes": []map[string]any{
			{
				"timeframe":      "1m",
				"win_rate":       0.66,
				"average_return": 0.010,
			},
		},
		"confidence_calibration": []map[string]any{
			{
				"label":        "0.80-0.90",
				"lower_bound":  0.80,
				"upper_bound":  0.90,
				"success_rate": 0.72,
			},
		},
		"timestamp": time.Date(2026, 8, 2, 11, 0, 0, 0, time.UTC),
	}
}

func alertPayload(recID string) map[string]any {
	return map[string]any{
		"alert_id":          "ALERT-1",
		"recommendation_id": recID,
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"alert_type":        "CONFIDENCE_INCREASED",
		"current_status":    "ACTIVE",
		"confidence":        0.85,
		"message":           "Confidence increased for NIFTY",
		"reason":            "validated recommendation confidence increased",
		"generated_at":      time.Date(2026, 8, 2, 10, 5, 0, 0, time.UTC),
	}
}

func publish(t *testing.T, bus *eventbus.Bus, eventType events.Type, payload map[string]any, at time.Time) {
	t.Helper()
	evt, err := events.NewEventWithTime(eventType, "test", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func startEngine(t *testing.T, bus *eventbus.Bus, at time.Time) *delivery.Engine {
	t.Helper()
	engine, err := delivery.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	return engine
}

func subscribeDelivery(t *testing.T, bus *eventbus.Bus) (*eventbus.Subscription, chan delivery.RecommendationDeliveryUpdated) {
	t.Helper()
	updates := make(chan delivery.RecommendationDeliveryUpdated, 64)
	sub := bus.Subscribe(64, func(evt events.Event) bool {
		return evt.Type == events.RecommendationDeliveryUpdated
	})
	go func() {
		for evt := range sub.C {
			var update delivery.RecommendationDeliveryUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()
	return sub, updates
}

func drainDelivery(t *testing.T, ch <-chan delivery.RecommendationDeliveryUpdated, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for delivery update %d", i+1)
		}
	}
}

func TestDocumentCreation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	sub, updates := subscribeDelivery(t, bus)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-1", nil), at)
	drainDelivery(t, updates, 1)

	doc, ok := engine.GetRecommendation("REC-1")
	require.True(t, ok)
	require.Equal(t, "REC-1", doc.RecommendationID)
	require.Equal(t, "NIFTY", doc.Symbol)
	require.Equal(t, delivery.LevelBuy, doc.CurrentRecommendationLevel)
	require.Equal(t, 3, doc.OpportunityRank)
	require.Len(t, doc.ScannerMatches, 2)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestIncrementalUpdateMerge(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	sub, updates := subscribeDelivery(t, bus)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-2", nil), at)
	drainDelivery(t, updates, 1)
	publish(t, bus, events.RecommendationIntelligenceUpdated, intelligencePayload("REC-2"), at)
	drainDelivery(t, updates, 1)

	doc, ok := engine.GetRecommendation("REC-2")
	require.True(t, ok)
	require.NotEmpty(t, doc.ResearchSummary)
	require.NotEmpty(t, doc.IntelligenceSummary)
	require.NotEmpty(t, doc.Evidence.Optimization)
	require.InDelta(t, 0.75, doc.OptimizationScore, 0.0001)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestTimelineOrdering(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	sub, updates := subscribeDelivery(t, bus)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-3", map[string]any{
		"latest_timeline_entry": map[string]any{
			"timestamp": time.Date(2026, 8, 2, 10, 10, 0, 0, time.UTC),
			"event":     "Status Changed",
			"reason":    "later status change",
			"new_value": "WATCH",
		},
	}), at)
	drainDelivery(t, updates, 1)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-3", map[string]any{
		"latest_timeline_entry": map[string]any{
			"timestamp": time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
			"event":     "Recommendation Created",
			"reason":    "created first",
			"new_value": "ACTIVE",
		},
	}), at)
	drainDelivery(t, updates, 1)

	doc, ok := engine.GetRecommendation("REC-3")
	require.True(t, ok)
	require.GreaterOrEqual(t, len(doc.Timeline), 2)
	require.True(t, !doc.Timeline[0].Timestamp.After(doc.Timeline[len(doc.Timeline)-1].Timestamp))

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestAlertAppend(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	sub, updates := subscribeDelivery(t, bus)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-4", nil), at)
	drainDelivery(t, updates, 1)
	publish(t, bus, events.AlertGenerated, alertPayload("REC-4"), at)
	drainDelivery(t, updates, 1)

	doc, ok := engine.GetRecommendation("REC-4")
	require.True(t, ok)
	require.Len(t, doc.AlertHistory, 1)
	require.Equal(t, "ALERT-1", doc.AlertHistory[0].AlertID)

	foundAlertTimeline := false
	for _, entry := range doc.Timeline {
		if entry.Event == delivery.TimelineAlertGenerated {
			foundAlertTimeline = true
			break
		}
	}
	require.True(t, foundAlertTimeline)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestQualityAppend(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	sub, updates := subscribeDelivery(t, bus)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-5", nil), at)
	drainDelivery(t, updates, 1)
	publish(t, bus, events.RecommendationQualityUpdated, qualityPayload("REC-5", false), at)
	drainDelivery(t, updates, 1)

	doc, ok := engine.GetRecommendation("REC-5")
	require.True(t, ok)
	require.NotNil(t, doc.QualityEvaluation)
	require.InDelta(t, 100.0, doc.EntryPrice, 0.0001)
	require.InDelta(t, 0.02, doc.MaximumFavorableExcursion, 0.0001)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestFeedbackAppend(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	sub, updates := subscribeDelivery(t, bus)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-6", nil), at)
	drainDelivery(t, updates, 1)
	publish(t, bus, events.RecommendationFeedbackUpdated, feedbackPayload(), at)
	drainDelivery(t, updates, 1)

	doc, ok := engine.GetRecommendation("REC-6")
	require.True(t, ok)
	require.NotNil(t, doc.FeedbackMetrics)
	require.NotNil(t, doc.HistoricalPerformance)
	require.InDelta(t, 0.70, doc.FeedbackMetrics.StrategySuccessRate, 0.0001)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestLookupByID(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-7", nil), at)
	time.Sleep(50 * time.Millisecond)

	doc, ok := engine.GetRecommendation("REC-7")
	require.True(t, ok)
	require.Equal(t, "REC-7", doc.RecommendationID)

	_, ok = engine.GetRecommendation("missing")
	require.False(t, ok)

	require.NoError(t, engine.Close())
}

func TestLookupBySymbolAndStrategy(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-8", nil), at)
	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-9", map[string]any{
		"symbol":   "BANKNIFTY",
		"strategy": "macd_cross",
	}), at)
	time.Sleep(50 * time.Millisecond)

	repo := engine.Repository()
	bySymbol := repo.ListBySymbol("NIFTY")
	require.Len(t, bySymbol, 1)
	require.Equal(t, "REC-8", bySymbol[0].RecommendationID)

	byStrategy := repo.ListByStrategy("macd_cross")
	require.Len(t, byStrategy, 1)
	require.Equal(t, "REC-9", byStrategy[0].RecommendationID)

	require.NoError(t, engine.Close())
}

func TestListActiveAndClosed(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	repo := engine.Repository()

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-10", nil), at)
	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-11", map[string]any{
		"current_status": "CLOSED",
		"latest_timeline_entry": map[string]any{
			"timestamp": at,
			"event":     "Closed",
			"reason":    "completed",
			"new_value": "CLOSED",
		},
	}), at)
	time.Sleep(50 * time.Millisecond)

	active := repo.ListActive()
	require.Len(t, active, 1)
	require.Equal(t, "REC-10", active[0].RecommendationID)

	closed := repo.ListClosed()
	require.Len(t, closed, 1)
	require.Equal(t, "REC-11", closed[0].RecommendationID)

	require.NoError(t, engine.Close())
}

func TestConcurrency(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			recID := fmt.Sprintf("REC-CONC-%02d", i)
			publish(t, bus, events.RecommendationStateUpdated, statePayload(recID, nil), at)
		}(i)
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	repo := engine.Repository()
	docs := repo.LatestRecommendations(50)
	require.GreaterOrEqual(t, len(docs), 20)

	require.NoError(t, engine.Close())
}

func TestGracefulShutdown(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)
	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-SHUTDOWN", nil), at)
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, engine.Close())

	report := engine.Health()
	require.Equal(t, "recommendation_delivery_engine", report.Component)
}

func TestHealthMetrics(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)

	publish(t, bus, events.RecommendationStateUpdated, statePayload("REC-HEALTH", nil), at)
	time.Sleep(50 * time.Millisecond)

	_, ok := engine.GetRecommendation("REC-HEALTH")
	require.True(t, ok)
	_, ok = engine.GetRecommendation("missing-health")
	require.False(t, ok)

	report := engine.Health()
	require.Equal(t, "recommendation_delivery_engine", report.Component)
	require.NotEmpty(t, report.Details["documents"])
	require.NotEmpty(t, report.Details["events_processed"])
	require.NotEmpty(t, report.Details["cache_hits"])
	require.NotEmpty(t, report.Details["cache_misses"])

	require.NoError(t, engine.Close())
}
