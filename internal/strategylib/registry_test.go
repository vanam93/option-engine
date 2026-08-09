package strategylib_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/catalog"
)

func TestRegistrySearch(t *testing.T) {
	catalog.RegisterAll()

	all := strategylib.All()
	require.Len(t, all, 13)

	trend := strategylib.ByCategory(strategylib.CategoryTrend)
	require.NotEmpty(t, trend)

	regime := strategylib.ByRegime(strategylib.RegimeTrending)
	require.NotEmpty(t, regime)

	tf := strategylib.ByTimeframe("5m")
	require.NotEmpty(t, tf)

	risk := strategylib.ByRisk(strategylib.RiskMedium)
	require.NotEmpty(t, risk)

	supported := strategylib.SupportedStrategies("NIFTY50", "5m")
	require.NotEmpty(t, supported)

	s, ok := strategylib.Get("ema_cross")
	require.True(t, ok)
	require.Equal(t, "ema_cross", s.Name())
	require.NotEmpty(t, s.Metadata().Description)
	require.Greater(t, s.WarmupBars(), 0)

	desc, ok := strategylib.GetDescriptor("ema_cross")
	require.True(t, ok)
	require.Equal(t, "ema_cross", desc.Name)
	require.NotEmpty(t, desc.Version)
	require.NotEmpty(t, desc.ParameterRanges)

	names := strategylib.Names()
	require.Len(t, names, 13)
}
