package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/alerts"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/performance"
	"github.com/vanam-gangireddy/option-engine/internal/api"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
	"github.com/vanam-gangireddy/option-engine/internal/recommendationstate"
	"github.com/vanam-gangireddy/option-engine/internal/research"
)

type mockRecommendations struct {
	items map[string]recommendationstate.Recommendation
	lines map[string][]recommendationstate.TimelineEntry
}

func (m *mockRecommendations) List(symbol, strategy, timeframe, status string, confidenceMin float64) []recommendationstate.Recommendation {
	out := make([]recommendationstate.Recommendation, 0)
	for _, item := range m.items {
		if symbol != "" && item.Symbol != symbol {
			continue
		}
		if strategy != "" && item.Strategy != strategy {
			continue
		}
		if timeframe != "" && item.Timeframe != timeframe {
			continue
		}
		if status != "" && string(item.CurrentStatus) != status {
			continue
		}
		if confidenceMin > 0 && item.Confidence < confidenceMin {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (m *mockRecommendations) Get(id string) (recommendationstate.Recommendation, []recommendationstate.TimelineEntry, bool) {
	rec, ok := m.items[id]
	if !ok {
		return recommendationstate.Recommendation{}, nil, false
	}
	return rec, m.lines[id], true
}

type mockAlerts struct{ items []alerts.AlertGenerated }

func (m *mockAlerts) List(symbol, strategy, timeframe, status string, confidenceMin float64) []alerts.AlertGenerated {
	_ = strategy
	_ = timeframe
	_ = status
	_ = confidenceMin
	out := make([]alerts.AlertGenerated, 0)
	for _, item := range m.items {
		if symbol != "" && item.Symbol != symbol {
			continue
		}
		out = append(out, item)
	}
	return out
}

type mockOpportunities struct{}

func (m *mockOpportunities) Snapshot() opportunity.OpportunitySnapshot {
	return opportunity.OpportunitySnapshot{}
}

type mockPerformance struct{}

func (m *mockPerformance) State() performance.PerformanceSnapshot {
	return performance.PerformanceSnapshot{TotalTrades: 3}
}

type mockResearch struct {
	experiments []research.ResearchExperiment
	bundles     map[string]research.ResearchBundle
}

func (m *mockResearch) ListExperiments(ctx context.Context, filter research.QueryFilter) ([]research.ResearchExperiment, error) {
	_ = ctx
	out := make([]research.ResearchExperiment, 0)
	for _, exp := range m.experiments {
		if filter.Symbol != "" && exp.Symbol != filter.Symbol {
			continue
		}
		out = append(out, exp)
	}
	return out, nil
}

func (m *mockResearch) GetResearchBundle(ctx context.Context, experimentID string) (research.ResearchBundle, error) {
	_ = ctx
	if bundle, ok := m.bundles[experimentID]; ok {
		return bundle, nil
	}
	return research.ResearchBundle{}, research.ErrNotFound
}

type mockHealth struct{}

func (m *mockHealth) Health() health.Report { return health.Report{Component: "mock"} }

func testEngine(t *testing.T) *gin.Engine {
	t.Helper()
	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	recs := &mockRecommendations{
		items: map[string]recommendationstate.Recommendation{
			"REC-1": {
				RecommendationID: "REC-1",
				Symbol:           "NIFTY",
				Timeframe:        "1m",
				Strategy:         "ema_cross",
				CurrentStatus:    recommendationstate.StatusActive,
				Confidence:       0.82,
				CreatedAt:        now,
				UpdatedAt:        now,
			},
			"REC-2": {
				RecommendationID: "REC-2",
				Symbol:           "BANKNIFTY",
				Timeframe:        "5m",
				Strategy:         "rsi",
				CurrentStatus:    recommendationstate.StatusWatch,
				Confidence:       0.55,
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		},
		lines: map[string][]recommendationstate.TimelineEntry{
			"REC-1": {
				{Timestamp: now, Event: "Recommendation Created", Reason: "created"},
				{Timestamp: now, Event: "Status Changed", Reason: "active", PreviousValue: "CREATED", NewValue: "ACTIVE"},
			},
		},
	}
	alertItems := &mockAlerts{items: []alerts.AlertGenerated{
		{AlertID: "ALT-1", RecommendationID: "REC-1", Symbol: "NIFTY", Timeframe: "1m", AlertType: alerts.AlertRecommendationCreated, GeneratedAt: now},
	}}
	researchStore := &mockResearch{
		experiments: []research.ResearchExperiment{
			{ExperimentID: "EXP-1", Strategy: "ema_cross", Symbol: "NIFTY", Timeframe: "5m", CreatedAt: now},
		},
		bundles: map[string]research.ResearchBundle{
			"EXP-1": {
				Experiment: research.ResearchExperiment{ExperimentID: "EXP-1", Symbol: "NIFTY"},
				Optimization: []research.OptimizationResult{
					{ExperimentID: "EXP-1", Score: 0.91, WinRate: 0.62},
				},
			},
		},
	}

	cfg := api.Config{Enabled: true, DefaultLimit: 10, MaxLimit: 100}
	repo := api.NewRepository(cfg, recs, alertItems, &mockOpportunities{}, &mockPerformance{}, researchStore, &mockHealth{})
	srv, err := api.NewServer(cfg, repo)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	srv.Register(engine)
	return engine
}

func TestListRecommendations(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.NotNil(t, body.Pagination)
	require.Equal(t, 2, body.Pagination.Total)
}

func TestFilterBySymbol(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?symbol=NIFTY", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "NIFTY", body.Filters.Symbol)
}

func TestRecommendationDetail(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/REC-1", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.True(t, body.Success)
}

func TestTimelineEndpoint(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/REC-1/timeline", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.True(t, body.Success)
}

func TestAlertEndpoint(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, 1, body.Pagination.Total)
}

func TestOptimizationEndpoint(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/optimization", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.True(t, body.Success)
}

func TestResearchEndpoint(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/research/EXP-1", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.True(t, body.Success)
}

func TestPagination(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?limit=1&page=1", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, 1, body.Pagination.Limit)
	require.Equal(t, 1, body.Pagination.Page)
	require.Equal(t, 2, body.Pagination.Total)
	require.NotNil(t, body.Pagination.NextPage)
}

func TestIntelligenceHealthEndpoint(t *testing.T) {
	engine := testEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/intelligence", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body api.Response
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.True(t, body.Success)
}
