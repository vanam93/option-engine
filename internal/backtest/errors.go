package backtest

import "errors"

var (
	ErrNilProvider = errors.New("backtest: nil provider")
	ErrNotStarted  = errors.New("backtest: engine not started")
)
