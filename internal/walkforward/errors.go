package walkforward

import "fmt"

const engineName = "walkforward_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("walkforward: event bus is required")

// ErrNilRunner is returned when no backtest runner is configured.
var ErrNilRunner = fmt.Errorf("walkforward: backtest runner is required")

// ErrInvalidRange is returned when the data range cannot produce windows.
var ErrInvalidRange = fmt.Errorf("walkforward: data range too short for configured windows")
