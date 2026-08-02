package laboratory_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/backtestrunner"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

type mockBacktestRunner struct {
	mu        sync.Mutex
	running   int
	delay     time.Duration
	lastError error
	sessions  map[string]backtestrunner.Session
}

func newMockBacktestRunner() *mockBacktestRunner {
	return &mockBacktestRunner{
		sessions: make(map[string]backtestrunner.Session),
	}
}

func (m *mockBacktestRunner) StartSession(ctx context.Context, req backtestrunner.SessionRequest) (string, error) {
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
			return "", ctx.Err()
		}
	}
	if m.lastError != nil {
		return "", m.lastError
	}

	at := time.Now().UTC()
	session := backtestrunner.Session{
		BacktestID: "BT-MOCK-001",
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Status:     backtestrunner.SessionStatusCompleted,
		CreatedAt:  at,
		CompletedAt: func() *time.Time { t := at; return &t }(),
		Request:    req,
		Summary: &backtestrunner.SessionSummary{
			BacktestID:               "BT-MOCK-001",
			RecommendationsGenerated: 2,
			BuyCount:                 1,
			WatchCount:               1,
			AverageConfidence:        0.75,
			AverageReturn:            0.02,
			WinRate:                  0.5,
			OptimizationRuns:         1,
			WalkForwardRuns:          1,
			MonteCarloRuns:           1,
			ResearchReportsGenerated: 1,
			QualityDistribution:      map[string]int{"GOOD": 1},
			FeedbackSummary: backtestrunner.FeedbackSummary{
				TotalRecommendations: 2,
				SuccessRate:          0.5,
			},
			GeneratedAt: at,
		},
	}

	m.mu.Lock()
	m.sessions[session.BacktestID] = session
	m.mu.Unlock()
	return session.BacktestID, nil
}

func (m *mockBacktestRunner) GetSession(id string) (backtestrunner.Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	return session, ok
}

func (m *mockBacktestRunner) Active() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func testConfig() laboratory.Config {
	return laboratory.Config{
		Enabled:           true,
		AutoVersion:       true,
		ConcurrentStudies: 1,
	}
}

func testStudyRequest() laboratory.StudyRequest {
	start := time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC)
	return laboratory.StudyRequest{
		Name:       "EMA Cross Study",
		Strategy:   "ema_cross",
		Parameters: map[string]string{"fast": "9", "slow": "21"},
		Symbols:    []string{"NIFTY"},
		Timeframes: []market.Timeframe{market.TF1m},
		StartTime:  start,
		EndTime:    start.Add(6*time.Hour + 15*time.Minute),
	}
}

func startEngine(t *testing.T, bus *eventbus.Bus, runner *mockBacktestRunner) *laboratory.Engine {
	t.Helper()
	engine, err := laboratory.New(testConfig(), bus, clock.NewReplay(time.Date(2026, 8, 2, 9, 0, 0, 0, time.UTC)), runner, runner)
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	return engine
}

func TestCreateStudy(t *testing.T) {
	bus := eventbus.New()
	runner := newMockBacktestRunner()
	engine := startEngine(t, bus, runner)

	studyID, err := engine.CreateStudy(testStudyRequest())
	require.NoError(t, err)
	require.NotEmpty(t, studyID)

	study, ok := engine.GetStudy(studyID)
	require.True(t, ok)
	require.Equal(t, laboratory.StudyStatusPending, study.Status)
	require.Equal(t, "v1", study.ResearchVersion)
	require.Equal(t, "ema_cross", study.Strategy)

	require.NoError(t, engine.Close())
}

