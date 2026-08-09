package indicators

import (
	"time"
)

// OpeningRange tracks the high and low of the first N minutes of each session.
type OpeningRange struct {
	windowMinutes int
	sessionKey    string
	orHigh        float64
	orLow         float64
	windowClosed  bool
	warmed        bool
}

// OpeningRangeResult is the output of an opening range update.
type OpeningRangeResult struct {
	High          float64
	Low           float64
	WindowClosed  bool
	WarmedUp      bool
	WithinWindow  bool
}

// NewOpeningRange creates an opening range tracker for the first windowMinutes of each session.
func NewOpeningRange(windowMinutes int) *OpeningRange {
	return &OpeningRange{windowMinutes: windowMinutes}
}

// Update ingests candle open time and OHLC. Session day resets the range.
func (o *OpeningRange) Update(openTime time.Time, high, low float64) OpeningRangeResult {
	if openTime.IsZero() {
		return OpeningRangeResult{}
	}

	key := openTime.Format("2006-01-02")
	if key != o.sessionKey {
		o.sessionKey = key
		o.orHigh = high
		o.orLow = low
		o.windowClosed = false
		o.warmed = false
	}

	minutes := sessionMinutes(openTime)
	within := minutes < o.windowMinutes
	if within && !o.windowClosed {
		if high > o.orHigh {
			o.orHigh = high
		}
		if low < o.orLow {
			o.orLow = low
		}
		o.warmed = true
	} else if !o.windowClosed && minutes >= o.windowMinutes {
		o.windowClosed = true
	}

	return OpeningRangeResult{
		High:         o.orHigh,
		Low:          o.orLow,
		WindowClosed: o.windowClosed,
		WarmedUp:     o.warmed,
		WithinWindow: within,
	}
}

// Value returns opening range high, low, and whether the range is established.
func (o *OpeningRange) Value() (high, low float64, windowClosed, warmed bool) {
	return o.orHigh, o.orLow, o.windowClosed, o.warmed
}

func sessionMinutes(openTime time.Time) int {
	h, m, _ := openTime.Clock()
	// NSE cash session opens at 09:15 local exchange time.
	sessionStartMinutes := 9*60 + 15
	currentMinutes := h*60 + m
	if currentMinutes < sessionStartMinutes {
		return 0
	}
	return currentMinutes - sessionStartMinutes
}

// SessionDayKey returns the calendar date key for a session.
func SessionDayKey(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
