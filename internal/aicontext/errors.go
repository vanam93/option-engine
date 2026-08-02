package aicontext

import "errors"

var (
	ErrNilBus           = errors.New("aicontext: nil event bus")
	ErrNilStudySource   = errors.New("aicontext: nil study source")
	ErrNilReportSource  = errors.New("aicontext: nil report source")
	ErrContextNotFound  = errors.New("aicontext: context not found")
	ErrEngineClosed     = errors.New("aicontext: engine closed")
)

const engineName = "ai_context_engine"
