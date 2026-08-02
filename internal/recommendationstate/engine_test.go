package recommendationstate_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/recommendationstate"
)

func testConfig() recommendationstate.Config {
	return recommendationstate.Config{
		Enabled:          true,
		SubscriberBuffer: 16,
		MaxActive:        1000,
	}
}

func validValidatedPayload(at time.Time, confidence float64) map[string]any {
	return map[string]any{
		"symbol":            "NIFTY",
		"timeframe":         "1m",
		"strategy":          "ema_cross",
		"recommendation":    "BUY",
		"confidence":        confidence,
		"validation_status": "VALID",
		"validated_at":      at,
	}
}

func publishValidated(t *testing.T, bus *eventbus.Bus, payload map[string]any, at time.Time) {
	t.Helper()
	evt, err := events.NewEventWithTime(events.ValidatedRecommendation, "validation", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func TestRecommendationCreation(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := recommendationstate.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan recommendationstate.RecommendationStateUpdated, 2)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationStateUpdated
	})
	go func() {
		for evt := range sub.C {
			var update recommendationstate.RecommendationStateUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))
	publishValidated(t, bus, validValidatedPayload(at, 0.82), at)

	select {
	case update := <-updates:
		require.Equal(t, "NIFTY", update.Symbol)
		require.Equal(t, "1m", update.Timeframe)
		require.Equal(t, "ema_cross", update.Strategy)
		require.Equal(t, recommendationstate.StatusActive, update.CurrentStatus)
		require.Contains(t, update.RecommendationID, "REC-20260802-NIFTY-")
		require.Equal(t, "Status Changed", update.LatestTimelineEntry.Event)
		require.NotEmpty(t, update.Summary)

		rec, timeline, ok := engine.Get(update.RecommendationID)
		require.True(t, ok)
		require.Equal(t, update.RecommendationID, rec.RecommendationID)
		require.Equal(t, "Recommendation Created", timeline[0].Event)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for recommendation.state.updated")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestRecommendationUpdateAndTimelineAppend(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	clk := clock.NewReplay(at)
	engine, err := recommendationstate.New(testConfig(), bus, clk)
	require.NoError(t, err)

	updates := make(chan recommendationstate.RecommendationStateUpdated, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationStateUpdated
	})
	go func() {
		for evt := range sub.C {
			var update recommendationstate.RecommendationStateUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))
	publishValidated(t, bus, validValidatedPayload(at, 0.75), at)

	var firstID string
	select {
	case update := <-updates:
		firstID = update.RecommendationID
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first recommendation.state.updated")
	}

	clk.Advance(time.Minute)
	publishValidated(t, bus, validValidatedPayload(at.Add(time.Minute), 0.88), at.Add(time.Minute))

	select {
	case update := <-updates:
		require.Equal(t, firstID, update.RecommendationID)
		require.Equal(t, "Confidence Increased", update.LatestTimelineEntry.Event)
		require.InDelta(t, 0.88, update.Confidence, 0.0001)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for updated recommendation.state.updated")
	}

	_, timeline, ok := engine.Get(firstID)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(timeline), 3)

	report := engine.Health()
	require.Equal(t, "2", report.Details["updates_processed"])
	require.Equal(t, "1", report.Details["duplicates_merged"])

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestDuplicateMergePreservesRecommendationID(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := recommendationstate.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan recommendationstate.RecommendationStateUpdated, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationStateUpdated
	})
	go func() {
		for evt := range sub.C {
			var update recommendationstate.RecommendationStateUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	payload := validValidatedPayload(at, 0.80)
	publishValidated(t, bus, payload, at)

	var firstID string
	select {
	case update := <-updates:
		firstID = update.RecommendationID
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first recommendation.state.updated")
	}

	payload["confidence"] = 0.80
	publishValidated(t, bus, payload, at.Add(time.Second))

	select {
	case update := <-updates:
		require.Equal(t, firstID, update.RecommendationID)
		require.Equal(t, recommendationstate.StatusActive, update.CurrentStatus)
	case <-time.After(2 * time.Second):
		t.Fatal("expected duplicate merge to publish state update")
	}

	report := engine.Health()
	require.Equal(t, "1", report.Details["active_recommendations"])
	require.Equal(t, "1", report.Details["duplicates_merged"])

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestRecommendationStateUpdatedPublished(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := recommendationstate.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan events.Event, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.RecommendationStateUpdated
	})
	go func() {
		for evt := range sub.C {
			updates <- evt
		}
	}()

	require.NoError(t, engine.Start(context.Background()))
	publishValidated(t, bus, validValidatedPayload(at, 0.91), at)

	select {
	case evt := <-updates:
		require.Equal(t, events.RecommendationStateUpdated, evt.Type)
		require.Equal(t, "recommendation_state_engine", evt.Source)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for recommendation.state.updated event")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}
