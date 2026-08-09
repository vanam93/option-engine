package researchengine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/donchian"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/ema_cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestChampionshipRanksStrategies(t *testing.T) {
	closes := []float64{10, 11, 12, 16, 17, 18, 8, 9, 10, 11, 12, 20, 21, 22, 23, 24, 25}
	candles := testutil.ClosesToCandles(closes)

	champ := researchengine.NewChampionshipEngine(researchengine.SimulatorConfig{
		InitialCapital: 100000,
		Quantity:       1,
	}, researchengine.DefaultRankingWeights())

	strategies := []strategylib.Strategy{
		donchian.New(map[string]any{"period": 3}),
		ema_cross.New(map[string]any{"fast": 3, "slow": 5}),
	}

	board := champ.Run(strategies, candles, researchengine.ChampionshipMeta{DataSource: "test"})
	require.Len(t, board.Entries, 2)
	require.Equal(t, 2, board.StrategyCount)
	require.Equal(t, 1, board.Entries[0].Rank)

	for i := 1; i < len(board.Entries); i++ {
		require.GreaterOrEqual(t, board.Entries[i-1].OverallScore, board.Entries[i].OverallScore)
	}
	require.Greater(t, board.Entries[0].TotalTrades, 0)
}

func TestChampionshipExport(t *testing.T) {
	board := researchengine.StrategyLeaderboard{
		Symbol:    "TEST",
		Timeframe: "5m",
		Entries: []researchengine.LeaderboardEntry{
			{Rank: 1, Strategy: "ema_cross", Category: "trend", OverallScore: 0.75, TotalTrades: 10, WinRate: 0.6, ProfitFactor: 1.5, SharpeRatio: 1.1, MaxDrawdownPct: 5, NetProfit: 1000, Expectancy: 100},
		},
	}

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "leaderboard.json")
	csvPath := filepath.Join(dir, "leaderboard.csv")

	require.NoError(t, researchengine.ExportLeaderboardJSON(board, jsonPath))
	require.NoError(t, researchengine.ExportLeaderboardCSV(board, csvPath))

	_, err := os.Stat(jsonPath)
	require.NoError(t, err)
	_, err = os.Stat(csvPath)
	require.NoError(t, err)

	out := researchengine.FormatLeaderboard(board)
	require.Contains(t, out, "ema_cross")
	require.Contains(t, out, "Best strategy")
}

func TestComputeOverallScoreZeroTrades(t *testing.T) {
	score, comps := researchengine.ComputeOverallScore(researchengine.Statistics{}, researchengine.DefaultRankingWeights())
	require.Equal(t, 0.0, score)
	require.Equal(t, researchengine.ScoreComponents{}, comps)
}
