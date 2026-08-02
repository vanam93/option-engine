package backtestrunner_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/backtestrunner"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/delivery"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/feedback"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

type mockReplayRunner struct {
	mu        sync.Mutex
	running   int
	delay     time.Duration
	lastError error
}

func (m *mockReplayRunner) Execute(ctx context.Context, req backtestrunner.SessionRequest) (time.Duration, error) {
	m.mu.Lock()
	m.running++
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.running--
		m.mu.Unlock()
	}()

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	if m.lastError != nil {
		return 0, m.lastError
	}
	return 10 * time.Millisecond, nil
}

func (m *mockReplayRunner) Active() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

type mockDeliverySource struct {
	docs []delivery.DeliveryDocument
}

func (m *mockDeliverySource) ListRecommendations(filter delivery.Filter) []delivery.DeliveryDocument {
	return append([]delivery.DeliveryDocument(nil), m.docs...)
}

func testConfig() backtestrunner.Config {
	return backtestrunner.Config{
		Enabled:            true,
		AutoStart:          false,
		ConcurrentSessions: 1,
		SubscriberBuffer:   32,
	}
}

func testRequest() backtestrunner.SessionRequest {
	start := time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC)
	return backtestrunner.SessionRequest{
		StartTime: start,
		EndTime:   start.Add(6*time.Hour + 15*time.Minute),
		Symbols:   []string{"NIFTY"},
		Mode:      backtestrunner.RunModeSingleDay,
		Speed:     1000,
		Timeframe: market.TF1m,
	}
}

func sampleDeliveryDoc(id string, level delivery.Level, confidence, ret float64, closed bool) delivery.DeliveryDocument {
	status := delivery.StatusActive
	if closed {
		status = delivery.StatusClosed
	}
	return delivery.DeliveryDocument{
		RecommendationID:         id,
		Symbol:                   "NIFTY",
		Timeframe:                "1m",
		Strategy:                 "ema_cross",
		CurrentRecommendationLevel: level,
		CurrentConfidence:        confidence,
		CurrentStatus:            status,
		CurrentReturn:            ret,
		HoldingTime:              30 * time.Minute,
		UpdatedAt:                time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
		QualityEvaluation: &delivery.QualityEvaluation{
			Outcome:        "SUCCESS",
			Classification: "GOOD",
			QualityScore:   0.78,
		},
	}
}

func publishDelivery(t *testing.T, bus *eventbus.Bus, doc delivery.DeliveryDocument) {
	t.Helper()
	payload := delivery.RecommendationDeliveryUpdated{
		RecommendationID: doc.RecommendationID,
		Symbol:           doc.Symbol,
		Timeframe:        doc.Timeframe,
		Strategy:         doc.Strategy,
		Document:         doc,
		GeneratedAt:      doc.UpdatedAt,
	}
	evt, err := events.NewEventWithTime(events.RecommendationDeliveryUpdated, "test", payload, doc.UpdatedAt)
	require.NoError(t, err)
	bus.Publish(evt)
}

func startEngine(t *testing.T, bus *eventbus.Bus, runner backtestrunner.ReplayRunner, docs []delivery.DeliveryDocument) *backtestrunner.Engine {
	t.Helper()
	engine, err := backtestrunner.New(testConfig(), bus, clock.NewReplay(time.Date(2026, 8, 2, 9, 0, 0, 0, time.UTC)), runner, &mockDeliverySource{docs: docs})
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	return engine
}

func TestSingleSession(t *testing.T) {
	bus := eventbus.New()
	runner := &mockReplayRunner{}
	docs := []delivery.DeliveryDocument{
		sampleDeliveryDoc("REC-1", delivery.LevelBuy, 0.82, 0.02, true),
	}
	engine := startEngine(t, bus, runner, docs)

	started := make(chan backtestrunner.SessionStarted, 1)
	completed := make(chan backtestrunner.SessionCompleted, 1)
	sub := bus.Subscribe(16, func(evt events.Event) bool {
		return evt.Type == events.BacktestSessionStarted || evt.Type == events.BacktestSessionCompleted
	})
	go func() {
		for evt := range sub.C {
			switch evt.Type {
			case events.BacktestSessionStarted:
				var payload backtestrunner.SessionStarted
				if err := json.Unmarshal(evt.Payload, &payload); err == nil {
					started <- payload
				}
			case events.BacktestSessionCompleted:
				var payload backtestrunner.SessionCompleted
				if err := json.Unmarshal(evt.Payload, &payload); err == nil {
					completed <- payload
				}
			}
		}
	}()
	defer sub.Close()

	publishDelivery(t, bus, docs[0])

	id, err := engine.StartSession(context.Background(), testRequest())
	require.NoError(t, err)
	require.NotEmpty(t, id)

	select {
	case evt := <-started:
		require.Equal(t, id, evt.BacktestID)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for backtest.session.started")
	}

	select {
	case evt := <-completed:
		require.Equal(t, id, evt.BacktestID)
		require.Equal(t, backtestrunner.SessionStatusCompleted, evt.Status)
		require.Equal(t, 1, evt.Summary.RecommendationsGenerated)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for backtest.session.completed")
	}

	session, ok := engine.Repository().GetSession(id)
	require.True(t, ok)
	require.Equal(t, backtestrunner.SessionStatusCompleted, session.Status)
	require.NotNil(t, session.Summary)

	require.NoError(t, engine.Close())
}

