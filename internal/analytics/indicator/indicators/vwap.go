package indicators

import "time"

// SessionVWAP accumulates session volume-weighted average price.
type SessionVWAP struct {
	sessionKey string
	pvSum      float64
	volSum     float64
	value      float64
	warmed     bool
}

// VWAPResult is the output of a session VWAP update.
type VWAPResult struct {
	Value    float64
	WarmedUp bool
}

// NewSessionVWAP creates a session VWAP accumulator.
func NewSessionVWAP() *SessionVWAP {
	return &SessionVWAP{}
}

// Update ingests a candle timestamp, typical price inputs, and volume.
// Session resets when the calendar date of openTime changes.
func (v *SessionVWAP) Update(openTime time.Time, high, low, close float64, volume int64) VWAPResult {
	if openTime.IsZero() || volume <= 0 {
		if v.warmed {
			return VWAPResult{Value: v.value, WarmedUp: true}
		}
		return VWAPResult{}
	}

	key := openTime.Format("2006-01-02")
	if key != v.sessionKey {
		v.sessionKey = key
		v.pvSum = 0
		v.volSum = 0
		v.warmed = false
	}

	typical := (high + low + close) / 3.0
	v.pvSum += typical * float64(volume)
	v.volSum += float64(volume)
	if v.volSum > 0 {
		v.value = v.pvSum / v.volSum
		v.warmed = true
	}

	return VWAPResult{Value: v.value, WarmedUp: v.warmed}
}

// Value returns session VWAP and warmed state.
func (v *SessionVWAP) Value() (float64, bool) {
	return v.value, v.warmed
}

// Reset clears session state.
func (v *SessionVWAP) Reset() {
	v.sessionKey = ""
	v.pvSum = 0
	v.volSum = 0
	v.value = 0
	v.warmed = false
}
