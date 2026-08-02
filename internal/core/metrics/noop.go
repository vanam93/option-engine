package metrics

import "time"

// NoopRegistry discards all metric observations.
type NoopRegistry struct{}

func NewNoopRegistry() Registry { return NoopRegistry{} }

type noopCounter struct{}
type noopGauge struct{}
type noopHistogram struct{}
type noopTimer struct{}

func (NoopRegistry) Counter(string, string) Counter                { return noopCounter{} }
func (NoopRegistry) Gauge(string, string) Gauge                    { return noopGauge{} }
func (NoopRegistry) Histogram(string, string, []float64) Histogram { return noopHistogram{} }
func (NoopRegistry) Timer(string, string) Timer                    { return noopTimer{} }

func (noopCounter) Inc(Labels)                          {}
func (noopCounter) Add(float64, Labels)                 {}
func (noopGauge) Set(float64, Labels)                   {}
func (noopGauge) Inc(Labels)                            {}
func (noopGauge) Dec(Labels)                            {}
func (noopHistogram) Observe(float64, Labels)           {}
func (noopTimer) ObserveDuration(time.Duration, Labels) {}
