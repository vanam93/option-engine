package delivery

import "errors"

var (
	ErrNilBus = errors.New("delivery: nil event bus")
)

const engineName = "recommendation_delivery_engine"
