package quality

import "errors"

var (
	ErrNilBus = errors.New("quality: nil event bus")
)

const engineName = "recommendation_quality_engine"
