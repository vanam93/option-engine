package groww

import "time"

// API status constants.
const (
	apiStatusSuccess = "SUCCESS"
	apiVersion       = "1.0"
	providerName     = "groww"
)

// TokenResponse is returned by the Groww access token endpoint.
type TokenResponse struct {
	Token       string `json:"token"`
	TokenRefID  string `json:"tokenRefId"`
	SessionName string `json:"sessionName"`
	Expiry      string `json:"expiry"`
	IsActive    bool   `json:"isActive"`
}

// APIEnvelope wraps Groww API responses.
type APIEnvelope struct {
	Status  string         `json:"status"`
	Payload map[string]any `json:"payload"`
	Error   *APIError      `json:"error"`
}

// APIError describes a failed Groww API response.
type APIError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Metadata any    `json:"metadata"`
}

// CandlesPayload is the historical candles response payload.
type CandlesPayload struct {
	Candles           [][]any `json:"candles"`
	ClosingPrice      float64 `json:"closing_price"`
	StartTime         string  `json:"start_time"`
	EndTime           string  `json:"end_time"`
	IntervalInMinutes int     `json:"interval_in_minutes"`
}

// ExpiriesPayload is the expiries response payload.
type ExpiriesPayload struct {
	Expiries []string `json:"expiries"`
}

// ContractsPayload is the contracts response payload.
type ContractsPayload struct {
	Contracts []string `json:"contracts"`
}

// CandleRequest identifies a historical candle download.
type CandleRequest struct {
	Exchange       string
	Segment        string
	GrowwSymbol    string
	StartTime      time.Time
	EndTime        time.Time
	CandleInterval string
}

// intervalMaxDuration limits per Groww backtesting API request.
var intervalMaxDuration = map[string]time.Duration{
	"1minute":   30 * 24 * time.Hour,
	"3minute":   30 * 24 * time.Hour,
	"5minute":   30 * 24 * time.Hour,
	"15minute":  90 * 24 * time.Hour,
	"30minute":  90 * 24 * time.Hour,
	"1hour":     180 * 24 * time.Hour,
	"1day":      180 * 24 * time.Hour,
}

// timeframeToGroww maps internal timeframes to Groww candle intervals.
var timeframeToGroww = map[string]string{
	"1m":  "1minute",
	"3m":  "3minute",
	"5m":  "5minute",
	"15m": "15minute",
	"30m": "30minute",
	"1h":  "1hour",
	"1d":  "1day",
}
