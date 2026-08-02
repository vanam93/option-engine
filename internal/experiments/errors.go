package experiments

import "fmt"

const engineName = "experiment_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("experiments: event bus is required")

// ErrNilRunner is returned when no backtest runner is configured.
var ErrNilRunner = fmt.Errorf("experiments: backtest runner is required")
