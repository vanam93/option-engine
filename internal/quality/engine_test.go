package quality_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/quality"
)

func testConfig() quality.Config {
	return quality.Config{
		Enabled:                true,
		SubscriberBuffer:       16,
		TrackingTimeoutMinutes: 120,
		ExcellentThreshold:     0.90,
		GoodThreshold:          0.75,
		AverageThreshold:       0.50,
		SuccessReturnPct:       0.005,
		FailureReturnPct:       -0.005,
	}
}

func intelligencePayload(recID string, overrides map[string]any) map[string]any {
	payload := map[string]any{
		"recommendation_id": recID,
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"strategy":          "ema_cross",
		"generated_at":      time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
		"document": map[string]any{
			"recommendation_level": "BUY",
			"confidence":           0.82,
			"current_status":       "ACTIVE",
		},
	}
	for key, value := range overrides {
		payload[key] = value
	}
	return payload
}

func statePayload(recID string, status string, at time.Time) map[string]any {
	return map[string]any{
		"recommendation_id": recID,
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"strategy":          "ema_cross",
		"current_status":    status,
		"confidence":        0.82,
		"latest_timeline_entry": map[string]any{
			"timestamp": at,
			"event":     "Status Changed",
		},
	}
}

func candle(symbol string, close float64, at time.Time) market.Candle {
	return market.Candle{
		ID:        uuid.New(),
		Symbol:    symbol,
		Timeframe: market.TF1m,
		Open:      close - 1,
		High:      close + 2,
		Low:       close - 2,
		Close:     close,
		OpenTime:  at.Add(-time.Minute),
		CloseTime: at,
	}
}

