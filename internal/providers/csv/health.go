package csv

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

// healthMetrics tracks CSV provider runtime statistics.
type healthMetrics struct {
	rowsRead              atomic.Uint64
	candlesPublished      atomic.Uint64
	parseErrors           atomic.Uint64
	publishErrors         atomic.Uint64
	connected             atomic.Bool
	currentFile           atomic.Value // string
	currentOffset         atomic.Int64
	totalPublishLatencyNS atomic.Uint64
	publishSamples        atomic.Uint64
}

func newHealthMetrics() *healthMetrics {
	m := &healthMetrics{}
	m.currentFile.Store("")
	return m
}

func (m *healthMetrics) recordRowRead() {
	m.rowsRead.Add(1)
}

func (m *healthMetrics) recordCandlePublished(latency time.Duration) {
	m.candlesPublished.Add(1)
	if latency > 0 {
		m.totalPublishLatencyNS.Add(uint64(latency))
		m.publishSamples.Add(1)
	}
}

func (m *healthMetrics) recordParseError() {
	m.parseErrors.Add(1)
}

func (m *healthMetrics) syncParseErrors(total int64) {
	m.parseErrors.Store(uint64(total))
}

func (m *healthMetrics) recordPublishError() {
	m.publishErrors.Add(1)
}

func (m *healthMetrics) setConnected(v bool) {
	m.connected.Store(v)
}

func (m *healthMetrics) setCurrentFile(path string) {
	m.currentFile.Store(path)
}

func (m *healthMetrics) setCurrentOffset(offset int64) {
	m.currentOffset.Store(offset)
}

func (m *healthMetrics) averagePublishLatency() time.Duration {
	samples := m.publishSamples.Load()
	if samples == 0 {
		return 0
	}
	avgNS := m.totalPublishLatencyNS.Load() / samples
	return time.Duration(avgNS)
}

func buildHealthReport(connected bool, metrics *healthMetrics, message string) health.Report {
	status := health.StatusHealthy
	if !connected {
		status = health.StatusDegraded
	}
	currentFile, _ := metrics.currentFile.Load().(string)
	return health.Report{
		Component: providerName,
		Status:    status,
		Connected: connected,
		Message:   message,
		Details: map[string]string{
			"rows_read":                 u64String(metrics.rowsRead.Load()),
			"candles_published":         u64String(metrics.candlesPublished.Load()),
			"parse_errors":              u64String(metrics.parseErrors.Load()),
			"publish_errors":            u64String(metrics.publishErrors.Load()),
			"current_file":              currentFile,
			"current_offset":            i64String(metrics.currentOffset.Load()),
			"average_publish_latency":   metrics.averagePublishLatency().String(),
		},
	}
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
