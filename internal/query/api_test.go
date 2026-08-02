package query_test

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
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
	"github.com/vanam-gangireddy/option-engine/internal/query"
	"github.com/vanam-gangireddy/option-engine/internal/recommendationstate"
	"github.com/vanam-gangireddy/option-engine/internal/research"
	"github.com/vanam-gangireddy/option-engine/internal/scanner"
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

type mockScanner struct{}

func (m *mockScanner) Snapshot() scanner.ScannerSnapshot { return scanner.ScannerSnapshot{} }

type mockPerformance struct{}

func (m *mockPerformance) State() performance.PerformanceSnapshot {
	return performance.PerformanceSnapshot{TotalTrades: 3}
}

type mockOptimization struct{}

func (m *mockOptimization) State() optimization.StateSnapshot { return optimization.StateSnapshot{} }

type mockResearch struct{}

func (m *mockResearch) GetResearchBundle(ctx context.Context, experimentID string) (research.ResearchBundle, error) {
	return research.ResearchBundle{}, research.ErrNotFound
}

type mockHealth struct{}

func (m *mockHealth) Health() health.Report { return health.Report{Component: "mock"} }

func testAPI(t *testing.T) (*query.API, *gin.Engine) {
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
		{AlertID: "ALT-1", RecommendationID: "REC-1", Symbol: "NIFTY", Timeframe: "1m", AlertType: alerts.AlertRecommendationCreated},
	}}

	repo := query.NewRepository(recs, alertItems, &mockOpportunities{}, &mockScanner{}, &mockPerformance{}, &mockOptimization{}, &mockResearch{}, &mockHealth{})
	api, err := query.NewAPI(query.Config{Enabled: true, DefaultLimit: 10, MaxLimit: 100}, repo)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	query.RegisterRoutes(engine.Group("/api/v1"), api)
	return api, engine
}

func TestListRecommendations(t *testing.T) {
	_, engine := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body query.ListResponse[query.RecommendationView]
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
	require.Equal(t, 2, body.Metadata.Pagination.Total)
}

func TestFilterBySymbol(t *testing.T) {
	_, engine := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?symbol=NIFTY", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body query.ListResponse[query.RecommendationView]
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	require.Equal(t, "NIFTY", body.Data[0].Symbol)
	require.Equal(t, "NIFTY", body.Metadata.Filters.Symbol)
}

func TestRecommendationDetail(t *testing.T) {
	_, engine := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/REC-1", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body query.ItemResponse[query.RecommendationView]
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "REC-1", body.Data.RecommendationID)
}

func TestTimelineEndpoint(t *testing.T) {
	_, engine := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/REC-1/timeline", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body query.TimelineResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "REC-1", body.ID)
	require.Len(t, body.Timeline, 2)
}

func TestAlertEndpoint(t *testing.T) {
	_, engine := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body query.ListResponse[query.AlertView]
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	require.Equal(t, "ALT-1", body.Data[0].AlertID)
}

func TestPagination(t *testing.T) {
	_, engine := testAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?limit=1&offset=0", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body query.ListResponse[query.RecommendationView]
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	require.Equal(t, 1, body.Metadata.Pagination.Limit)
	require.Equal(t, 0, body.Metadata.Pagination.Offset)
	require.Equal(t, 2, body.Metadata.Pagination.Total)
	require.True(t, body.Metadata.Pagination.HasMore)
}
