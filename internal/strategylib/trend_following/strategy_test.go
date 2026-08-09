package trend_following_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/trend_following"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestTrendFollowingRuns(t *testing.T) {
	s := trend_following.New(map[string]any{"fast": 3, "slow": 5, "adx_period": 3, "adx_threshold": 5})
	closes := make([]float64, 40)
	for i := range closes {
		closes[i] = 100 + float64(i)
	}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.NotEmpty(t, decisions)
}
