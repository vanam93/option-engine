package strategylib_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

func TestSignalActionable(t *testing.T) {
	sig := strategylib.Signal{Decision: strategylib.DecisionBuy}
	require.True(t, sig.IsAction())
	sig.Decision = strategylib.DecisionIgnore
	require.False(t, sig.IsAction())
}

func TestSignalBuilderAction(t *testing.T) {
	at := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	builder := strategylib.NewSignalBuilder(map[string]any{"fast": 9}, at)
	sig := builder.Action(
		strategylib.DecisionBuy,
		0.8, 0.7, 0.75,
		[]string{"test reason"},
		[]string{"tag"},
		map[string]float64{"rsi": 45},
	)
	require.Equal(t, strategylib.DecisionBuy, sig.Decision)
	require.InDelta(t, 0.8, sig.Confidence, 0.001)
	require.Equal(t, []string{"test reason"}, sig.Reasons)
	require.InDelta(t, 45, sig.Indicators["rsi"], 0.001)
	require.Equal(t, 9, sig.Parameters["fast"])
	require.Equal(t, at, sig.GeneratedAt)
}
