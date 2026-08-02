package di_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/execution"
	"github.com/vanam-gangireddy/option-engine/internal/execution/paper"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func TestDIResolvesExecutionAdapter(t *testing.T) {
	bus := eventbus.New()
	paperEngine, err := paper.New(paper.Config{
		Enabled:          true,
		SubscriberBuffer: 8,
		SlippagePercent:  0.05,
		DefaultPrice:     "100",
	}, bus, clock.NewSystem())
	require.NoError(t, err)

	var adapter execution.ExecutionAdapter = paperEngine
	require.NotNil(t, adapter)
	require.Equal(t, execution.CapabilityModeSimulated, adapter.Capabilities().Mode)

	concrete, ok := adapter.(*paper.Engine)
	require.True(t, ok)
	require.Same(t, paperEngine, concrete)
}
