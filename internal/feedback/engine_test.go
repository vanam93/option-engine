package feedback_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/feedback"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func testConfig() feedback.Config {
	return feedback.Config{
		Enabled:           true,
		SubscriberBuffer:  16,
		RollingWindows:    []int{25, 50, 100, 250},
		ConfidenceBuckets: []float64{0.60, 0.70, 0.80, 0.90, 0.95},
	}
}

func qualityPayload(recID string, outcome string, completed bool, overrides map[string]any) map[string]any {
	payload := map[string]any{
		"recommendation_id": recID,
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"strategy":          "ema_cross",
		"generated_at":      time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
		"report": map[string]any{
			"recommendation_id":    recID,
			"symbol":               "NIFTY",
			"timeframe":            "1m",
			"strategy":             "ema_cross",
			"recommendation_level": "BUY",
			"confidence":           0.82,
			"outcome":              outcome,
			"quality_score":        0.78,
			"completed":            completed,
			"evaluated_at":         time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC),
			"quality_metrics": map[string]any{
				"mfe":                 0.02,
				"mae":                 0.01,
				"maximum_drawdown":    0.01,
				"return_pct":          0.008,
				"holding_duration_ms": int64(30 * time.Minute / time.Millisecond),
			},
		},
	}
	for key, value := range overrides {
		payload[key] = value
	}
	return payload
}

