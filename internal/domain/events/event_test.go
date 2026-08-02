package events_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

func TestNewEvent(t *testing.T) {
	tick := market.Tick{Symbol: "NIFTY", LTP: 22500.50}

	evt, err := events.NewEvent(events.MarketDataReceived, "test-provider", tick)
	require.NoError(t, err)

	assert.NotEmpty(t, evt.ID)
	assert.Equal(t, events.MarketDataReceived, evt.Type)
	assert.Equal(t, "test-provider", evt.Source)
	assert.NotEmpty(t, evt.Payload)
	assert.False(t, evt.Timestamp.IsZero())
}
