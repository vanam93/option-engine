package groww

import (
	"context"
	"sync"
	"time"
)

// rateLimiter provides a simple token-bucket limiter for outbound HTTP requests.
type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

func newRateLimiter(requestsPerSecond float64) *rateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 5
	}
	return &rateLimiter{
		interval: time.Duration(float64(time.Second) / requestsPerSecond),
	}
}

func (r *rateLimiter) Wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		now := time.Now()
		var wait time.Duration
		if !r.last.IsZero() {
			if elapsed := now.Sub(r.last); elapsed < r.interval {
				wait = r.interval - elapsed
			}
		}
		if wait == 0 {
			r.last = now
			r.mu.Unlock()
			return nil
		}
		r.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
