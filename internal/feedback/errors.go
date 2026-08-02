package feedback

import "errors"

var (
	ErrNilBus = errors.New("feedback: nil event bus")
)

const engineName = "feedback_engine"
