package airesearch

import "errors"

var (
	ErrNilBus          = errors.New("airesearch: nil event bus")
	ErrNilStudySource  = errors.New("airesearch: nil study source")
	ErrNilAnalyzer     = errors.New("airesearch: nil analyzer")
	ErrReportNotFound  = errors.New("airesearch: report not found")
	ErrEngineClosed    = errors.New("airesearch: engine closed")
	ErrUnknownAnalyzer = errors.New("airesearch: unknown analyzer")
)

const engineName = "ai_research_engine"
