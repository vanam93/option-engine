package validation_test

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
	"github.com/vanam-gangireddy/option-engine/internal/validation"
)

func testConfig() validation.Config {
	return validation.Config{
		Enabled:              true,
		SubscriberBuffer:     16,
		MinConfidence:        0.70,
		MinOptimizationScore: 0.60,
		MinWalkforwardScore:  0.60,
		MinMonteCarloScore:   0.60,
		MinWinRate:           0.50,
		MaxDrawdown:          0.20,
		FreshnessSeconds:     300,
		SuppressDuplicates:   true,
	}
}

func validRecommendationPayload(at time.Time) map[string]any {
	return map[string]any{
		"symbol":             "NIFTY",
		"timeframe":          "1m",
		"recommendation":     "BUY",
		"confidence":         0.82,
		"optimization_score": 0.75,
		"walkforward_score":  0.72,
		"monte_carlo_score":  0.68,
		"win_rate":           0.58,
		"drawdown":           0.10,
		"generated_at":       at,
	}
}

func publishRecommendation(t *testing.T, bus *eventbus.Bus, payload map[string]any, at time.Time) {
	t.Helper()
	evt, err := events.NewEventWithTime(events.RecommendationUpdated, "recommendation", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func TestDuplicateSuppression(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := validation.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan validation.ValidatedRecommendation, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.ValidatedRecommendation
	})
	go func() {
		for evt := range sub.C {
			var update validation.ValidatedRecommendation
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	payload := validRecommendationPayload(at)
	publishRecommendation(t, bus, payload, at)

	select {
	case update := <-updates:
		require.Equal(t, validation.StatusValid, update.ValidationStatus)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first validated.recommendation")
	}

	publishRecommendation(t, bus, payload, at)

	select {
	case <-updates:
		t.Fatal("duplicate recommendation should be suppressed")
	case <-time.After(300 * time.Millisecond):
	}

	report := engine.Health()
	require.Equal(t, "1", report.Details["validated"])
	require.Equal(t, "1", report.Details["duplicate_suppressed"])

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestValidatedRecommendationPublished(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := validation.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan validation.ValidatedRecommendation, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.ValidatedRecommendation
	})
	go func() {
		for evt := range sub.C {
			var update validation.ValidatedRecommendation
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	publishRecommendation(t, bus, validRecommendationPayload(at), at)

	select {
	case update := <-updates:
		require.Equal(t, "NIFTY", update.Symbol)
		require.Equal(t, "1m", update.Timeframe)
		require.Equal(t, recommendation.LevelBuy, update.Recommendation)
		require.Equal(t, validation.StatusValid, update.ValidationStatus)
		require.NotZero(t, update.ValidatedAt)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for validated.recommendation")
	}

	latest, ok := engine.Latest("NIFTY", "1m")
	require.True(t, ok)
	require.Equal(t, validation.StatusValid, latest.ValidationStatus)

	require.NoError(t, engine.Close())
	sub.Close()
}
