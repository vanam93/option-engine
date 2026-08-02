package research_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/research"
)

func sampleReport() research.UnifiedReport {
	return research.UnifiedReport{
		ResearchID:   "research-test",
		ExperimentID: "exp-test",
		Version:      1,
		Strategy:     "trend_following",
		Symbol:       "NIFTY",
		Timeframe:    "5m",
		Parameters:   []byte(`{"ema_fast":9}`),
		Optimization: []research.OptimizationResult{{Score: 0.82, WinRate: 0.6}},
		Summary: research.ReportSummary{
			BestScore:           0.82,
			LatestValidation:    0.65,
			ProbabilityOfProfit: 0.7,
			RiskOfRuin:          0.1,
		},
		GeneratedAt: time.Now().UTC(),
	}
}

func TestJSONExportValidReport(t *testing.T) {
	dir := t.TempDir()
	exporter := research.JSONExporter{}
	path, err := exporter.Export(sampleReport(), dir)
	require.NoError(t, err)
	require.FileExists(t, path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded research.UnifiedReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "exp-test", decoded.ExperimentID)
	require.InDelta(t, 0.82, decoded.Summary.BestScore, 0.001)
}

func TestCSVExportCorrectRows(t *testing.T) {
	dir := t.TempDir()
	exporter := research.CSVExporter{}
	path, err := exporter.Export(sampleReport(), dir)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	require.True(t, strings.Contains(content, "experiment,experiment_id,exp-test"))
	require.True(t, strings.Contains(content, "summary,best_score,0.8200"))
	require.True(t, strings.Contains(filepath.Base(path), "exp-test_v1.csv"))
}
