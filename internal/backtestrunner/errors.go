package backtestrunner

import "errors"

var (
	ErrNilBus             = errors.New("backtestrunner: nil event bus")
	ErrNilRunner          = errors.New("backtestrunner: nil replay runner")
	ErrNilDeliverySource  = errors.New("backtestrunner: nil delivery source")
	ErrSessionNotFound    = errors.New("backtestrunner: session not found")
	ErrConcurrentLimit    = errors.New("backtestrunner: concurrent session limit reached")
	ErrEngineClosed       = errors.New("backtestrunner: engine closed")
	ErrInvalidSession     = errors.New("backtestrunner: invalid session request")
	ErrSessionInProgress  = errors.New("backtestrunner: session already running")
)

const engineName = "backtest_runner"