func TestMultipleSessions(t *testing.T) {
	bus := eventbus.New()
	runner := &mockReplayRunner{}
	engine, err := backtestrunner.New(testConfig(), bus, clock.NewReplay(time.Now()), runner, &mockDeliverySource{})
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))

	id1, err := engine.StartSession(context.Background(), testRequest())
	require.NoError(t, err)
	id2, err := engine.StartSession(context.Background(), testRequest())
	require.NoError(t, err)
	require.NotEqual(t, id1, id2)
	require.Equal(t, 2, engine.Repository().Count())

	require.NoError(t, engine.Close())
}

func TestSessionSummary(t *testing.T) {
	docs := []delivery.DeliveryDocument{
		sampleDeliveryDoc("REC-BEST", delivery.LevelBuy, 0.90, 0.05, true),
		sampleDeliveryDoc("REC-WORST", delivery.LevelWatch, 0.55, -0.02, false),
		sampleDeliveryDoc("REC-AVOID", delivery.LevelAvoid, 0.40, -0.01, true),
	}
	snap := backtestrunner.CollectorSnapshot{
		ResearchReports:  2,
		AlertsGenerated:  3,
		OptimizationRuns: 1,
		WalkForwardRuns:  2,
		MonteCarloRuns:   1,
		Feedback: &feedback.RecommendationFeedbackUpdated{
			Overall: feedback.OverallStatistics{
				TotalRecommendations: 3,
				SuccessRate:          0.66,
				WinRate:              0.60,
			},
		},
	}

	summary := backtestrunner.BuildSummary("BT-TEST", docs, snap, time.Now().UTC())
	require.Equal(t, 3, summary.RecommendationsGenerated)
	require.Equal(t, 2, summary.RecommendationsClosed)
	require.Equal(t, 1, summary.BuyCount)
	require.Equal(t, 1, summary.WatchCount)
	require.Equal(t, 1, summary.AvoidCount)
	require.Equal(t, "REC-BEST", summary.BestRecommendation)
	require.Equal(t, "REC-WORST", summary.WorstRecommendation)
	require.Equal(t, 2, summary.ResearchReportsGenerated)
	require.Equal(t, 3, summary.AlertsGenerated)
	require.Equal(t, 1, summary.OptimizationRuns)
	require.InDelta(t, 0.66, summary.FeedbackSummary.SuccessRate, 0.0001)
}

func TestRepositoryLookup(t *testing.T) {
	repo := backtestrunner.NewRepository()
	start := time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC)
	session := backtestrunner.Session{
		BacktestID: "BT-1",
		StartTime:  start,
		EndTime:    start.Add(6 * time.Hour),
		Status:     backtestrunner.SessionStatusCompleted,
		Request: backtestrunner.SessionRequest{
			Symbols: []string{"NIFTY", "BANKNIFTY"},
		},
	}
	repo.Save(session)

	got, ok := repo.GetSession("BT-1")
	require.True(t, ok)
	require.Equal(t, "BT-1", got.BacktestID)

	latest, ok := repo.LatestSession()
	require.True(t, ok)
	require.Equal(t, "BT-1", latest.BacktestID)

	require.Len(t, repo.ListSessions(), 1)
	require.Len(t, repo.ListByDate(start), 1)
	require.Len(t, repo.ListBySymbol("NIFTY"), 1)
	require.Len(t, repo.ListBySymbol("RELIANCE"), 0)
}

func TestGracefulShutdown(t *testing.T) {
	bus := eventbus.New()
	runner := &mockReplayRunner{delay: 200 * time.Millisecond}
	engine, err := backtestrunner.New(testConfig(), bus, clock.NewReplay(time.Now()), runner, &mockDeliverySource{})
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))

	done := make(chan struct{})
	go func() {
		_, _ = engine.StartSession(context.Background(), testRequest())
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, engine.Close())

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("session did not finish after shutdown")
	}

	report := engine.Health()
	require.False(t, report.Connected)
}

func TestReplayCompletion(t *testing.T) {
	bus := eventbus.New()
	runner := &mockReplayRunner{}
	engine := startEngine(t, bus, runner, nil)

	_, err := engine.StartSession(context.Background(), testRequest())
	require.NoError(t, err)
	require.Equal(t, 0, runner.Active())

	require.NoError(t, engine.Close())
}

func TestConcurrentSessionProtection(t *testing.T) {
	bus := eventbus.New()
	runner := &mockReplayRunner{delay: 300 * time.Millisecond}
	cfg := testConfig()
	cfg.ConcurrentSessions = 1
	engine, err := backtestrunner.New(cfg, bus, clock.NewReplay(time.Now()), runner, &mockDeliverySource{})
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))

	firstDone := make(chan struct{})
	go func() {
		_, _ = engine.StartSession(context.Background(), testRequest())
		close(firstDone)
	}()

	time.Sleep(20 * time.Millisecond)
	_, err = engine.StartSession(context.Background(), testRequest())
	require.ErrorIs(t, err, backtestrunner.ErrConcurrentLimit)

	require.NoError(t, engine.Close())

	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("first session did not complete")
	}
}

func TestHealthMetrics(t *testing.T) {
	bus := eventbus.New()
	runner := &mockReplayRunner{}
	engine := startEngine(t, bus, runner, []delivery.DeliveryDocument{
		sampleDeliveryDoc("REC-H", delivery.LevelBuy, 0.8, 0.01, true),
	})

	_, err := engine.StartSession(context.Background(), testRequest())
	require.NoError(t, err)

	report := engine.Health()
	require.Equal(t, "backtest_runner", report.Component)
	require.Equal(t, "1", report.Details["sessions_started"])
	require.Equal(t, "1", report.Details["sessions_completed"])
	require.Equal(t, "0", report.Details["sessions_failed"])
	require.Equal(t, "1", report.Details["recommendations_processed"])

	require.NoError(t, engine.Close())
}
