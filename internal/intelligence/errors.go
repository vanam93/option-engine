package intelligence

import "errors"

var (
	ErrNilBus = errors.New("intelligence: nil event bus")
)

const engineName = "recommendation_intelligence_engine"
