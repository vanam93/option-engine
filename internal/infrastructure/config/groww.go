package config

// GrowwConfig controls the Groww historical data provider.
type GrowwConfig struct {
	Enabled           bool    `mapstructure:"enabled"`
	APIKey            string  `mapstructure:"api_key"`
	APISecret         string  `mapstructure:"api_secret"`
	AccessToken       string  `mapstructure:"access_token"`
	BaseURL           string  `mapstructure:"base_url"`
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	RetryAttempts     int     `mapstructure:"retry_attempts"`
	RetryBackoffMS    int     `mapstructure:"retry_backoff_ms"`
	ReplaySpeed       string  `mapstructure:"replay_speed"`
	CandleInterval    string  `mapstructure:"candle_interval"`
	Timeframe         string  `mapstructure:"timeframe"`
	Exchange          string  `mapstructure:"exchange"`
	Segment           string  `mapstructure:"segment"`
	StartTime         string  `mapstructure:"start_time"`
	EndTime           string  `mapstructure:"end_time"`
}

// ToProviderConfig maps typed Groww settings for the provider factory.
func (c GrowwConfig) ToProviderConfig() map[string]any {
	return map[string]any{
		"enabled":              c.Enabled,
		"api_key":              c.APIKey,
		"api_secret":           c.APISecret,
		"access_token":         c.AccessToken,
		"base_url":             c.BaseURL,
		"requests_per_second":  c.RequestsPerSecond,
		"retry_attempts":       c.RetryAttempts,
		"retry_backoff_ms":     c.RetryBackoffMS,
		"replay_speed":         c.ReplaySpeed,
		"candle_interval":      c.CandleInterval,
		"timeframe":            c.Timeframe,
		"exchange":             c.Exchange,
		"segment":              c.Segment,
		"start_time":           c.StartTime,
		"end_time":             c.EndTime,
	}
}
