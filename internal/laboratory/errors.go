package laboratory

import "errors"

var (
	ErrNilBus             = errors.New("laboratory: nil event bus")
	ErrNilBacktestRunner  = errors.New("laboratory: nil backtest runner")
	ErrNilSessionSource   = errors.New("laboratory: nil session source")
	ErrStudyNotFound      = errors.New("laboratory: study not found")
	ErrComparisonNotFound = errors.New("laboratory: comparison not found")
	ErrConcurrentLimit    = errors.New("laboratory: concurrent study limit reached")
	ErrEngineClosed       = errors.New("laboratory: engine closed")
	ErrInvalidStudy       = errors.New("laboratory: invalid study request")
	ErrStudyInProgress    = errors.New("laboratory: study already running")
	ErrStudyNotCompleted  = errors.New("laboratory: study not completed")
)

const engineName = "strategy_laboratory"