func TestExecuteStudy(t *testing.T) {
	bus := eventbus.New()
	runner := newMockBacktestRunner()
	engine := startEngine(t, bus, runner)

	started := make(chan laboratory.StudyStarted, 1)
	completed := make(chan laboratory.StudyCompleted, 1)
	sub := bus.Subscribe(16, func(evt events.Event) bool {
		return evt.Type == events.StudyStarted || evt.Type == events.StudyCompleted
	})
	go func() {
		for evt := range sub.C {
			switch evt.Type {
			case events.StudyStarted:
				var payload laboratory.StudyStarted
				if err := json.Unmarshal(evt.Payload, &payload); err == nil {
					started <- payload
				}
			case events.StudyCompleted:
				var payload laboratory.StudyCompleted
				if err := json.Unmarshal(evt.Payload, &payload); err == nil {
					completed <- payload
				}
			}
		}
	}()
	defer sub.Close()

	studyID, err := engine.CreateStudy(testStudyRequest())
	require.NoError(t, err)

	err = engine.ExecuteStudy(context.Background(), studyID)
	require.NoError(t, err)

	select {
	case evt := <-started:
		require.Equal(t, studyID, evt.StudyID)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for study.started")
	}

	select {
	case evt := <-completed:
		require.Equal(t, studyID, evt.StudyID)
		require.Equal(t, laboratory.StudyStatusCompleted, evt.Status)
		require.Len(t, evt.BacktestSessionIDs, 1)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for study.completed")
	}

	study, ok := engine.GetStudy(studyID)
	require.True(t, ok)
	require.Equal(t, laboratory.StudyStatusCompleted, study.Status)
	require.NotNil(t, study.Output)
	require.Len(t, study.BacktestSessionIDs, 1)
	require.Equal(t, 1, study.Output.ResearchReports)
	require.Len(t, study.Output.BacktestSummaries, 1)

	require.NoError(t, engine.Close())
}

func TestRepositoryLookup(t *testing.T) {
	repo := laboratory.NewRepository()
	start := time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC)
	study := laboratory.Study{
		StudyID:         "STUDY-1",
		Name:            "Test",
		Strategy:        "ema_cross",
		Symbols:         []string{"NIFTY"},
		StartTime:       start,
		EndTime:         start.Add(6 * time.Hour),
		Status:          laboratory.StudyStatusCompleted,
		ResearchVersion: "v1",
	}
	repo.SaveStudy(study)

	got, ok := repo.GetStudy("STUDY-1")
	require.True(t, ok)
	require.Equal(t, "STUDY-1", got.StudyID)

	latest, ok := repo.LatestStudy()
	require.True(t, ok)
	require.Equal(t, "STUDY-1", latest.StudyID)

	require.Len(t, repo.ListStudies(), 1)
	require.Len(t, repo.ListByStrategy("ema_cross"), 1)
	require.Len(t, repo.ListByVersion("v1"), 1)
}

func TestStudyComparison(t *testing.T) {
	bus := eventbus.New()
	runner := newMockBacktestRunner()
	engine := startEngine(t, bus, runner)

	compared := make(chan laboratory.StudyCompared, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.StudyCompared
	})
	go func() {
		for evt := range sub.C {
			var payload laboratory.StudyCompared
			if err := json.Unmarshal(evt.Payload, &payload); err == nil {
				compared <- payload
			}
		}
	}()
	defer sub.Close()

	studyID, err := engine.CreateStudy(testStudyRequest())
	require.NoError(t, err)
	require.NoError(t, engine.ExecuteStudy(context.Background(), studyID))

	comparison, err := engine.CompareStudies(laboratory.ComparisonCriteria{
		Strategy: "ema_cross",
		Symbol:   "NIFTY",
	})
	require.NoError(t, err)
	require.Len(t, comparison.StudyIDs, 1)
	require.Equal(t, studyID, comparison.StudyIDs[0])

	stored, ok := engine.GetComparison(comparison.ComparisonID)
	require.True(t, ok)
	require.Equal(t, comparison.ComparisonID, stored.ComparisonID)

	select {
	case evt := <-compared:
		require.Equal(t, comparison.ComparisonID, evt.ComparisonID)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for study.compared")
	}

	require.NoError(t, engine.Close())
}