func publishIntelligence(t *testing.T, bus *eventbus.Bus, payload map[string]any, at time.Time) {
	t.Helper()
	evt, err := events.NewEventWithTime(events.RecommendationIntelligenceUpdated, "intelligence_engine", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func publishState(t *testing.T, bus *eventbus.Bus, payload map[string]any, at time.Time) {
	t.Helper()
	evt, err := events.NewEventWithTime(events.RecommendationStateUpdated, "recommendation_state_engine", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func publishCandle(t *testing.T, bus *eventbus.Bus, c market.Candle) {
	t.Helper()
	at := c.CloseTime
	if at.IsZero() {
		at = time.Now().UTC()
	}
	evt, err := events.NewEventWithTime(events.CandleClosed, "candle_engine", c, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func startEngine(t *testing.T, bus *eventbus.Bus, at time.Time) *quality.Engine {
	t.Helper()
	engine, err := quality.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	return engine
}

func subscribeQuality(t *testing.T, bus *eventbus.Bus) (*eventbus.Subscription, chan quality.RecommendationQualityUpdated) {
	t.Helper()
	updates := make(chan quality.RecommendationQualityUpdated, 8)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationQualityUpdated
	})
	go func() {
		for evt := range sub.C {
			var update quality.RecommendationQualityUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()
	return sub, updates
}

func TestRecommendationTracking(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeQuality(t, bus)
	engine := startEngine(t, bus, at)

	publishIntelligence(t, bus, intelligencePayload("REC-1", nil), at)
	publishCandle(t, bus, candle("NIFTY", 100, at.Add(time.Minute)))

	update := waitForEntryPrice(t, updates)
	require.Equal(t, "REC-1", update.RecommendationID)
	require.True(t, update.Report.TrackingActive)
	require.Equal(t, 100.0, update.Report.PriceStatistics.EntryPrice)

	report, ok := engine.GetLatest("REC-1")
	require.True(t, ok)
	require.Equal(t, "REC-1", report.RecommendationID)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestRecommendationClose(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeQuality(t, bus)
	engine := startEngine(t, bus, at)

	publishIntelligence(t, bus, intelligencePayload("REC-2", nil), at)
	publishCandle(t, bus, candle("NIFTY", 100, at.Add(time.Minute)))
	drainUpdates(updates)

	closeAt := at.Add(5 * time.Minute)
	publishCandle(t, bus, candle("NIFTY", 106, closeAt))
	publishState(t, bus, statePayload("REC-2", "CLOSED", closeAt), closeAt)

	update := waitForCompleted(t, updates)
	require.Equal(t, "REC-2", update.RecommendationID)
	require.True(t, update.Report.Completed)
	require.False(t, update.Report.TrackingActive)
	require.Equal(t, quality.StatusClosed, update.Report.CurrentStatus)

	report, ok := engine.GetLatest("REC-2")
	require.True(t, ok)
	require.True(t, report.Completed)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestTrackingTimeout(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	cfg := testConfig()
	cfg.TrackingTimeoutMinutes = 30
	engine, err := quality.New(cfg, bus, clock.NewReplay(at))
	require.NoError(t, err)
	sub, updates := subscribeQuality(t, bus)
	require.NoError(t, engine.Start(context.Background()))

	publishIntelligence(t, bus, intelligencePayload("REC-3", nil), at)
	publishCandle(t, bus, candle("NIFTY", 100, at.Add(time.Minute)))
	drainUpdates(updates)

	expireAt := at.Add(31 * time.Minute)
	publishCandle(t, bus, candle("NIFTY", 101, expireAt))

	update := waitForCompleted(t, updates)
	require.Equal(t, "REC-3", update.RecommendationID)
	require.True(t, update.Report.Completed)
	require.Equal(t, quality.OutcomeExpired, update.Report.Outcome)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestSuccessfulRecommendation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeQuality(t, bus)
	engine := startEngine(t, bus, at)

	publishIntelligence(t, bus, intelligencePayload("REC-4", nil), at)
	publishCandle(t, bus, candle("NIFTY", 100, at.Add(time.Minute)))
	drainUpdates(updates)

	closeAt := at.Add(10 * time.Minute)
	publishCandle(t, bus, candle("NIFTY", 110, closeAt))
	publishState(t, bus, statePayload("REC-4", "CLOSED", closeAt), closeAt)

	update := waitForCompleted(t, updates)
	require.Equal(t, quality.OutcomeSuccess, update.Report.Outcome)
	require.Greater(t, update.Report.QualityMetrics.ReturnPct, 0.05)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestFailedRecommendation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeQuality(t, bus)
	engine := startEngine(t, bus, at)

	publishIntelligence(t, bus, intelligencePayload("REC-5", nil), at)
	publishCandle(t, bus, candle("NIFTY", 100, at.Add(time.Minute)))
	drainUpdates(updates)

	closeAt := at.Add(10 * time.Minute)
	publishCandle(t, bus, candle("NIFTY", 90, closeAt))
	publishState(t, bus, statePayload("REC-5", "CLOSED", closeAt), closeAt)

	update := waitForCompleted(t, updates)
	require.Equal(t, quality.OutcomeFailed, update.Report.Outcome)
	require.Equal(t, quality.ClassificationFailed, update.Report.Classification)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestExpiredRecommendation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	cfg := testConfig()
	cfg.TrackingTimeoutMinutes = 15
	engine, err := quality.New(cfg, bus, clock.NewReplay(at))
	require.NoError(t, err)
	sub, updates := subscribeQuality(t, bus)
	require.NoError(t, engine.Start(context.Background()))

	publishIntelligence(t, bus, intelligencePayload("REC-6", nil), at)
	publishCandle(t, bus, candle("NIFTY", 100, at.Add(time.Minute)))
	drainUpdates(updates)

	expireAt := at.Add(16 * time.Minute)
	publishCandle(t, bus, candle("NIFTY", 100.2, expireAt))

	update := waitForCompleted(t, updates)
	require.Equal(t, quality.OutcomeExpired, update.Report.Outcome)

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestHealthMetrics(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)

	publishIntelligence(t, bus, intelligencePayload("REC-7", nil), at)
	publishCandle(t, bus, candle("NIFTY", 100, at.Add(time.Minute)))

	time.Sleep(50 * time.Millisecond)
	report := engine.Health()
	require.Equal(t, "recommendation_quality_engine", report.Component)
	require.Equal(t, "true", report.Details["enabled"])
	require.NotEmpty(t, report.Details["recommendations_tracked"])

	require.NoError(t, engine.Close())
}

func TestEventPublishing(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	sub, updates := subscribeQuality(t, bus)
	engine := startEngine(t, bus, at)

	publishIntelligence(t, bus, intelligencePayload("REC-8", nil), at)
	publishCandle(t, bus, candle("NIFTY", 100, at.Add(time.Minute)))

	select {
	case update := <-updates:
		require.Equal(t, "REC-8", update.RecommendationID)
		require.NotEmpty(t, update.Report.RecommendationID)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for recommendation.quality.updated")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestThreadSafety(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine := startEngine(t, bus, at)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			recID := "REC-TS-" + string(rune('A'+i))
			publishIntelligence(t, bus, intelligencePayload(recID, nil), at.Add(time.Duration(i)*time.Minute))
			publishCandle(t, bus, candle("NIFTY", 100+float64(i), at.Add(time.Duration(i+1)*time.Minute)))
		}(i)
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	report := engine.Health()
	require.NotEmpty(t, report.Details["active_trackers"])

	require.NoError(t, engine.Close())
}

func waitForEntryPrice(t *testing.T, updates chan quality.RecommendationQualityUpdated) quality.RecommendationQualityUpdated {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case update := <-updates:
			if update.Report.PriceStatistics.EntryPrice > 0 {
				return update
			}
		case <-deadline:
			t.Fatal("timeout waiting for entry price quality update")
		}
	}
}

func waitForCompleted(t *testing.T, updates chan quality.RecommendationQualityUpdated) quality.RecommendationQualityUpdated {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case update := <-updates:
			if update.Report.Completed {
				return update
			}
		case <-deadline:
			t.Fatal("timeout waiting for completed quality update")
		}
	}
}

func drainUpdates(ch chan quality.RecommendationQualityUpdated) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
