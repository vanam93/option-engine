package metrics

import "time"

// Labels holds dimensional metadata for a metric observation.
type Labels map[string]string

// Counter tracks monotonically increasing values.
type Counter interface {
	Inc(labels Labels)
	Add(value float64, labels Labels)
}

// Gauge tracks a value that can go up or down.
type Gauge interface {
	Set(value float64, labels Labels)
	Inc(labels Labels)
	Dec(labels Labels)
}

// Histogram records distributions (future Prometheus histogram).
type Histogram interface {
	Observe(value float64, labels Labels)
}

// Timer is a convenience for latency histograms.
type Timer interface {
	ObserveDuration(d time.Duration, labels Labels)
}

// Registry collects named metrics. Implementations may export to Prometheus later.
type Registry interface {
	Counter(name, help string) Counter
	Gauge(name, help string) Gauge
	Histogram(name, help string, buckets []float64) Histogram
	Timer(name, help string) Timer
}
