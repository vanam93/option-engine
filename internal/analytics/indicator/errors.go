package indicator

import "fmt"

const engineName = "indicator_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("indicator: event bus is required")
