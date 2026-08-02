package groww

import "github.com/vanam-gangireddy/option-engine/internal/domain/market"

// Test exports for internal provider tests.

// GenerateChecksumForTest exposes checksum generation for unit tests.
func GenerateChecksumForTest(secret, timestamp string) string {
	return generateChecksum(secret, timestamp)
}

// NewHealthMetricsForTest constructs metrics for tests.
func NewHealthMetricsForTest() *healthMetrics {
	return newHealthMetrics()
}

// NewClientForTest constructs an HTTP client for tests.
func NewClientForTest(cfg Config, metrics *healthMetrics) *Client {
	return newClient(cfg, metrics)
}

// NewAuthenticatorForTest constructs an authenticator for tests.
func NewAuthenticatorForTest(cfg Config, client *Client) *Authenticator {
	return newAuthenticator(cfg, client)
}

// SetTokenForTest sets a token without calling the auth endpoint.
func (a *Authenticator) SetTokenForTest(token string) {
	a.token = token
}

// NewHistoricalServiceForTest constructs a historical service for tests.
func NewHistoricalServiceForTest(cfg Config, client *Client, auth *Authenticator) *HistoricalService {
	return newHistoricalService(cfg, client, auth)
}

// NewRateLimiterForTest constructs a rate limiter for tests.
func NewRateLimiterForTest(rps float64) *rateLimiter {
	return newRateLimiter(rps)
}

// RetriesForTest returns recorded retry count.
func (m *healthMetrics) RetriesForTest() uint64 {
	return m.retries.Load()
}

// MapGrowwCandlesForTest maps raw Groww candles for unit tests.
func MapGrowwCandlesForTest(raw [][]any, symbol string, tf market.Timeframe) ([]market.Candle, error) {
	return mapGrowwCandles(raw, symbol, tf)
}
