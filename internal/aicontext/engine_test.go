package aicontext_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/aicontext"
	"github.com/vanam-gangireddy/option-engine/internal/airesearch"
	"github.com/vanam-gangireddy/option-engine/internal/backtestrunner"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

type mockStudySource struct {
	mu      sync.Mutex
	studies map[string]laboratory.Study
}

func newMockStudySource() *mockStudySource {
	return &mockStudySource{studies: make(map[string]laboratory.Study)}
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

type mockReportSource struct {
	mu      sync.Mutex
	reports map[string]airesearch.ResearchReport
}

func newMockReportSource() *mockReportSource {
	return &mockReportSource{reports: make(map[string]airesearch.ResearchReport)}
}

func (m *mockReportSource) Get(id string) (airesearch.ResearchReport, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	report, ok := m.reports[id]
	return report, ok
}

func (m *mockReportSource) save(report airesearch.ResearchReport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports[report.ReportID] = report
}

func testConfig() aicontext.Config {
	return aicontext.Config{
		Enabled:         true,
		ExecutivePrompt: true,
		TechnicalPrompt: true,
		JSONPrompt:      true,
	}
}

func testStudy(studyID string) laboratory.Study {
	start := time.Date(2026, 8, 2, 9, 15, 0, 0, time.UTC)
	completed := start.Add(6 * time.Hour)
	return laboratory.Study{
		StudyID:            studyID,
		Name:               "EMA Cross Study",
		Strategy:           "ema_cross",
		Parameters:         map[string]string{"fast": "9", "slow": "21"},
		Symbols:            []string{"NIFTY", "BANKNIFTY"},
		Timeframes:         []market.Timeframe{market.TF1m, market.TF5m},
		StartTime:          start,
		EndTime:            start.Add(6*time.Hour + 15*time.Minute),
		CreatedAt:          start,
		Status:             laboratory.StudyStatusCompleted,
		ResearchVersion:    "v1",
		CompletedAt:        &completed,
		BacktestSessionIDs: []string{"BT-MOCK-001"},
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
					AverageHoldingTime:       5 * time.Minute,
					OptimizationRuns:         2,
					WalkForwardRuns:          1,
					MonteCarloRuns:           1,
					AlertsGenerated:          1,
					SymbolDistribution:       map[string]int{"NIFTY": 3, "BANKNIFTY": 1},
					TimeframeDistribution:    map[string]int{"1m": 3, "5m": 1},
					QualityDistribution:      map[string]int{"GOOD": 2, "FAILED": 1},
				},
			},
			OptimizationSummaries: []laboratory.OptimizationSummary{
				{BacktestID: "BT-MOCK-001", OptimizationRuns: 2},
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

func testReport(reportID, studyID string) airesearch.ResearchReport {
	return airesearch.ResearchReport{
		ReportID:        reportID,
		StudyID:         studyID,
		ResearchVersion: "v1",
		Analyzer:        airesearch.AnalyzerRuleBased,
		GeneratedAt:     time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
		Sections: airesearch.ReportSections{
			ExecutiveSummary:           "Research report for EMA Cross Study with positive returns.",
			StrategyOverview:           "Strategy: ema_cross",
			Strengths:                  "Win rate meets benchmark; Walk-forward validation completed",
			Weaknesses:                 "No Monte Carlo stress tests recorded",
			ConsistencyAnalysis:        "Win rate range indicates moderate consistency.",
			RiskAnalysis:               "Risk profile acceptable.",
			ConfidenceAssessment:       "Average confidence: 0.7200 (high).",
			MarketRegimeSuitability:    "Quality distribution: GOOD=2, FAILED=1.",
			SuggestedFutureExperiments: "Expand symbol universe; Run cross-strategy comparison",
			OverallVerdict:             "PASS — strategy demonstrates strong metrics",
		},
	}
}

func startEngine(t *testing.T, bus *eventbus.Bus, studies *mockStudySource, reports *mockReportSource) *aicontext.Engine {
	t.Helper()
	engine, err := aicontext.New(testConfig(), bus, clock.NewReplay(time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC)), studies, reports)
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	return engine
}

func publishStudyAICompleted(t *testing.T, bus *eventbus.Bus, reportID, studyID string) {
	t.Helper()
	payload := airesearch.StudyAICompleted{
		ReportID:         reportID,
		StudyID:          studyID,
		ResearchVersion:  "v1",
		Analyzer:         airesearch.AnalyzerRuleBased,
		ExecutiveSummary: "summary",
		OverallVerdict:   "PASS",
		CompletedAt:      time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	bus.Publish(events.Event{
		Type:      events.StudyAICompleted,
		Source:    "test",
		Timestamp: time.Now().UTC(),
		Payload:   data,
	})
}

func TestContextGeneration(t *testing.T) {
	bus := eventbus.New()
	studies := newMockStudySource()
	reports := newMockReportSource()
	studyID := "STUDY-CTX-001"
	reportID := "REP-CTX-001"
	studies.save(testStudy(studyID))
	reports.save(testReport(reportID, studyID))

	engine := startEngine(t, bus, studies, reports)
	defer func() { _ = engine.Close() }()

	publishStudyAICompleted(t, bus, reportID, studyID)

	deadline := time.After(2 * time.Second)
	for {
		if engine.Repository().Count() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for context generation")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	ctx, ok := engine.Repository().Latest()
	require.True(t, ok)
	require.Equal(t, studyID, ctx.StudyID)
	require.Equal(t, reportID, ctx.ReportID)
	require.Equal(t, "ema_cross", ctx.Metadata.Strategy)
	require.Equal(t, 4, ctx.Performance.RecommendationsGenerated)
	require.Equal(t, 2, ctx.Optimization.TotalRuns)
	require.Equal(t, 1, ctx.WalkForward.TotalRuns)
	require.Equal(t, 1, ctx.MonteCarlo.TotalRuns)
	require.NotEmpty(t, ctx.Timeline)
	require.NotEmpty(t, ctx.KeyFindings)
}

func TestExecutivePrompt(t *testing.T) {
	builder := aicontext.NewContextBuilder(testConfig())
	study := testStudy("STUDY-EXEC-001")
	report := testReport("REP-EXEC-001", study.StudyID)

	ctx, err := builder.Build(study, report, time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC))
	require.NoError(t, err)

	require.NotEmpty(t, ctx.ExecutivePrompt)
	require.Contains(t, ctx.ExecutivePrompt, "Executive Summary")
	require.Contains(t, ctx.ExecutivePrompt, "EMA Cross Study")
	require.Contains(t, ctx.ExecutivePrompt, "ema_cross")
	require.Contains(t, ctx.ExecutivePrompt, "PASS")
	require.NotContains(t, ctx.ExecutivePrompt, "--- Walk-Forward Summary ---")
}

func TestTechnicalPrompt(t *testing.T) {
	builder := aicontext.NewContextBuilder(testConfig())
	study := testStudy("STUDY-TECH-001")
	report := testReport("REP-TECH-001", study.StudyID)

	ctx, err := builder.Build(study, report, time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC))
	require.NoError(t, err)

	require.NotEmpty(t, ctx.TechnicalPrompt)
	require.Contains(t, ctx.TechnicalPrompt, "Technical Report")
	require.Contains(t, ctx.TechnicalPrompt, "Performance Summary")
	require.Contains(t, ctx.TechnicalPrompt, "Walk-Forward Summary")
	require.Contains(t, ctx.TechnicalPrompt, "Monte Carlo Summary")
	require.Contains(t, ctx.TechnicalPrompt, "Trade Statistics")
	require.Contains(t, ctx.TechnicalPrompt, "Timeline")
	require.Contains(t, ctx.TechnicalPrompt, "fast=9")
}

func TestJSONPrompt(t *testing.T) {
	builder := aicontext.NewContextBuilder(testConfig())
	study := testStudy("STUDY-JSON-001")
	report := testReport("REP-JSON-001", study.StudyID)
	at := time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC)

	ctx1, err := builder.Build(study, report, at)
	require.NoError(t, err)
	ctx2, err := builder.Build(study, report, at)
	require.NoError(t, err)

	require.NotEmpty(t, ctx1.JSONPrompt)
	require.Contains(t, ctx1.JSONPrompt, "\"type\":\"ai_research_context\"")
	require.NotContains(t, ctx1.JSONPrompt, "---")
	require.NotContains(t, ctx1.JSONPrompt, "#")

	// Deterministic: same input produces same JSON body (context IDs differ by timestamp/uuid in Build)
	require.Equal(t, ctx1.JSONPrompt, ctx2.JSONPrompt)
}

