package researchengine_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/donchian"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestSimulatorProducesTrades(t *testing.T) {
	s := donchian.New(map[string]any{"period": 3})
	closes := []float64{10, 11, 12, 16, 17, 18, 8}
	candles := testutil.ClosesToCandles(closes)

	sim := researchengine.NewSimulator(researchengine.SimulatorConfig{
		InitialCapital: 100000,
		Quantity:       1,
		Commission:     20,
		SlippagePct:    0.01,
	})
	journal := sim.Run(s, candles)
	require.Greater(t, journal.Len(), 0)

	stats := researchengine.ComputeStatistics(journal, 100000)
	require.Greater(t, stats.TotalTrades, 0)
	require.NotEmpty(t, stats.EquityCurve)
}

func TestEngineRunStrategy(t *testing.T) {
	engine := researchengine.NewEngine(researchengine.SimulatorConfig{InitialCapital: 100000})
	s := donchian.New(map[string]any{"period": 3})
	candles := testutil.ClosesToCandles([]float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	result := engine.RunStrategy(s, candles)
	require.Equal(t, donchian.Name, result.Strategy)
	require.NotNil(t, result.Journal)
}

func TestLoadCSVCandles(t *testing.T) {
	path := "data/raw/nifty50/5min.csv"
	candles, err := researchengine.LoadCSVCandles(path, "NIFTY50", market.TF5m)
	if err != nil {
		t.Skip("csv data not available:", err)
	}
	require.NotEmpty(t, candles)
}
