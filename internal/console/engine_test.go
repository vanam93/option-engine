package console_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/console"
	"github.com/vanam-gangireddy/option-engine/internal/delivery"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func testConfig() console.Config {
	return console.Config{
		Enabled:          true,
		RefreshInterval:  time.Hour,
		SubscriberBuffer: 32,
	}
}

func sampleDocument(recID string) delivery.DeliveryDocument {
	return delivery.DeliveryDocument{
		RecommendationID:           recID,
		Symbol:                       "NIFTY",
		Timeframe:                    "1m",
		Strategy:                     "ema_cross",
		Recommendation:               "BUY",
		CurrentStatus:                delivery.StatusActive,
		CurrentConfidence:            0.82,
		CurrentRecommendationLevel:   delivery.LevelBuy,
		OptimizationScore:            0.75,
		WalkForwardResult:            0.71,
		MonteCarloResult:             0.68,
		EntryPrice:                   100.0,
		LatestPrice:                  101.5,
		CurrentReturn:                0.015,
		CurrentPnL:                   1.5,
		ResearchSummary:              "Strong optimization and walk-forward support.",
		UpdatedAt:                    time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC),
		FeedbackMetrics: &delivery.FeedbackMetrics{
			StrategyWinRate: 0.68,
		},
		QualityEvaluation: &delivery.QualityEvaluation{
			Outcome:        "SUCCESS",
			Classification: "GOOD",
			QualityScore:   0.78,
		},
		Timeline: []delivery.TimelineEntry{
			{
				Timestamp: time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
				Event:     delivery.TimelineCreated,
				Reason:    "validated buy recommendation",
				NewValue:  "ACTIVE",
			},
			{
				Timestamp: time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC),
				Event:     delivery.TimelineQualityEvaluated,
				Reason:    "SUCCESS",
			},
		},
	}
}

func deliveryPayload(recID string, doc delivery.DeliveryDocument, at time.Time) map[string]any {
	return map[string]any{
		"recommendation_id": recID,
		"symbol":            doc.Symbol,
		"timeframe":         doc.Timeframe,
		"strategy":          doc.Strategy,
		"document":          doc,
		"generated_at":      at,
	}
}

func publishDelivery(t *testing.T, bus *eventbus.Bus, recID string, doc delivery.DeliveryDocument, at time.Time) {
	t.Helper()
	payload := deliveryPayload(recID, doc, at)
	evt, err := events.NewEventWithTime(events.RecommendationDeliveryUpdated, "test", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func startConsole(t *testing.T, bus *eventbus.Bus, out *bytes.Buffer) *console.Engine {
	t.Helper()
	engine, err := console.NewWithWriter(testConfig(), bus, out, false)
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	return engine
}

func TestConsoleRendering(t *testing.T) {
	bus := eventbus.New()
	var out bytes.Buffer
	engine := startConsole(t, bus, &out)

	at := time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC)
	doc := sampleDocument("REC-CONSOLE-1")
	publishDelivery(t, bus, "REC-CONSOLE-1", doc, at)

	require.Eventually(t, func() bool {
		text := out.String()
		return strings.Contains(text, "REC-CONSOLE-1") &&
			strings.Contains(text, "NIFTY") &&
			strings.Contains(text, "ema_cross") &&
			strings.Contains(text, "Buy") &&
			strings.Contains(text, "82.0%") &&
			strings.Contains(text, "Optimization Score") &&
			strings.Contains(text, "Research Summary") &&
			strings.Contains(text, "Strong optimization")
	}, 2*time.Second, 10*time.Millisecond)

	report := engine.Health()
	require.Equal(t, "recommendation_console", report.Component)
	require.Equal(t, "1", report.Details["documents_rendered"])

	require.NoError(t, engine.Close())
}

func TestIncrementalUpdates(t *testing.T) {
	bus := eventbus.New()
	var out bytes.Buffer
	engine := startConsole(t, bus, &out)

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	doc := sampleDocument("REC-CONSOLE-2")
	publishDelivery(t, bus, "REC-CONSOLE-2", doc, at)

	require.Eventually(t, func() bool {
		return strings.Contains(out.String(), "REC-CONSOLE-2")
	}, 2*time.Second, 10*time.Millisecond)

	doc.CurrentConfidence = 0.91
	doc.CurrentRecommendationLevel = delivery.LevelStrongBuy
	doc.Recommendation = "STRONG_BUY"
	doc.UpdatedAt = at.Add(5 * time.Minute)
	publishDelivery(t, bus, "REC-CONSOLE-2", doc, at.Add(5*time.Minute))

	require.Eventually(t, func() bool {
		text := out.String()
		return strings.Contains(text, "91.0%") && strings.Contains(text, "Strong Buy")
	}, 2*time.Second, 10*time.Millisecond)

	report := engine.Health()
	require.Equal(t, "1", report.Details["documents_rendered"])
	require.Equal(t, "1", report.Details["updates_rendered"])

	require.NoError(t, engine.Close())
}

func TestTimelineRendering(t *testing.T) {
	bus := eventbus.New()
	var out bytes.Buffer
	engine := startConsole(t, bus, &out)

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	doc := sampleDocument("REC-CONSOLE-3")
	publishDelivery(t, bus, "REC-CONSOLE-3", doc, at)

	require.Eventually(t, func() bool {
		text := out.String()
		return strings.Contains(text, "Timeline") &&
			strings.Contains(text, "Created") &&
			strings.Contains(text, "Quality Evaluated")
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, engine.Close())
}

func TestShutdown(t *testing.T) {
	bus := eventbus.New()
	var out bytes.Buffer
	engine := startConsole(t, bus, &out)

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	publishDelivery(t, bus, "REC-CONSOLE-4", sampleDocument("REC-CONSOLE-4"), at)

	require.Eventually(t, func() bool {
		return strings.Contains(out.String(), "REC-CONSOLE-4")
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, engine.Close())
	require.NoError(t, engine.Close())

	report := engine.Health()
	require.False(t, report.Connected)
}

func TestConsumesOnlyDeliveryUpdated(t *testing.T) {
	bus := eventbus.New()
	var out bytes.Buffer
	engine := startConsole(t, bus, &out)
	defer engine.Close()

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	statePayload := map[string]any{
		"recommendation_id": "REC-OTHER",
		"symbol":            "NIFTY",
		"current_status":    "ACTIVE",
	}
	evt, err := events.NewEventWithTime(events.RecommendationStateUpdated, "test", statePayload, at)
	require.NoError(t, err)
	bus.Publish(evt)

	time.Sleep(50 * time.Millisecond)
	require.Empty(t, out.String())
}

func TestParseDeliveryUpdate(t *testing.T) {
	doc := sampleDocument("REC-PARSE")
	payload, err := json.Marshal(delivery.RecommendationDeliveryUpdated{
		RecommendationID: "REC-PARSE",
		Document:         doc,
		GeneratedAt:      doc.UpdatedAt,
	})
	require.NoError(t, err)

	var parsed delivery.RecommendationDeliveryUpdated
	require.NoError(t, json.Unmarshal(payload, &parsed))
	require.Equal(t, "REC-PARSE", parsed.RecommendationID)
	require.Equal(t, "NIFTY", parsed.Document.Symbol)
}
