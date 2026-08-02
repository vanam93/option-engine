package recommendationstate

import "errors"

var (
	ErrNilBus = errors.New("recommendationstate: nil event bus")
)

const engineName = "recommendation_state_engine"
