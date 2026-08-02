package backtestrunner

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// SessionStatus describes backtest session lifecycle state.
type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "PENDING"
	SessionStatusRunning   SessionStatus = "RUNNING"
	SessionStatusCompleted SessionStatus = "COMPLETED"
	SessionStatusFailed    SessionStatus = "FAILED"
)

// RunMode identifies how a session interprets its configured period.
type RunMode string

const (
	RunModeSingleDay RunMode = "single_day"
	RunModeDateRange RunMode = "date_range"
	RunModeMultiDay  RunMode = "multi_day"
)

// SessionRequest configures a historical research session.
type SessionRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Symbols   []string
	Expiries  []string
	Mode      RunMode
	Speed     float64
	Timeframe market.Timeframe
	DataPath  string
}

// Session is an immutable historical research session record.
type Session struct {
	BacktestID     string
	StartTime      time.Time
	EndTime        time.Time
	ReplayDuration time.Duration
	Status         SessionStatus
	CreatedAt      time.Time
	CompletedAt    *time.Time
	Request        SessionRequest
	Summary        *SessionSummary
	Error          string
}

// Validate checks session request fields.
func (r SessionRequest) Validate() error {
	if r.StartTime.IsZero() {
		return fmt.Errorf("%w: start_time required", ErrInvalidSession)
	}
	if r.EndTime.IsZero() {
		return fmt.Errorf("%w: end_time required", ErrInvalidSession)
	}
	if r.EndTime.Before(r.StartTime) {
		return fmt.Errorf("%w: end_time must be after start_time", ErrInvalidSession)
	}
	if len(r.Symbols) == 0 {
		return fmt.Errorf("%w: symbols required", ErrInvalidSession)
	}
	return nil
}

func (r SessionRequest) withDefaults() SessionRequest {
	out := r
	if out.Mode == "" {
		if sameCalendarDay(out.StartTime, out.EndTime) {
			out.Mode = RunModeSingleDay
		} else if spansMultipleDays(out.StartTime, out.EndTime) {
			out.Mode = RunModeMultiDay
		} else {
			out.Mode = RunModeDateRange
		}
	}
	if out.Speed <= 0 {
		out.Speed = 1.0
	}
	if out.Timeframe == "" {
		out.Timeframe = market.TF1m
	}
	out.Symbols = append([]string(nil), out.Symbols...)
	out.Expiries = append([]string(nil), out.Expiries...)
	return out
}

func newSession(req SessionRequest, at time.Time) Session {
	req = req.withDefaults()
	return Session{
		BacktestID: generateBacktestID(at),
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Status:     SessionStatusPending,
		CreatedAt:  at,
		Request:    req,
	}
}

func generateBacktestID(at time.Time) string {
	return fmt.Sprintf("BT-%s-%s", at.UTC().Format("20060102T150405"), uuid.NewString()[:8])
}

func sameCalendarDay(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func spansMultipleDays(start, end time.Time) bool {
	return !sameCalendarDay(start, end)
}
