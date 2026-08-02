package calendar_test

import (
	"testing"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/calendar"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalendarTradingDay(t *testing.T) {
	cfg := calendar.Config{
		Timezone:     "Asia/Kolkata",
		RegularOpen:  "09:15",
		RegularClose: "15:30",
		Holidays:     []string{"2024-01-26"},
		ExpiryWeekday: time.Thursday,
	}
	cal, err := calendar.New(cfg, clock.NewSystem())
	require.NoError(t, err)

	// Republic Day holiday
	holiday := time.Date(2024, 1, 26, 10, 0, 0, 0, time.UTC)
	assert.False(t, cal.IsTradingDay(holiday))

	// Regular Thursday
	thu := time.Date(2024, 1, 18, 10, 0, 0, 0, time.UTC)
	assert.True(t, cal.IsTradingDay(thu))
	assert.True(t, cal.IsExpiryDay(thu))
}

func TestCalendarMarketHours(t *testing.T) {
	cfg := calendar.Config{
		Timezone:     "Asia/Kolkata",
		RegularOpen:  "09:15",
		RegularClose: "15:30",
	}
	cal, err := calendar.New(cfg, clock.NewSystem())
	require.NoError(t, err)

	day := time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC)
	open, err := cal.MarketOpen(day)
	require.NoError(t, err)
	close, err := cal.MarketClose(day)
	require.NoError(t, err)

	assert.True(t, close.After(open))
}

func TestMuhuratSession(t *testing.T) {
	cfg := calendar.Config{
		Timezone:     "Asia/Kolkata",
		RegularOpen:  "09:15",
		RegularClose: "15:30",
		MuhuratOpen:  "18:00",
		MuhuratClose: "19:15",
		MuhuratDays:  []string{"2024-11-01"},
	}
	cal, err := calendar.New(cfg, clock.NewSystem())
	require.NoError(t, err)

	loc, _ := time.LoadLocation("Asia/Kolkata")
	mu := time.Date(2024, 11, 1, 12, 0, 0, 0, loc)
	assert.Equal(t, calendar.SessionMuhurat, cal.SessionTypeFor(mu))
}
