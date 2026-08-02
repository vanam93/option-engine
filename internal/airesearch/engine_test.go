package airesearch_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/airesearch"
	"github.com/vanam-gangireddy/option-engine/internal/backtestrunner"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

type mockStudySource struct {
	mu     sync.Mutex
	studies map[string]laboratory.Study
}

func newMockStudySource() *mockStudySource {
	return &mockStudySource{
		studies: make(map[string]laboratory.Study),
	}
}

func (m *mockStudySource) GetStudy(id string) (laboratory.Study, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	study, ok := m.studies[id]
	return study, ok
}

func (m *mockStudySource) save(study laboratory.Study) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.studies[study.StudyID] = study
}

func testConfig() airesearch.Config {
	return airesearch.Config{
		Enabled:  true,
		Analyzer: airesearch.AnalyzerRuleBased,
	}
}

func testStudy(studyID string) laboratory.Study {
	start := time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC)
	completed := start.Add(6 * time.Hour)
	return laboratory.Study{
		StudyID:         studyID,
		Name:            "EMA Cross Study",
		Strategy:        "ema_cross",
		Parameters:      map[string]string{"fast": "9", "slow": "21"},
		Symbols:         []string{"NIFTY", "BANKNIFTY"},
		Timeframes:      []market.Timeframe{market.TF1m, market.TF5m},
		StartTime:       start,
		EndTime:         start.Add(6*time.Hour + 15*time.Minute),
		Status:          laboratory.StudyStatusCompleted,
		ResearchVersion: "v1",
		CompletedAt:     &completed,
		Output: &laboratory.StudyOutput{
			BacktestSummaries: []backtestrunner.SessionSummary{
				{
					BacktestID:               "BT-MOCK-001",
					RecommendationsGenerated: 4,
					RecommendationsClosed:    3,
					BuyCount:                 2,
					WatchCount:               1,
					AvoidCount:               1,
					AverageConfidence:        0.72,
					HighestConfidence:        0.85,
					LowestConfidence:         0.55,
					AverageReturn:            0.03,
					WinRate:                  0.6,
					LossRate:                 0.4,
					OptimizationRuns:         2,
					WalkForwardRuns:          1,
					MonteCarloRuns:           1,
					AlertsGenerated:          1,
					ResearchReportsGenerated: 1,
					SymbolDistribution:       map[string]int{"NIFTY": 3, "BANKNIFTY": 1},
					TimeframeDistribution:    map[string]int{"1m": 3, "5m": 1},
					QualityDistribution:      map[string]int{"GOOD": 2, "AVERAGE": 1},
				},
			},
			WalkForwardSummaries: []laboratory.WalkForwardSummary{
				{BacktestID: "BT-MOCK-001", WalkForwardRuns: 1},
			},
			MonteCarloSummaries: []laboratory.MonteCarloSummary{
				{BacktestID: "BT-MOCK-001", MonteCarloRuns: 1},
			},
		},
	}
}

func startEngine(t *testing.T, bus *eventbus.Bus, source *mockStudySource) *airesearch.Engine {
	t.Helper()
	engine, err := airesearch.New(testConfig(), bus, clock.NewReplay(time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)), source)
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	return engine
}

