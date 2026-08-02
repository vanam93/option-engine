package candle

// VolumeMode defines how Tick.Volume is interpreted during aggregation.
//
// Canonical NSE market data feeds publish cumulative session volume on each
// tick. The candle engine defaults to VolumeCumulative.
//
// Volume contract:
//   - VolumeCumulative: Tick.Volume is the running total for the session.
//     Each tick contributes max(0, current_volume - last_volume) to the bar.
//     When volume is unchanged, no volume/VWAP weight is added; OHLC still updates.
//     When volume decreases (session reset), the new volume is treated as the delta.
//   - VolumeIncremental: Tick.Volume is the trade size for this tick only.
//     Each tick contributes Tick.Volume when positive; zero volume skips volume/VWAP.
type VolumeMode string

const (
	VolumeCumulative  VolumeMode = "cumulative"
	VolumeIncremental VolumeMode = "incremental"
)

// OrderPolicy defines how out-of-order ticks are handled.
type OrderPolicy string

const (
	// OrderRejectOlder drops ticks whose bucket open is before the active builder.
	OrderRejectOlder OrderPolicy = "reject_older"
	// OrderRejectStale drops ticks whose bucket open is before or equal to the
	// last closed bucket for the series.
	OrderRejectStale OrderPolicy = "reject_stale"
)
