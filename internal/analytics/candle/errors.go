package candle

import "errors"

var (
	// ErrNilBus is returned when the event bus dependency is missing.
	ErrNilBus = errors.New("candle: event bus is required")
)