func publishStudyCompleted(t *testing.T, bus *eventbus.Bus, studyID string) {
	t.Helper()
	payload := laboratory.StudyCompleted{
		StudyID:            studyID,
		Status:             laboratory.StudyStatusCompleted,
		ResearchVersion:    "v1",
		BacktestSessionIDs: []string{"BT-MOCK-001"},
		CompletedAt:        time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	bus.Publish(events.Event{
		Type:      events.StudyCompleted,
		Source:    "test",
		Timestamp: time.Now().UTC(),
		Payload:   data,
	})
}

func TestStudyConsumption(t *testing.T) {
	bus := eventbus.New()
	source := newMockStudySource()
	studyID := "STUDY-TEST-001"
	source.save(testStudy(studyID))

	engine := startEngine(t, bus, source)
	defer func() { _ = engine.Close() }()

	publishStudyCompleted(t, bus, studyID)

	deadline := time.After(2 * time.Second)
	for {
		if engine.Repository().Count() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for report generation")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	report, ok := engine.Repository().Latest()
	require.True(t, ok)
	require.Equal(t, studyID, report.StudyID)
	require.NotEmpty(t, report.Sections.ExecutiveSummary)
	require.NotEmpty(t, report.Sections.OverallVerdict)
}

func TestReportGeneration(t *testing.T) {
	builder := airesearch.NewReportBuilder(testConfig(), airesearch.NewRuleBasedAnalyzer())
	study := testStudy("STUDY-REPORT-001")

	report, err := builder.Build(context.Background(), study, time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	require.NotEmpty(t, report.ReportID)
	require.Equal(t, study.StudyID, report.StudyID)
	require.NotEmpty(t, report.Prompt)
	require.Contains(t, report.Prompt, "ema_cross")
	require.NotEmpty(t, report.FormattedText)

	sections := report.Sections
	require.NotEmpty(t, sections.ExecutiveSummary)
	require.NotEmpty(t, sections.StrategyOverview)
	require.NotEmpty(t, sections.Strengths)
	require.NotEmpty(t, sections.Weaknesses)
	require.NotEmpty(t, sections.BestPerformingSymbols)
	require.NotEmpty(t, sections.WorstPerformingSymbols)
	require.NotEmpty(t, sections.BestTimeframes)
	require.NotEmpty(t, sections.WorstTimeframes)
	require.NotEmpty(t, sections.ParameterSensitivity)
	require.NotEmpty(t, sections.ConsistencyAnalysis)
	require.NotEmpty(t, sections.WalkForwardSummary)
	require.NotEmpty(t, sections.MonteCarloSummary)
	require.NotEmpty(t, sections.RiskAnalysis)
	require.NotEmpty(t, sections.ConfidenceAssessment)
	require.NotEmpty(t, sections.MarketRegimeSuitability)
	require.NotEmpty(t, sections.SuggestedImprovements)
	require.NotEmpty(t, sections.SuggestedFutureExperiments)
	require.NotEmpty(t, sections.OverallVerdict)
	require.Contains(t, sections.BestPerformingSymbols, "NIFTY")
}

func TestRepository(t *testing.T) {
	repo := airesearch.NewRepository()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	report1 := airesearch.ResearchReport{
		ReportID:   "REP-001",
		StudyID:    "STUDY-1",
		GeneratedAt: at,
	}
	report2 := airesearch.ResearchReport{
		ReportID:   "REP-002",
		StudyID:    "STUDY-2",
		GeneratedAt: at.Add(time.Minute),
	}

	repo.Save(report1)
	repo.Save(report2)

	got, ok := repo.Get("REP-001")
	require.True(t, ok)
	require.Equal(t, "STUDY-1", got.StudyID)

	latest, ok := repo.Latest()
	require.True(t, ok)
	require.Equal(t, "REP-002", latest.ReportID)

	list := repo.List()
	require.Len(t, list, 2)
	require.Equal(t, "REP-001", list[0].ReportID)
	require.Equal(t, "REP-002", list[1].ReportID)
}

func TestPublish(t *testing.T) {
	bus := eventbus.New()
	source := newMockStudySource()
	studyID := "STUDY-PUBLISH-001"
	source.save(testStudy(studyID))

	completed := make(chan airesearch.StudyAICompleted, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.StudyAICompleted
	})
	go func() {
		for evt := range sub.C {
			var payload airesearch.StudyAICompleted
			if err := json.Unmarshal(evt.Payload, &payload); err == nil {
				completed <- payload
			}
		}
	}()
	defer sub.Close()

	engine := startEngine(t, bus, source)
	defer func() { _ = engine.Close() }()

	publishStudyCompleted(t, bus, studyID)

	select {
	case evt := <-completed:
		require.Equal(t, studyID, evt.StudyID)
		require.NotEmpty(t, evt.ReportID)
		require.Equal(t, airesearch.AnalyzerRuleBased, evt.Analyzer)
		require.NotEmpty(t, evt.ExecutiveSummary)
		require.NotEmpty(t, evt.OverallVerdict)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for study.ai.completed")
	}
}

func TestHealth(t *testing.T) {
	bus := eventbus.New()
	source := newMockStudySource()
	studyID := "STUDY-HEALTH-001"
	source.save(testStudy(studyID))

	engine := startEngine(t, bus, source)

	publishStudyCompleted(t, bus, studyID)

	deadline := time.After(2 * time.Second)
	for {
		if engine.Repository().Count() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for report")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	report := engine.Health()
	require.Equal(t, "ai_research_engine", report.Component)
	require.Equal(t, "1", report.Details["reports_generated"])
	require.Equal(t, "1", report.Details["repository_entries"])
	require.Equal(t, airesearch.AnalyzerRuleBased, report.Details["analyzer"])

	require.NoError(t, engine.Close())
	report = engine.Health()
	require.False(t, report.Connected)
}

func TestEventIsolation(t *testing.T) {
	bus := eventbus.New()
	source := newMockStudySource()
	engine := startEngine(t, bus, source)
	defer func() { _ = engine.Close() }()

	bus.Publish(events.Event{
		Type:      events.StudyStarted,
		Source:    "test",
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage(`{"study_id":"STUDY-1"}`),
	})

	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 0, engine.Repository().Count())
}

func TestGracefulShutdown(t *testing.T) {
	bus := eventbus.New()
	source := newMockStudySource()
	engine, err := airesearch.New(testConfig(), bus, clock.NewReplay(time.Now()), source)
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	require.NoError(t, engine.Close())

	report := engine.Health()
	require.False(t, report.Connected)
}

// Ensure mock satisfies interface.
var _ airesearch.StudySource = (*mockStudySource)(nil)
