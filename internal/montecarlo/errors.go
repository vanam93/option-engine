package montecarlo

import "fmt"

const engineName = "montecarlo_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("montecarlo: event bus is required")

// ErrNoTrades is returned when walk-forward metrics cannot produce trade samples.
var ErrNoTrades = fmt.Errorf("montecarlo: no trade returns available")