func publishQuality(t *testing.T, bus *eventbus.Bus, payload map[string]any, at time.Time) {
	t.Helper()
	evt, err := events.NewEventWithTime(events.RecommendationQualityUpdated, "quality_engine", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func startEngine(t *testing.T, bus *eventbus.Bus, at time.Time) *feedback.Engine {
	t.Helper()
	engine, err := feedback.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	return engine
}

func subscribeFeedback(t *testing.T, bus *eventbus.Bus) (*eventbus.Subscription, chan feedback.RecommendationFeedbackUpdated) {
	t.Helper()
	updates := make(chan feedback.RecommendationFeedbackUpdated, 32)
	sub := bus.Subscribe(32, func(evt events.Event) bool {
		return evt.Type == events.RecommendationFeedbackUpdated
	})
	go func() {
		for evt := range sub.C {
			var update feedback.RecommendationFeedbackUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()
	return sub, updates
}

func waitFeedback(t *testing.T, ch <-chan feedback.RecommendationFeedbackUpdated) feedback.RecommendationFeedbackUpdated {
	t.Helper()
	select {
	case update := <-ch:
		return update
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for recommendation.feedback.updated")
		return feedback.RecommendationFeedbackUpdated{}
	}
}

func TestOverallAggregation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, nil), at)
	update := waitFeedback(t, updates)

	require.Equal(t, 1, update.Overall.TotalRecommendations)
	require.Equal(t, 1, update.Overall.Successful)
	require.Equal(t, 1.0, update.Overall.SuccessRate)
	require.Equal(t, 1.0, update.Overall.WinRate)
	require.InDelta(t, 0.008, update.Overall.AverageReturn, 1e-6)

	publishQuality(t, bus, qualityPayload("REC-2", "FAILED", true, map[string]any{
		"strategy": "rsi_reversal",
		"report": map[string]any{
			"recommendation_id":    "REC-2",
			"symbol":               "BANKNIFTY",
			"timeframe":            "5m",
			"strategy":             "rsi_reversal",
			"recommendation_level": "BUY",
			"confidence":           0.75,
			"outcome":              "FAILED",
			"quality_score":        0.20,
			"completed":            true,
			"evaluated_at":         at,
			"quality_metrics": map[string]any{
				"return_pct":          -0.01,
				"holding_duration_ms": int64(20 * time.Minute / time.Millisecond),
			},
		},
	}), at)

	update = waitFeedback(t, updates)
	require.Equal(t, 2, update.Overall.TotalRecommendations)
	require.Equal(t, 1, update.Overall.Successful)
	require.Equal(t, 1, update.Overall.Failed)
	require.Equal(t, 0.5, update.Overall.SuccessRate)
	require.Equal(t, 0.5, update.Overall.WinRate)
	require.Equal(t, 1, update.Overall.FalsePositives)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestStrategyAggregation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, nil), at)
	waitFeedback(t, updates)

	publishQuality(t, bus, qualityPayload("REC-2", "SUCCESS", true, map[string]any{
		"strategy": "rsi_reversal",
		"report": map[string]any{
			"recommendation_id": "REC-2",
			"strategy":          "rsi_reversal",
			"outcome":           "SUCCESS",
			"completed":         true,
			"quality_score":     0.80,
			"quality_metrics": map[string]any{
				"return_pct": 0.01,
			},
		},
	}), at)

	update := waitFeedback(t, updates)
	require.Len(t, update.Strategies, 2)
	require.Equal(t, 1, update.Strategies[0].Recommendations)
	require.Equal(t, 1, update.Strategies[1].Recommendations)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestSymbolAggregation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, nil), at)
	waitFeedback(t, updates)

	publishQuality(t, bus, qualityPayload("REC-2", "FAILED", true, map[string]any{
		"symbol": "BANKNIFTY",
		"report": map[string]any{
			"recommendation_id": "REC-2",
			"symbol":            "BANKNIFTY",
			"outcome":           "FAILED",
			"completed":         true,
			"quality_metrics": map[string]any{
				"return_pct": -0.005,
			},
		},
	}), at)

	update := waitFeedback(t, updates)
	require.Len(t, update.Symbols, 2)
	require.Equal(t, "BANKNIFTY", update.Symbols[0].Symbol)
	require.Equal(t, "NIFTY", update.Symbols[1].Symbol)
	require.Equal(t, 1, update.Symbols[1].Wins)
	require.Equal(t, 1, update.Symbols[0].Losses)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestTimeframeAggregation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, nil), at)
	waitFeedback(t, updates)

	publishQuality(t, bus, qualityPayload("REC-2", "FAILED", true, map[string]any{
		"timeframe": "5m",
		"report": map[string]any{
			"recommendation_id": "REC-2",
			"timeframe":         "5m",
			"outcome":           "FAILED",
			"completed":         true,
			"quality_metrics": map[string]any{
				"return_pct": -0.01,
			},
		},
	}), at)

	update := waitFeedback(t, updates)
	require.Len(t, update.Timeframes, 2)
	require.Equal(t, 1.0, update.Timeframes[0].WinRate)
	require.Equal(t, 0.0, update.Timeframes[1].WinRate)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestConfidenceBucketAggregation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, map[string]any{
		"report": map[string]any{
			"recommendation_id": "REC-1",
			"confidence":        0.82,
			"outcome":           "SUCCESS",
			"completed":         true,
			"quality_metrics": map[string]any{
				"return_pct": 0.01,
			},
		},
	}), at)

	update := waitFeedback(t, updates)
	require.NotEmpty(t, update.ConfidenceCalibration)
	found := false
	for _, bucket := range update.ConfidenceCalibration {
		if bucket.LowerBound == 0.80 && bucket.UpperBound == 0.90 {
			found = true
			require.Equal(t, 1, bucket.Recommendations)
			require.Equal(t, 1, bucket.Successes)
		}
	}
	require.True(t, found)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestRollingWindowUpdates(t *testing.T) {
	cfg := testConfig()
	cfg.RollingWindows = []int{2, 3}
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine, err := feedback.New(cfg, bus, clock.NewReplay(at))
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))

	for i := 0; i < 3; i++ {
		recID := fmt.Sprintf("REC-%d", i)
		publishQuality(t, bus, qualityPayload(recID, "SUCCESS", true, map[string]any{
			"recommendation_id": recID,
			"report": map[string]any{
				"recommendation_id": recID,
				"outcome":           "SUCCESS",
				"completed":         true,
				"quality_metrics": map[string]any{
					"return_pct": 0.01,
				},
			},
		}), at)
		waitFeedback(t, updates)
	}

	snapshot, ok := engine.GetSnapshot()
	require.True(t, ok)
	require.Len(t, snapshot.Rolling, 2)
	require.Equal(t, 2, snapshot.Rolling[0].WindowSize)
	require.Equal(t, 1.0, snapshot.Rolling[0].SuccessRate)
	require.Equal(t, 3, snapshot.Rolling[1].WindowSize)
	require.Equal(t, 1.0, snapshot.Rolling[1].SuccessRate)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestDuplicateRecommendationHandling(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, nil), at)
	update := waitFeedback(t, updates)
	require.Equal(t, 1, update.Overall.TotalRecommendations)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, nil), at)
	select {
	case update := <-updates:
		t.Fatalf("unexpected duplicate feedback event: total=%d", update.Overall.TotalRecommendations)
	case <-time.After(200 * time.Millisecond):
	}

	snapshot, ok := engine.GetSnapshot()
	require.True(t, ok)
	require.Equal(t, 1, snapshot.Overall.TotalRecommendations)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestFeedbackEventPublishing(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, nil), at)
	update := waitFeedback(t, updates)

	require.Equal(t, "1", update.Version)
	require.NotZero(t, update.Timestamp)
	require.NotEmpty(t, update.Rolling)
	require.NotEmpty(t, update.ConfidenceCalibration)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestHealthMetrics(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", true, nil), at)
	time.Sleep(50 * time.Millisecond)

	report := engine.Health()
	require.Equal(t, "feedback_engine", report.Component)
	require.Equal(t, "1", report.Details["events_processed"])
	require.Equal(t, "1", report.Details["feedback_generated"])
	require.Equal(t, "1", report.Details["tracked_recommendations"])
	require.Equal(t, "1", report.Details["tracked_strategies"])
	require.Equal(t, "1", report.Details["tracked_symbols"])
	require.Equal(t, "1", report.Details["tracked_timeframes"])

	require.NoError(t, engine.Close())
}

func TestIgnoresIncompleteQualityEvents(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeFeedback(t, bus)
	engine := startEngine(t, bus, at)

	publishQuality(t, bus, qualityPayload("REC-1", "SUCCESS", false, nil), at)
	select {
	case <-updates:
		t.Fatal("unexpected feedback for incomplete quality event")
	case <-time.After(200 * time.Millisecond):
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestThreadSafety(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			recID := fmt.Sprintf("REC-%d", idx)
			publishQuality(t, bus, qualityPayload(recID, "SUCCESS", true, map[string]any{
				"recommendation_id": recID,
				"report": map[string]any{
					"recommendation_id": recID,
					"outcome":           "SUCCESS",
					"completed":         true,
					"quality_metrics": map[string]any{
						"return_pct": 0.01,
					},
				},
			}), at)
		}(i)
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	snapshot, ok := engine.GetSnapshot()
	require.True(t, ok)
	require.Equal(t, 20, snapshot.Overall.TotalRecommendations)

	require.NoError(t, engine.Close())
}

func TestGracefulShutdown(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)

	done := make(chan struct{})
	go func() {
		require.NoError(t, engine.Close())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("engine close timed out")
	}

	report := engine.Health()
	require.False(t, report.Connected)
}
