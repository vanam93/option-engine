package portfolio

import "fmt"

const engineName = "portfolio_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("portfolio: event bus is required")
