package groww

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

// healthMetrics tracks provider runtime statistics.
type healthMetrics struct {
	requests        atomic.Uint64
	errors          atomic.Uint64
	retries         atomic.Uint64
	candlesStreamed atomic.Uint64
	latencyMS       atomic.Int64
	authenticated   atomic.Bool
	connected       atomic.Bool
	lastRequest     atomic.Int64
	lastResponse    atomic.Int64
}

func newHealthMetrics() *healthMetrics {
	return &healthMetrics{}
}

func (m *healthMetrics) recordRequestStart() {
	m.requests.Add(1)
	m.lastRequest.Store(time.Now().UnixMilli())
}

func (m *healthMetrics) recordRequestEnd(latency time.Duration, err error) {
	m.latencyMS.Store(latency.Milliseconds())
	m.lastResponse.Store(time.Now().UnixMilli())
	if err != nil {
		m.errors.Add(1)
	}
}

func (m *healthMetrics) recordRetry() {
	m.retries.Add(1)
}

func (m *healthMetrics) recordCandle() {
	m.candlesStreamed.Add(1)
}

func (m *healthMetrics) setConnected(v bool) {
	m.connected.Store(v)
}

func (m *healthMetrics) setAuthenticated(v bool) {
	m.authenticated.Store(v)
}

func buildHealthReport(connected bool, metrics *healthMetrics, message string) health.Report {
	status := health.StatusHealthy
	if !connected {
		status = health.StatusDegraded
	}
	if !metrics.authenticated.Load() && connected {
		status = health.StatusDegraded
	}
	var lastResp *time.Time
	if ts := metrics.lastResponse.Load(); ts > 0 {
		t := time.UnixMilli(ts)
		lastResp = &t
	}
	return health.Report{
		Component: providerName,
		Status:    status,
		Latency:   metrics.latencyMS.Load(),
		Connected: connected,
		Message:   message,
		Details: map[string]string{
			"authenticated":    boolString(metrics.authenticated.Load()),
			"requests":         u64String(metrics.requests.Load()),
			"errors":           u64String(metrics.errors.Load()),
			"retries":          u64String(metrics.retries.Load()),
			"candles_streamed": u64String(metrics.candlesStreamed.Load()),
			"latency_ms":       i64String(metrics.latencyMS.Load()),
			"last_request":     i64String(metrics.lastRequest.Load()),
			"last_response":    i64String(metrics.lastResponse.Load()),
		},
		LastEventTime: lastResp,
	}
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func u64String(v uint64) string {
	return i64String(int64(v))
}

func i64String(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
