package opportunity

import "errors"

var (
	ErrNilBus = errors.New("opportunity: nil event bus")
)

const engineName = "opportunity_engine"
