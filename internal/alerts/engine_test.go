package alerts_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/alerts"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func testConfig() alerts.Config {
	return alerts.Config{
		Enabled:                   true,
		SubscriberBuffer:          16,
		ConfidenceChangeThreshold: 0.05,
		CooldownSeconds:           300,
	}
}

func stateUpdatePayload(recID, symbol, timeframe string, status alerts.Status, confidence float64, entry alerts.TimelineEntry) map[string]any {
	return map[string]any{
		"recommendation_id":     recID,
		"symbol":                symbol,
		"timeframe":             timeframe,
		"strategy":              "ema_cross",
		"current_status":        status,
		"confidence":            confidence,
		"latest_timeline_entry": entry,
		"summary":               "test summary",
	}
}

func publishStateUpdate(t *testing.T, bus *eventbus.Bus, payload map[string]any, at time.Time) {
	t.Helper()
	evt, err := events.NewEventWithTime(events.RecommendationStateUpdated, "recommendation_state_engine", payload, at)
	require.NoError(t, err)
	bus.Publish(evt)
}

func TestRecommendationCreatedAlert(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := alerts.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	generated := make(chan alerts.AlertGenerated, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.AlertGenerated
	})
	go func() {
		for evt := range sub.C {
			var alert alerts.AlertGenerated
			if err := json.Unmarshal(evt.Payload, &alert); err == nil {
				generated <- alert
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	entry := alerts.TimelineEntry{
		Timestamp:     at,
		Event:         "Status Changed",
		Reason:        "initial activation from validated recommendation",
		PreviousValue: "CREATED",
		NewValue:      "ACTIVE",
	}
	payload := stateUpdatePayload("REC-20260802-NIFTY-000001", "NIFTY", "1m", alerts.StatusActive, 0.82, entry)
	publishStateUpdate(t, bus, payload, at)

	var created, entryZone bool
	deadline := time.After(2 * time.Second)
	for !created || !entryZone {
		select {
		case alert := <-generated:
			switch alert.AlertType {
			case alerts.AlertRecommendationCreated:
				created = true
				require.Equal(t, "REC-20260802-NIFTY-000001", alert.RecommendationID)
				require.Contains(t, alert.AlertID, "ALT-20260802-NIFTY-")
			case alerts.AlertEntryZoneReached:
				entryZone = true
			}
		case <-deadline:
			t.Fatalf("timeout waiting for created/entry alerts: created=%v entry=%v", created, entryZone)
		}
	}

	report := engine.Health()
	require.Equal(t, "2", report.Details["alerts_generated"])
	require.Equal(t, "1", report.Details["created_alerts"])

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestStatusChangeAlert(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := alerts.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	generated := make(chan alerts.AlertGenerated, 2)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.AlertGenerated
	})
	go func() {
		for evt := range sub.C {
			var alert alerts.AlertGenerated
			if err := json.Unmarshal(evt.Payload, &alert); err == nil {
				generated <- alert
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	// Seed recommendation as already seen.
	seed := stateUpdatePayload("REC-20260802-NIFTY-000002", "NIFTY", "1m", alerts.StatusActive, 0.80, alerts.TimelineEntry{
		Timestamp: at, Event: "Status Changed", PreviousValue: "CREATED", NewValue: "ACTIVE",
	})
	publishStateUpdate(t, bus, seed, at)
	for range 2 {
		select {
		case <-generated:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout seeding recommendation")
		}
	}

	entry := alerts.TimelineEntry{
		Timestamp:     at.Add(time.Minute),
		Event:         "Status Changed",
		Reason:        "validated recommendation status changed",
		PreviousValue: "ACTIVE",
		NewValue:      "WATCH",
	}
	payload := stateUpdatePayload("REC-20260802-NIFTY-000002", "NIFTY", "1m", alerts.StatusWatch, 0.75, entry)
	publishStateUpdate(t, bus, payload, at.Add(time.Minute))

	select {
	case alert := <-generated:
		require.Equal(t, alerts.AlertStatusChanged, alert.AlertType)
		require.Equal(t, alerts.StatusWatch, alert.CurrentStatus)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for status change alert")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestConfidenceIncreaseAlert(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := alerts.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	generated := make(chan alerts.AlertGenerated, 2)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.AlertGenerated
	})
	go func() {
		for evt := range sub.C {
			var alert alerts.AlertGenerated
			if err := json.Unmarshal(evt.Payload, &alert); err == nil {
				generated <- alert
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	seed := stateUpdatePayload("REC-20260802-NIFTY-000003", "NIFTY", "1m", alerts.StatusActive, 0.70, alerts.TimelineEntry{
		Timestamp: at, Event: "Status Changed", PreviousValue: "CREATED", NewValue: "ACTIVE",
	})
	publishStateUpdate(t, bus, seed, at)
	for range 2 {
		select {
		case <-generated:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout seeding recommendation")
		}
	}

	entry := alerts.TimelineEntry{
		Timestamp:     at.Add(time.Minute),
		Event:         "Confidence Increased",
		Reason:        "validated recommendation confidence increased",
		PreviousValue: "0.7000",
		NewValue:      "0.8200",
	}
	payload := stateUpdatePayload("REC-20260802-NIFTY-000003", "NIFTY", "1m", alerts.StatusActive, 0.82, entry)
	publishStateUpdate(t, bus, payload, at.Add(time.Minute))

	select {
	case alert := <-generated:
		require.Equal(t, alerts.AlertConfidenceIncreased, alert.AlertType)
		require.InDelta(t, 0.82, alert.Confidence, 0.0001)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for confidence increase alert")
	}

	report := engine.Health()
	require.Equal(t, "1", report.Details["confidence_alerts"])

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestDuplicateSuppression(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := alerts.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	generated := make(chan alerts.AlertGenerated, 8)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.AlertGenerated
	})
	go func() {
		for evt := range sub.C {
			var alert alerts.AlertGenerated
			if err := json.Unmarshal(evt.Payload, &alert); err == nil {
				generated <- alert
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	entry := alerts.TimelineEntry{
		Timestamp:     at,
		Event:         "Status Changed",
		Reason:        "validated recommendation status changed",
		PreviousValue: "ACTIVE",
		NewValue:      "WATCH",
	}
	payload := stateUpdatePayload("REC-20260802-NIFTY-000004", "NIFTY", "1m", alerts.StatusWatch, 0.72, entry)

	// First observation still emits created + status alerts.
	publishStateUpdate(t, bus, payload, at)
	firstBatch := 0
	deadline := time.After(2 * time.Second)
	for firstBatch < 2 {
		select {
		case <-generated:
			firstBatch++
		case <-deadline:
			t.Fatalf("timeout on first publish, got %d alerts", firstBatch)
		}
	}

	// Identical update should not emit another status alert.
	publishStateUpdate(t, bus, payload, at.Add(time.Second))
	select {
	case alert := <-generated:
		t.Fatalf("expected duplicate suppression, got alert %s", alert.AlertType)
	case <-time.After(300 * time.Millisecond):
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestCooldownBehavior(t *testing.T) {
	cfg := testConfig()
	cfg.CooldownSeconds = 60
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := alerts.New(cfg, bus, clock.NewReplay(at))
	require.NoError(t, err)

	generated := make(chan alerts.AlertGenerated, 8)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.AlertGenerated
	})
	go func() {
		for evt := range sub.C {
			var alert alerts.AlertGenerated
			if err := json.Unmarshal(evt.Payload, &alert); err == nil {
				generated <- alert
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	seed := stateUpdatePayload("REC-20260802-NIFTY-000005", "NIFTY", "1m", alerts.StatusActive, 0.70, alerts.TimelineEntry{
		Timestamp: at, Event: "Status Changed", PreviousValue: "CREATED", NewValue: "ACTIVE",
	})
	publishStateUpdate(t, bus, seed, at)
	for range 2 {
		select {
		case <-generated:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout seeding recommendation")
		}
	}

	confidenceEntry := alerts.TimelineEntry{
		Timestamp:     at.Add(time.Minute),
		Event:         "Confidence Increased",
		Reason:        "validated recommendation confidence increased",
		PreviousValue: "0.7000",
		NewValue:      "0.8200",
	}
	payload := stateUpdatePayload("REC-20260802-NIFTY-000005", "NIFTY", "1m", alerts.StatusActive, 0.82, confidenceEntry)
	publishStateUpdate(t, bus, payload, at.Add(time.Minute))

	select {
	case <-generated:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first confidence alert")
	}

	// Within cooldown: another confidence increase suppressed despite meaningful delta.
	payload["confidence"] = 0.90
	confidenceEntry.NewValue = "0.9000"
	confidenceEntry.PreviousValue = "0.8200"
	confidenceEntry.Timestamp = at.Add(time.Minute + 30*time.Second)
	payload["latest_timeline_entry"] = confidenceEntry
	publishStateUpdate(t, bus, payload, at.Add(time.Minute+30*time.Second))

	select {
	case alert := <-generated:
		t.Fatalf("expected cooldown suppression, got alert %s", alert.AlertType)
	case <-time.After(300 * time.Millisecond):
	}

	report := engine.Health()
	require.Equal(t, "1", report.Details["cooldown_suppressed"])

	confidenceEntry.Timestamp = at.Add(2*time.Minute + time.Second)
	confidenceEntry.PreviousValue = "0.9000"
	confidenceEntry.NewValue = "0.9600"
	payload["confidence"] = 0.96
	payload["latest_timeline_entry"] = confidenceEntry
	publishStateUpdate(t, bus, payload, at.Add(2*time.Minute+time.Second))

	select {
	case alert := <-generated:
		require.Equal(t, alerts.AlertConfidenceIncreased, alert.AlertType)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for post-cooldown confidence alert")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestAlertGeneratedPublished(t *testing.T) {
	bus := eventbus.New()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := alerts.New(testConfig(), bus, clock.NewReplay(at))
	require.NoError(t, err)

	updates := make(chan events.Event, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.AlertGenerated
	})
	go func() {
		for evt := range sub.C {
			updates <- evt
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	seed := stateUpdatePayload("REC-20260802-NIFTY-000006", "NIFTY", "1m", alerts.StatusActive, 0.80, alerts.TimelineEntry{
		Timestamp: at, Event: "Status Changed", PreviousValue: "CREATED", NewValue: "ACTIVE",
	})
	publishStateUpdate(t, bus, seed, at)
	for range 2 {
		select {
		case <-updates:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout seeding recommendation")
		}
	}

	entry := alerts.TimelineEntry{
		Timestamp:     at.Add(time.Minute),
		Event:         "Closed",
		Reason:        "validation rejected",
		PreviousValue: "ACTIVE",
		NewValue:      "CLOSED",
	}
	payload := stateUpdatePayload("REC-20260802-NIFTY-000006", "NIFTY", "1m", alerts.StatusClosed, 0.65, entry)
	publishStateUpdate(t, bus, payload, at.Add(time.Minute))

	select {
	case evt := <-updates:
		require.Equal(t, events.AlertGenerated, evt.Type)
		require.Equal(t, "alert_engine", evt.Source)

		var alert alerts.AlertGenerated
		require.NoError(t, json.Unmarshal(evt.Payload, &alert))
		require.Equal(t, alerts.AlertRecommendationClosed, alert.AlertType)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for alert.generated event")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}
