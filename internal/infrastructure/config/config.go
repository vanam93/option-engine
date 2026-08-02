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
