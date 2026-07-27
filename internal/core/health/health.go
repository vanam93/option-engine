package health

import "time"

// Status represents the health of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
	StatusUnknown   Status = "unknown"
)

// Report is the standard health payload for every major component.
type Report struct {
	Component      string     `json:"component"`
	Status         Status     `json:"status"`
	Latency        int64      `json:"latency_ms"`
	Connected      bool       `json:"connected"`
	ReconnectCount int64      `json:"reconnect_count"`
	LastEventTime  *time.Time `json:"last_event_time,omitempty"`
	Message        string     `json:"message,omitempty"`
	Details        map[string]string `json:"details,omitempty"`
}

// Reporter exposes component health for probes and dashboards.
type Reporter interface {
	Health() Report
}
