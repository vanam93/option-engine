package recommendation

import "errors"

var (
	ErrNilBus = errors.New("recommendation: nil event bus")
)

const engineName = "recommendation_engine"