func TestDeterministicOutput(t *testing.T) {
	builder := aicontext.NewContextBuilder(testConfig())
	study := testStudy("STUDY-DET-001")
	report := testReport("REP-DET-001", study.StudyID)
	at := time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC)

	ctx1, err := builder.Build(study, report, at)
	require.NoError(t, err)
	ctx2, err := builder.Build(study, report, at)
	require.NoError(t, err)

	require.Equal(t, ctx1.ExecutivePrompt, ctx2.ExecutivePrompt)
	require.Equal(t, ctx1.TechnicalPrompt, ctx2.TechnicalPrompt)
	require.Equal(t, ctx1.JSONPrompt, ctx2.JSONPrompt)
	require.Equal(t, ctx1.Performance, ctx2.Performance)
	require.Equal(t, ctx1.Trades, ctx2.Trades)
}

func TestRepository(t *testing.T) {
	repo := aicontext.NewRepository()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	ctx1 := aicontext.AIContext{ContextID: "CTX-001", StudyID: "STUDY-1", GeneratedAt: at}
	ctx2 := aicontext.AIContext{ContextID: "CTX-002", StudyID: "STUDY-2", GeneratedAt: at.Add(time.Minute)}

	repo.Save(ctx1)
	repo.Save(ctx2)

	got, ok := repo.Get("CTX-001")
	require.True(t, ok)
	require.Equal(t, "STUDY-1", got.StudyID)

	latest, ok := repo.Latest()
	require.True(t, ok)
	require.Equal(t, "CTX-002", latest.ContextID)

	list := repo.List()
	require.Len(t, list, 2)
}

