package console

import "errors"

var (
	ErrNilBus = errors.New("console: nil event bus")
)

const engineName = "recommendation_console"
