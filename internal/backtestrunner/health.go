package backtestrunner

import (
	"strconv"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	sessionsStarted          uint64
	sessionsCompleted        uint64
	sessionsFailed           uint64
	recommendationsProcessed uint64
	totalSessionDurationNs   uint64
	sessionSamples           uint64
}

func (h *healthSnapshot) recordStarted() {
	h.sessionsStarted++
}

func (h *healthSnapshot) recordCompleted(duration time.Duration, recommendations int) {
	h.sessionsCompleted++
	h.recommendationsProcessed += uint64(recommendations)
	h.totalSessionDurationNs += uint64(duration.Nanoseconds())
	h.sessionSamples++
}

func (h *healthSnapshot) recordFailed() {
	h.sessionsFailed++
}

func (h *healthSnapshot) averageSessionDurationMs() float64 {
	if h.sessionSamples == 0 {
		return 0
	}
	return float64(h.totalSessionDurationNs) / float64(h.sessionSamples) / 1e6
}

func (h *healthSnapshot) report(cfg Config, connected bool, activeSessions int) health.Report {
	status := health.StatusHealthy
	if cfg.Enabled && !connected {
		status = health.StatusDegraded
	}

	return health.Report{
		Component: engineName,
		Status:    status,
		Connected: connected,
		Message:   "historical backtest runner",
		Details: map[string]string{
			"enabled":                      boolString(cfg.Enabled),
			"auto_start":                   boolString(cfg.AutoStart),
			"concurrent_sessions":          strconv.Itoa(cfg.ConcurrentSessions),
			"active_sessions":              strconv.Itoa(activeSessions),
			"sessions_started":             u64String(h.sessionsStarted),
			"sessions_completed":           u64String(h.sessionsCompleted),
			"sessions_failed":              u64String(h.sessionsFailed),
			"average_session_duration_ms":  floatString(h.averageSessionDurationMs()),
			"recommendations_processed":    u64String(h.recommendationsProcessed),
		},
	}
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func u64String(v uint64) string {
	return strconv.FormatUint(v, 10)
}

func floatString(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}
