package scanner

import (
	"time"
)

// Status identifies the outcome of a scanner evaluation.
type Status string

const (
	StatusMatch   Status = "MATCH"
	StatusWatch   Status = "WATCH"
	StatusNeutral Status = "NEUTRAL"
)

// ScannerUpdated is the payload published on scanner.updated events.
type ScannerUpdated struct {
	Symbol       string    `json:"symbol"`
	Timeframe    string    `json:"timeframe"`
	ScannerName  string    `json:"scanner_name"`
	Status       Status    `json:"status"`
	Score        float64   `json:"score"`
	Confidence   float64   `json:"confidence"`
	MatchedRules []string  `json:"matched_rules"`
	Timestamp    time.Time `json:"timestamp"`
}

// InputSignal mirrors the SignalGenerated payload consumed by the scanner.
type InputSignal struct {
	Symbol     string
	Timeframe  string
	Signal     string
	Strategy   string
	Confidence float64
	Timestamp  time.Time
}

// InputDecision mirrors the StrategyDecision payload consumed by the scanner.
type InputDecision struct {
	Symbol     string
	Timeframe  string
	Decision   string
	Strategy   string
	Confidence float64
	Reason     string
	Timestamp  time.Time
}

// InputPerformance mirrors the performance.updated payload consumed by the scanner.
type InputPerformance struct {
	Symbol        string
	Timeframe     string
	Strategy      string
	TotalTrades   int
	WinRate       float64
	RealizedPnL   float64
	UnrealizedPnL float64
	Drawdown      float64
	Timestamp     time.Time
}

// ScanResult is an internal evaluation outcome before publishing.
type ScanResult struct {
	Symbol       string
	Timeframe    string
	ScannerName  string
	Status       Status
	Score        float64
	Confidence   float64
	MatchedRules []string
	Timestamp    time.Time
}

func (r ScanResult) toEvent() ScannerUpdated {
	rules := append([]string(nil), r.MatchedRules...)
	return ScannerUpdated{
		Symbol:       r.Symbol,
		Timeframe:    r.Timeframe,
		ScannerName:  r.ScannerName,
		Status:       r.Status,
		Score:        r.Score,
		Confidence:   r.Confidence,
		MatchedRules: rules,
		Timestamp:    r.Timestamp,
	}
}
