package optimization

import "fmt"

const engineName = "optimization_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("optimization: event bus is required")
