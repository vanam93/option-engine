package candle

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// Duration maps a domain timeframe to a wall-clock duration.
func Duration(tf market.Timeframe) (time.Duration, error) {
	switch tf {
	case market.TF1m:
		return time.Minute, nil
	case market.TF3m:
		return 3 * time.Minute, nil
	case market.TF5m:
		return 5 * time.Minute, nil
	case market.TF15m:
		return 15 * time.Minute, nil
	case market.TF30m:
		return 30 * time.Minute, nil
	case market.TF1h:
		return time.Hour, nil
	case market.TF1d:
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("candle: unsupported timeframe %q", tf)
	}
}

// BucketStart returns the inclusive open time of the bar containing ts.
func BucketStart(ts time.Time, tf market.Timeframe, loc *time.Location) (time.Time, error) {
	d, err := Duration(tf)
	if err != nil {
		return time.Time{}, err
	}
	local := ts.In(loc)
	if tf == market.TF1d {
		y, m, day := local.Date()
		return time.Date(y, m, day, 0, 0, 0, 0, loc), nil
	}
	if d%time.Minute == 0 {
		local = local.Truncate(time.Minute)
	}
	epoch := local.UnixNano()
	width := d.Nanoseconds()
	if width <= 0 {
		return time.Time{}, fmt.Errorf("candle: invalid bucket width")
	}
	bucket := (epoch / width) * width
	return time.Unix(0, bucket).In(loc), nil
}

// BucketClose returns the exclusive close boundary of the bar that opens at open.
func BucketClose(open time.Time, tf market.Timeframe) (time.Time, error) {
	d, err := Duration(tf)
	if err != nil {
		return time.Time{}, err
	}
	return open.Add(d), nil
}
