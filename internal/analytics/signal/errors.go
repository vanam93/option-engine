package signal

import "fmt"

const engineName = "signal_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("signal: event bus is required")
