package scanner

import "errors"

var (
	ErrNilBus = errors.New("scanner: nil event bus")
)

const engineName = "scanner_engine"
