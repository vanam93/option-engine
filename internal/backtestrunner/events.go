package backtestrunner

import "time"

// SessionStarted is published on backtest.session.started.
type SessionStarted struct {
	BacktestID string        `json:"backtest_id"`
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
	Symbols    []string      `json:"symbols"`
	Expiries   []string      `json:"expiries,omitempty"`
	Mode       RunMode       `json:"mode"`
	StartedAt  time.Time     `json:"started_at"`
}

// SessionCompleted is published on backtest.session.completed.
type SessionCompleted struct {
	BacktestID     string         `json:"backtest_id"`
	Status         SessionStatus  `json:"status"`
	ReplayDuration time.Duration  `json:"replay_duration"`
	Summary        SessionSummary `json:"summary"`
	CompletedAt    time.Time      `json:"completed_at"`
	Error          string         `json:"error,omitempty"`
}
