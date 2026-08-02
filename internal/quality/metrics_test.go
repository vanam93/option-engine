package quality

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testTracker(entry, current, high, low float64) activeTracker {
	return activeTracker{
		recommendationID: "REC-TEST",
		symbol:           "NIFTY",
		timeframe:        "1m",
		level:            LevelBuy,
		confidence:       0.82,
		status:           StatusActive,
		entryTime:        time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
		entryPrice:       entry,
		currentPrice:     current,
		highestPrice:     high,
		lowestPrice:      low,
		hasPrice:         true,
		startedAt:        time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
	}
}

func TestPositiveReturnUnit(t *testing.T) {
	report := NewEvaluator(testConfigUnit()).BuildReport(testTracker(100, 105, 108, 99), time.Now().UTC(), true, false)
	require.Greater(t, report.PriceStatistics.PercentageReturn, 0.0)
	require.Greater(t, report.QualityMetrics.ReturnPct, 0.0)
}

func TestNegativeReturnUnit(t *testing.T) {
	report := NewEvaluator(testConfigUnit()).BuildReport(testTracker(100, 95, 101, 93), time.Now().UTC(), true, false)
	require.Less(t, report.PriceStatistics.PercentageReturn, 0.0)
	require.Less(t, report.QualityMetrics.ReturnPct, 0.0)
}

func TestMFECalculationUnit(t *testing.T) {
	report := NewEvaluator(testConfigUnit()).BuildReport(testTracker(100, 105, 112, 98), time.Now().UTC(), false, false)
	require.InDelta(t, 0.12, report.QualityMetrics.MFE, 0.0001)
}

func TestMAECalculationUnit(t *testing.T) {
	report := NewEvaluator(testConfigUnit()).BuildReport(testTracker(100, 105, 110, 94), time.Now().UTC(), false, false)
	require.InDelta(t, 0.06, report.QualityMetrics.MAE, 0.0001)
}

func TestQualityScoreUnit(t *testing.T) {
	score := ComputeQualityScore(ScoreInput{
		ReturnPct:   0.03,
		MFE:         0.04,
		MAE:         0.01,
		HoldingMins: 30,
		Confidence:  0.85,
		Level:       LevelBuy,
		Outcome:     OutcomeSuccess,
	})
	require.Greater(t, score, 0.5)
	require.LessOrEqual(t, score, 1.0)
}

func TestClassificationUnit(t *testing.T) {
	cfg := testConfigUnit()
	require.Equal(t, ClassificationExcellent, Classify(cfg, 0.95, OutcomeSuccess))
	require.Equal(t, ClassificationGood, Classify(cfg, 0.80, OutcomeSuccess))
	require.Equal(t, ClassificationAverage, Classify(cfg, 0.60, OutcomeNeutral))
	require.Equal(t, ClassificationPoor, Classify(cfg, 0.30, OutcomeNeutral))
	require.Equal(t, ClassificationFailed, Classify(cfg, 0.95, OutcomeFailed))
}

func testConfigUnit() Config {
	return Config{
		Enabled:                true,
		SubscriberBuffer:       16,
		TrackingTimeoutMinutes: 120,
		ExcellentThreshold:     0.90,
		GoodThreshold:          0.75,
		AverageThreshold:       0.50,
		SuccessReturnPct:       0.005,
		FailureReturnPct:       -0.005,
	}
}
