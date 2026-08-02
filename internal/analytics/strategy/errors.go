package strategy

import "fmt"

const engineName = "strategy_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("strategy: event bus is required")
