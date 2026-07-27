package clock

import "time"

// Clock abstracts time for testability and replay mode.
// Business logic must depend on Clock instead of time.Now().
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
}
