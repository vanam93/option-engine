package researchengine_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
)

func TestQualifyRejected(t *testing.T) {
	q := researchengine.Qualify(researchengine.Statistics{
		ProfitFactor:         0.32,
		MaxDrawdownPercent:   93.83,
		WinRate:              0.21,
	})
	require.Equal(t, researchengine.QualRejected, q.Status)
	require.NotEmpty(t, q.Reasons)
	require.Contains(t, q.Recommendation, "Do not optimize")
}

func TestQualifyExcellent(t *testing.T) {
	q := researchengine.Qualify(researchengine.Statistics{
		ProfitFactor:         2.5,
		MaxDrawdownPercent:   8,
		WinRate:              0.65,
	})
	require.Equal(t, researchengine.QualExcellent, q.Status)
}

func TestQualifyGood(t *testing.T) {
	q := researchengine.Qualify(researchengine.Statistics{
		ProfitFactor:         1.8,
		MaxDrawdownPercent:   12,
		WinRate:              0.45,
	})
	require.Equal(t, researchengine.QualGood, q.Status)
}

func TestQualifyAverage(t *testing.T) {
	q := researchengine.Qualify(researchengine.Statistics{
		ProfitFactor:         1.3,
		MaxDrawdownPercent:   20,
		WinRate:              0.40,
	})
	require.Equal(t, researchengine.QualAverage, q.Status)
}

func TestQualifyPoor(t *testing.T) {
	q := researchengine.Qualify(researchengine.Statistics{
		ProfitFactor:         0.95,
		MaxDrawdownPercent:   15,
		WinRate:              0.40,
	})
	require.Equal(t, researchengine.QualPoor, q.Status)
}
