package validation

import "errors"

var (
	ErrNilBus = errors.New("validation: nil event bus")
)

const engineName = "validation_engine"
