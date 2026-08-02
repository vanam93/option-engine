package query

import (
	"strconv"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	mu sync.Mutex

	requests          uint64
	errors            uint64
	totalLatency      time.Duration
	repositoryLatency time.Duration
	cacheHits         uint64
	cacheMisses       uint64
}

var globalHealth healthSnapshot

func recordRequest(latency time.Duration, err error) {
	globalHealth.mu.Lock()
	defer globalHealth.mu.Unlock()
	globalHealth.requests++
	globalHealth.totalLatency += latency
	if err != nil {
		globalHealth.errors++
	}
}

func recordRepositoryLatency(latency time.Duration, hit bool) {
	globalHealth.mu.Lock()
	defer globalHealth.mu.Unlock()
	globalHealth.repositoryLatency += latency
	if hit {
		globalHealth.cacheHits++
	} else {
		globalHealth.cacheMisses++
	}
}

// Health reports query API runtime metrics.
func Health(cfg Config) health.Report {
	globalHealth.mu.Lock()
	defer globalHealth.mu.Unlock()

	status := health.StatusHealthy
	if cfg.Enabled && globalHealth.errors > 0 && globalHealth.requests > 0 {
		if float64(globalHealth.errors)/float64(globalHealth.requests) > 0.5 {
			status = health.StatusDegraded
		}
	}

	avgLatency := float64(0)
	if globalHealth.requests > 0 {
		avgLatency = float64(globalHealth.totalLatency.Milliseconds()) / float64(globalHealth.requests)
	}
	repoLatency := float64(0)
	if globalHealth.cacheHits+globalHealth.cacheMisses > 0 {
		repoLatency = float64(globalHealth.repositoryLatency.Milliseconds()) / float64(globalHealth.cacheHits+globalHealth.cacheMisses)
	}

	return health.Report{
		Component: componentName,
		Status:    status,
		Connected: cfg.Enabled,
		Message:   "research query api",
		Details: map[string]string{
			"enabled":            boolString(cfg.Enabled),
			"requests":           u64String(globalHealth.requests),
			"errors":             u64String(globalHealth.errors),
			"average_latency":    floatString(avgLatency),
			"repository_latency": floatString(repoLatency),
			"cache_hits":         u64String(globalHealth.cacheHits),
			"cache_misses":       u64String(globalHealth.cacheMisses),
		},
	}
}

// HealthReporter wraps query health for DI registration.
type HealthReporter struct {
	cfg Config
}

// NewHealthReporter creates a health reporter for the query API.
func NewHealthReporter(cfg Config) *HealthReporter {
	return &HealthReporter{cfg: cfg.withDefaults()}
}

func (h *HealthReporter) Health() health.Report {
	return Health(h.cfg)
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
