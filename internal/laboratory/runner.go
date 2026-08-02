package laboratory

import (
	"context"

	"github.com/vanam-gangireddy/option-engine/internal/backtestrunner"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// BacktestRunner executes historical research sessions.
type BacktestRunner interface {
	StartSession(ctx context.Context, req backtestrunner.SessionRequest) (string, error)
}

// SessionSource provides read access to completed backtest sessions.
type SessionSource interface {
	GetSession(id string) (backtestrunner.Session, bool)
}

// StudyRunner delegates study execution to the historical backtest runner.
type StudyRunner struct {
	backtest BacktestRunner
	sessions SessionSource
}

// NewStudyRunner creates a runner that routes studies through the backtest runner.
func NewStudyRunner(backtest BacktestRunner, sessions SessionSource) *StudyRunner {
	return &StudyRunner{
		backtest: backtest,
		sessions: sessions,
	}
}

// Execute runs a study through the historical backtest runner.
func (r *StudyRunner) Execute(ctx context.Context, study Study) (string, backtestrunner.Session, error) {
	req := buildSessionRequest(study)
	sessionID, err := r.backtest.StartSession(ctx, req)
	if err != nil {
		return sessionID, backtestrunner.Session{}, err
	}
	session, ok := r.sessions.GetSession(sessionID)
	if !ok {
		return sessionID, backtestrunner.Session{}, backtestrunner.ErrSessionNotFound
	}
	return sessionID, session, nil
}

func buildSessionRequest(study Study) backtestrunner.SessionRequest {
	timeframe := market.TF1m
	if len(study.Timeframes) > 0 {
		timeframe = study.Timeframes[0]
	}
	return backtestrunner.SessionRequest{
		StartTime: study.StartTime,
		EndTime:   study.EndTime,
		Symbols:   append([]string(nil), study.Symbols...),
		Timeframe: timeframe,
	}
}
