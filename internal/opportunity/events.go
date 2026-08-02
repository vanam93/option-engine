package opportunity

import "time"

// Classification identifies the opportunity tier.
type Classification string

const (
	ClassificationBuy    Classification = "BUY"
	ClassificationWatch  Classification = "WATCH"
	ClassificationIgnore Classification = "IGNORE"
)

// OpportunityUpdated is the payload published on opportunity.updated events.
type OpportunityUpdated struct {
	Symbol         string             `json:"symbol"`
	Timeframe      string             `json:"timeframe"`
	Rank           int                `json:"rank"`
	Confidence     float64            `json:"confidence"`
	Classification Classification     `json:"classification"`
	Score          float64            `json:"score"`
	Components     map[string]float64 `json:"components"`
	Timestamp      time.Time          `json:"timestamp"`
}

// InputScanner mirrors the scanner.updated payload.
type InputScanner struct {
	Symbol       string
	Timeframe    string
	ScannerName  string
	Status       string
	Score        float64
	Confidence   float64
	MatchedRules []string
	Timestamp    time.Time
}
