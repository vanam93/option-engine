package risk

import "fmt"

const engineName = "risk_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("risk: event bus is required")
