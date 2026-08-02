package alerts

import "errors"

var (
	ErrNilBus = errors.New("alerts: nil event bus")
)

const engineName = "alert_engine"
