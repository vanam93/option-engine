package researchengine_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/ema_cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestRunAllSharesIndicatorCache(t *testing.T) {
	closes := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}
	candles := testutil.ClosesToCandles(closes)

	engine := researchengine.NewEngine(researchengine.SimulatorConfig{InitialCapital: 100000})
	results := engine.RunAll([]strategylib.Strategy{
		ema_cross.NewDefault(),
		ema_cross.New(map[string]any{"fast": 5, "slow": 20}),
	}, candles)

	require.Len(t, results, 2)
	for _, r := range results {
		require.NotNil(t, r.Journal)
	}
}
