package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Env          string             `mapstructure:"env"`
	HTTP         HTTPConfig         `mapstructure:"http"`
	WebSocket    WSConfig           `mapstructure:"websocket"`
	Postgres     PostgresConfig     `mapstructure:"postgres"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	Shutdown     ShutdownConfig     `mapstructure:"shutdown"`
	Market       MarketConfig       `mapstructure:"market"`
	Provider     ProviderConfig     `mapstructure:"provider"`
	Heartbeat    HeartbeatConfig    `mapstructure:"heartbeat"`
	Subscription SubscriptionConfig `mapstructure:"subscription"`
	EventBus     EventBusConfig     `mapstructure:"event_bus"`
	Validation   ValidationConfig   `mapstructure:"validation"`
	Calendar     CalendarConfig     `mapstructure:"calendar"`
	Symbols      SymbolsConfig      `mapstructure:"symbols"`
	Analytics    AnalyticsConfig    `mapstructure:"analytics"`
}

type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type WSConfig struct {
	Path            string `mapstructure:"path"`
	ReadBufferSize  int    `mapstructure:"read_buffer_size"`
	WriteBufferSize int    `mapstructure:"write_buffer_size"`
}

type PostgresConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json | text
}

type ShutdownConfig struct {
	Timeout time.Duration `mapstructure:"timeout"`
}

// MarketConfig selects the active market data provider.
type MarketConfig struct {
	Provider string         `mapstructure:"provider"`
	Mock     map[string]any `mapstructure:"mock"`
	Replay   map[string]any `mapstructure:"replay"`
}

// ProviderConfig holds shared provider runtime settings.
type ProviderConfig struct {
	Reconnect ReconnectConfig `mapstructure:"reconnect"`
}

type ReconnectConfig struct {
	Interval   string `mapstructure:"interval"`
	MaxRetries int    `mapstructure:"max_retries"` // -1 = unlimited
}

type HeartbeatConfig struct {
	Interval string `mapstructure:"interval"`
}

type SubscriptionConfig struct {
	BatchSize int `mapstructure:"batch_size"`
}
type EventBusConfig struct {
	SubscriberBuffer int `mapstructure:"subscriber_buffer"`
}
type ValidationConfig struct {
	MaxTickAge time.Duration `mapstructure:"max_tick_age"`
}

// CalendarConfig drives the NSE market calendar.
type CalendarConfig struct {
	Timezone       string   `mapstructure:"timezone"`
	RegularOpen    string   `mapstructure:"regular_open"`
	RegularClose   string   `mapstructure:"regular_close"`
	MuhuratOpen    string   `mapstructure:"muhurat_open"`
	MuhuratClose   string   `mapstructure:"muhurat_close"`
	EarlyCloseAt   string   `mapstructure:"early_close_at"`
	Holidays       []string `mapstructure:"holidays"`
	MuhuratDays    []string `mapstructure:"muhurat_days"`
	EarlyCloseDays []string `mapstructure:"early_close_days"`
	ExpiryWeekday  int      `mapstructure:"expiry_weekday"`
}

// SymbolsConfig points to the instrument registry file.
type SymbolsConfig struct {
	File string `mapstructure:"file"`
}

// AnalyticsConfig groups Stage 3 analytics engine settings.
type AnalyticsConfig struct {
	Candle    CandleAnalyticsConfig    `mapstructure:"candle"`
	Indicator IndicatorAnalyticsConfig `mapstructure:"indicator"`
}

// IndicatorAnalyticsConfig controls the indicator computation engine.
type IndicatorAnalyticsConfig struct {
	Enabled          bool                   `mapstructure:"enabled"`
	SubscriberBuffer int                    `mapstructure:"subscriber_buffer"`
	EMA              []IndicatorPeriodConfig `mapstructure:"ema"`
	SMA              []IndicatorPeriodConfig `mapstructure:"sma"`
	RSI              []IndicatorPeriodConfig `mapstructure:"rsi"`
	ATR              []IndicatorPeriodConfig `mapstructure:"atr"`
}

// IndicatorPeriodConfig is a lookback period for an indicator.
type IndicatorPeriodConfig struct {
	Period int `mapstructure:"period"`
}

// CandleAnalyticsConfig controls the candle aggregation engine.
type CandleAnalyticsConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	Timeframes       []string `mapstructure:"timeframes"`
	Timezone         string   `mapstructure:"timezone"`
	SubscriberBuffer int      `mapstructure:"subscriber_buffer"`
	FlushOnShutdown  bool     `mapstructure:"flush_on_shutdown"`
	VolumeMode       string   `mapstructure:"volume_mode"`
	OrderPolicy      string   `mapstructure:"order_policy"`
	IdleEvictAfter   string   `mapstructure:"idle_evict_after"`
}

// Load reads configuration from file and environment variables.
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("OE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("env", "development")
	v.SetDefault("http.host", "0.0.0.0")
	v.SetDefault("http.port", 8080)
	v.SetDefault("websocket.path", "/ws")
	v.SetDefault("websocket.read_buffer_size", 1024)
	v.SetDefault("websocket.write_buffer_size", 1024)
	v.SetDefault("postgres.host", "localhost")
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.user", "option_engine")
	v.SetDefault("postgres.password", "option_engine")
	v.SetDefault("postgres.database", "option_engine")
	v.SetDefault("postgres.ssl_mode", "disable")
	v.SetDefault("postgres.max_conns", 10)
	v.SetDefault("postgres.min_conns", 2)
	v.SetDefault("postgres.max_conn_lifetime", "30m")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("shutdown.timeout", "15s")

	v.SetDefault("market.provider", "mock")
	v.SetDefault("provider.reconnect.interval", "5s")
	v.SetDefault("provider.reconnect.max_retries", -1)
	v.SetDefault("heartbeat.interval", "10s")
	v.SetDefault("subscription.batch_size", 200)
	v.SetDefault("event_bus.subscriber_buffer", 1024)
	v.SetDefault("validation.max_tick_age", "30s")

	v.SetDefault("calendar.timezone", "Asia/Kolkata")
	v.SetDefault("calendar.regular_open", "09:15")
	v.SetDefault("calendar.regular_close", "15:30")
	v.SetDefault("calendar.muhurat_open", "18:00")
	v.SetDefault("calendar.muhurat_close", "19:15")
	v.SetDefault("calendar.early_close_at", "13:00")
	v.SetDefault("calendar.expiry_weekday", 4)

	v.SetDefault("symbols.file", "configs/symbols.yaml")

	v.SetDefault("analytics.candle.enabled", true)
	v.SetDefault("analytics.candle.timeframes", []string{"1m", "5m"})
	v.SetDefault("analytics.candle.timezone", "Asia/Kolkata")
	v.SetDefault("analytics.candle.subscriber_buffer", 256)
	v.SetDefault("analytics.candle.flush_on_shutdown", true)
	v.SetDefault("analytics.candle.volume_mode", "cumulative")
	v.SetDefault("analytics.candle.order_policy", "reject_older")
	v.SetDefault("analytics.candle.idle_evict_after", "0s")

	v.SetDefault("analytics.indicator.enabled", true)
	v.SetDefault("analytics.indicator.subscriber_buffer", 256)
	v.SetDefault("analytics.indicator.ema", []map[string]any{{"period": 9}})
	v.SetDefault("analytics.indicator.sma", []map[string]any{{"period": 20}})
	v.SetDefault("analytics.indicator.rsi", []map[string]any{{"period": 14}})
	v.SetDefault("analytics.indicator.atr", []map[string]any{{"period": 14}})
}

// HTTPAddr returns the bind address for the HTTP server.
func (c *Config) HTTPAddr() string {
	return fmt.Sprintf("%s:%d", c.HTTP.Host, c.HTTP.Port)
}

// PostgresDSN returns a PostgreSQL connection string.
func (c *Config) PostgresDSN() string {
	p := c.Postgres
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode,
	)
}

// ActiveProviderConfig returns provider-specific settings for the selected provider.
func (c *Config) ActiveProviderConfig() map[string]any {
	switch c.Market.Provider {
	case "replay":
		if c.Market.Replay != nil {
			return c.Market.Replay
		}
	case "mock":
		if c.Market.Mock != nil {
			return c.Market.Mock
		}
	}
	return map[string]any{}
}

// ToCalendarConfig converts to core calendar config.
func (c *Config) ToCalendarConfig() CalendarConfig {
	return c.Calendar
}

// CandleEngineConfig maps analytics candle settings into the candle engine config.
func (c *Config) CandleEngineConfig() (CandleEngineConfig, error) {
	cfg := c.Analytics.Candle
	idleEvict, err := time.ParseDuration(cfg.IdleEvictAfter)
	if err != nil && cfg.IdleEvictAfter != "" {
		return CandleEngineConfig{}, fmt.Errorf("analytics.candle.idle_evict_after: %w", err)
	}
	return CandleEngineConfig{
		Enabled:          cfg.Enabled,
		Timezone:         cfg.Timezone,
		SubscriberBuffer: cfg.SubscriberBuffer,
		FlushOnShutdown:  cfg.FlushOnShutdown,
		VolumeMode:       cfg.VolumeMode,
		OrderPolicy:      cfg.OrderPolicy,
		IdleEvictAfter:   idleEvict,
		Timeframes:       append([]string(nil), cfg.Timeframes...),
	}, nil
}

// IndicatorEngineConfig is the validated indicator configuration used by DI wiring.
type IndicatorEngineConfig struct {
	Enabled          bool
	SubscriberBuffer int
	EMA              []IndicatorPeriodConfig
	SMA              []IndicatorPeriodConfig
	RSI              []IndicatorPeriodConfig
	ATR              []IndicatorPeriodConfig
}

// IndicatorEngineSettings maps analytics indicator settings.
func (c *Config) IndicatorEngineSettings() IndicatorEngineConfig {
	return IndicatorEngineConfig{
		Enabled:          c.Analytics.Indicator.Enabled,
		SubscriberBuffer: c.Analytics.Indicator.SubscriberBuffer,
		EMA:              append([]IndicatorPeriodConfig(nil), c.Analytics.Indicator.EMA...),
		SMA:              append([]IndicatorPeriodConfig(nil), c.Analytics.Indicator.SMA...),
		RSI:              append([]IndicatorPeriodConfig(nil), c.Analytics.Indicator.RSI...),
		ATR:              append([]IndicatorPeriodConfig(nil), c.Analytics.Indicator.ATR...),
	}
}

// CandleEngineConfig is the validated candle configuration used by DI wiring.
type CandleEngineConfig struct {
	Enabled          bool
	Timeframes       []string
	Timezone         string
	SubscriberBuffer int
	FlushOnShutdown  bool
	VolumeMode       string
	OrderPolicy      string
	IdleEvictAfter   time.Duration
}
