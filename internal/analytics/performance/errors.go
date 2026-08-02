package performance

import "fmt"

const engineName = "performance_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("performance: event bus is required")