func TestPublish(t *testing.T) {
	bus := eventbus.New()
	studies := newMockStudySource()
	reports := newMockReportSource()
	studyID := "STUDY-PUB-001"
	reportID := "REP-PUB-001"
	studies.save(testStudy(studyID))
	reports.save(testReport(reportID, studyID))

	completed := make(chan aicontext.StudyAIContextCompleted, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.StudyAIContextCompleted
	})
	go func() {
		for evt := range sub.C {
			var payload aicontext.StudyAIContextCompleted
			if err := json.Unmarshal(evt.Payload, &payload); err == nil {
				completed <- payload
			}
		}
	}()
	defer sub.Close()

	engine := startEngine(t, bus, studies, reports)
	defer func() { _ = engine.Close() }()

	publishStudyAICompleted(t, bus, reportID, studyID)

	select {
	case evt := <-completed:
		require.Equal(t, studyID, evt.StudyID)
		require.Equal(t, reportID, evt.ReportID)
		require.NotEmpty(t, evt.ContextID)
		require.Equal(t, "v1", evt.ResearchVersion)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for study.ai.context.completed")
	}
}

func TestHealth(t *testing.T) {
	bus := eventbus.New()
	studies := newMockStudySource()
	reports := newMockReportSource()
	studyID := "STUDY-HEALTH-001"
	reportID := "REP-HEALTH-001"
	studies.save(testStudy(studyID))
	reports.save(testReport(reportID, studyID))

	engine := startEngine(t, bus, studies, reports)

	publishStudyAICompleted(t, bus, reportID, studyID)

	deadline := time.After(2 * time.Second)
	for {
		if engine.Repository().Count() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for context")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	report := engine.Health()
	require.Equal(t, "ai_context_engine", report.Component)
	require.Equal(t, "1", report.Details["contexts_generated"])
	require.Equal(t, "1", report.Details["repository_entries"])

	require.NoError(t, engine.Close())
	report = engine.Health()
	require.False(t, report.Connected)
}

func TestEventIsolation(t *testing.T) {
	bus := eventbus.New()
	engine := startEngine(t, bus, newMockStudySource(), newMockReportSource())
	defer func() { _ = engine.Close() }()

	bus.Publish(events.Event{
		Type:      events.StudyCompleted,
		Source:    "test",
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage(`{"study_id":"STUDY-1"}`),
	})

	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 0, engine.Repository().Count())
}

func TestGracefulShutdown(t *testing.T) {
	bus := eventbus.New()
	engine, err := aicontext.New(testConfig(), bus, clock.NewReplay(time.Now()), newMockStudySource(), newMockReportSource())
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	require.NoError(t, engine.Close())

	report := engine.Health()
	require.False(t, report.Connected)
}

var (
	_ aicontext.StudySource  = (*mockStudySource)(nil)
	_ aicontext.ReportSource = (*mockReportSource)(nil)
)
