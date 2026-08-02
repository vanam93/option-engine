package paper

import "fmt"

const engineName = "paper_execution_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("paper execution: event bus is required")
