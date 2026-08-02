package research

import "fmt"

const engineName = "research_engine"

// ErrNilBus is returned when the event bus dependency is missing.
var ErrNilBus = fmt.Errorf("research: event bus is required")

// ErrNilRepository is returned when the repository dependency is missing.
var ErrNilRepository = fmt.Errorf("research: repository is required")

// ErrNotFound is returned when a research artifact is not found.
var ErrNotFound = fmt.Errorf("research: not found")