func TestVersionCreation(t *testing.T) {
	bus := eventbus.New()
	runner := newMockBacktestRunner()
	engine := startEngine(t, bus, runner)

	req := testStudyRequest()
	id1, err := engine.CreateStudy(req)
	require.NoError(t, err)

	id2, err := engine.CreateVersion(id1)
	require.NoError(t, err)
	require.NotEqual(t, id1, id2)

	study1, _ := engine.GetStudy(id1)
	study2, _ := engine.GetStudy(id2)
	require.Equal(t, "v1", study1.ResearchVersion)
	require.Equal(t, "v2", study2.ResearchVersion)

	require.NoError(t, engine.Close())
}

func TestCatalogLookup(t *testing.T) {
	bus := eventbus.New()
	runner := newMockBacktestRunner()
	engine := startEngine(t, bus, runner)

	studyID, err := engine.CreateStudy(testStudyRequest())
	require.NoError(t, err)
	require.NoError(t, engine.ExecuteStudy(context.Background(), studyID))

	catalog := engine.Catalog()
	require.Equal(t, 1, catalog.Count())
	require.Contains(t, catalog.LookupByStrategy("ema_cross"), studyID)
	require.Contains(t, catalog.LookupBySymbol("NIFTY"), studyID)
	require.Contains(t, catalog.LookupByTimeframe("1m"), studyID)
	require.Contains(t, catalog.LookupByVersion("v1"), studyID)
	require.Contains(t, catalog.LookupByStatus(laboratory.StudyStatusCompleted), studyID)

	require.NoError(t, engine.Close())
}

func TestConcurrentStudyProtection(t *testing.T) {
	bus := eventbus.New()
	runner := newMockBacktestRunner()
	runner.delay = 300 * time.Millisecond
	cfg := testConfig()
	cfg.ConcurrentStudies = 1
	engine, err := laboratory.New(cfg, bus, clock.NewReplay(time.Now()), runner, runner)
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))

	id1, err := engine.CreateStudy(testStudyRequest())
	require.NoError(t, err)
	id2, err := engine.CreateStudy(testStudyRequest())
	require.NoError(t, err)

	firstDone := make(chan struct{})
	go func() {
		_ = engine.ExecuteStudy(context.Background(), id1)
		close(firstDone)
	}()

	time.Sleep(20 * time.Millisecond)
	err = engine.ExecuteStudy(context.Background(), id2)
	require.ErrorIs(t, err, laboratory.ErrConcurrentLimit)

	require.NoError(t, engine.Close())

	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("first study did not complete")
	}
}

func TestGracefulShutdown(t *testing.T) {
	bus := eventbus.New()
	runner := newMockBacktestRunner()
	runner.delay = 200 * time.Millisecond
	engine, err := laboratory.New(testConfig(), bus, clock.NewReplay(time.Now()), runner, runner)
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))

	studyID, err := engine.CreateStudy(testStudyRequest())
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		_ = engine.ExecuteStudy(context.Background(), studyID)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, engine.Close())

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("study did not finish after shutdown")
	}

	report := engine.Health()
	require.False(t, report.Connected)
}

func TestHealthMetrics(t *testing.T) {
	bus := eventbus.New()
	runner := newMockBacktestRunner()
	engine := startEngine(t, bus, runner)

	studyID, err := engine.CreateStudy(testStudyRequest())
	require.NoError(t, err)
	require.NoError(t, engine.ExecuteStudy(context.Background(), studyID))

	_, err = engine.CompareStudies(laboratory.ComparisonCriteria{Strategy: "ema_cross"})
	require.NoError(t, err)

	report := engine.Health()
	require.Equal(t, "strategy_laboratory", report.Component)
	require.Equal(t, "1", report.Details["studies_created"])
	require.Equal(t, "1", report.Details["studies_completed"])
	require.Equal(t, "0", report.Details["studies_failed"])
	require.Equal(t, "1", report.Details["comparisons"])
	require.Equal(t, "1", report.Details["repository_entries"])

	require.NoError(t, engine.Close())
}

// Ensure mock satisfies interfaces.
var (
	_ laboratory.BacktestRunner = (*mockBacktestRunner)(nil)
	_ laboratory.SessionSource  = (*mockBacktestRunner)(nil)
)
