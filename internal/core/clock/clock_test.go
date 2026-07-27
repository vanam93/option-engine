package clock_test

import (
	"testing"
	"time"

	"github.com/option-engine/option-engine/internal/core/clock"
	"github.com/stretchr/testify/assert"
)

func TestSystemClock(t *testing.T) {
	clk := clock.NewSystem()
	before := time.Now().UTC()
	now := clk.Now()
	assert.False(t, now.Before(before))
}

func TestReplayClock(t *testing.T) {
	start := time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	rc := clock.NewReplay(start)

	assert.Equal(t, start, rc.Now())
	rc.Advance(time.Minute)
	assert.Equal(t, start.Add(time.Minute), rc.Now())

	rc.Set(start.Add(2 * time.Hour))
	assert.Equal(t, start.Add(2*time.Hour), rc.Now())
}
