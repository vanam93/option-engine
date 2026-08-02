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
	Execution    ExecutionConfig    `mapstructure:"execution"`
	Portfolio    PortfolioConfig    `mapstructure:"portfolio"`
	Backtest     BacktestConfig     `mapstructure:"backtest"`
	Optimization OptimizationConfig `mapstructure:"optimization"`
	Experiments  ExperimentsConfig  `mapstructure:"experiments"`
	WalkForward  WalkForwardConfig  `mapstructure:"walkforward"`
	MonteCarlo   MonteCarloConfig   `mapstructure:"montecarlo"`
	Research     ResearchConfig     `mapstructure:"research"`
	Scanner      ScannerConfig      `mapstructure:"scanner"`
	Intelligence IntelligenceConfig `mapstructure:"intelligence"`
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
	Candle      CandleAnalyticsConfig      `mapstructure:"candle"`
	Indicator   IndicatorAnalyticsConfig   `mapstructure:"indicator"`
	Signal      SignalAnalyticsConfig      `mapstructure:"signal"`
	Strategy    StrategyAnalyticsConfig    `mapstructure:"strategy"`
	Risk        RiskAnalyticsConfig        `mapstructure:"risk"`
	Performance PerformanceAnalyticsConfig `mapstructure:"performance"`
}

// SignalAnalyticsConfig controls the signal evaluation engine.
type SignalAnalyticsConfig struct {
	Enabled          bool                     `mapstructure:"enabled"`
	SubscriberBuffer int                      `mapstructure:"subscriber_buffer"`
	EMACross         EMACrossAnalyticsConfig  `mapstructure:"ema_cross"`
	MACDCross        MACDCrossAnalyticsConfig `mapstructure:"macd_cross"`
	RSI              RSISignalAnalyticsConfig `mapstructure:"rsi"`
	Bollinger        BollingerSignalConfig    `mapstructure:"bollinger"`
}

// EMACrossAnalyticsConfig configures the EMA crossover rule.
type EMACrossAnalyticsConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	FastPeriod int  `mapstructure:"fast_period"`
	SlowPeriod int  `mapstructure:"slow_period"`
}

// MACDCrossAnalyticsConfig configures the MACD crossover rule.
type MACDCrossAnalyticsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// RSISignalAnalyticsConfig configures RSI threshold signals.
type RSISignalAnalyticsConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Oversold   float64 `mapstructure:"oversold"`
	Overbought float64 `mapstructure:"overbought"`
}

// BollingerSignalConfig configures Bollinger band signals.
type BollingerSignalConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// StrategyAnalyticsConfig controls the strategy decision engine.
type StrategyAnalyticsConfig struct {
	Enabled          bool                          `mapstructure:"enabled"`
	SubscriberBuffer int                           `mapstructure:"subscriber_buffer"`
	MinConfidence    float64                       `mapstructure:"min_confidence"`
	TrendFollowing   TrendFollowingAnalyticsConfig `mapstructure:"trend_following"`
	MeanReversion    MeanReversionAnalyticsConfig  `mapstructure:"mean_reversion"`
	Breakout         BreakoutAnalyticsConfig       `mapstructure:"breakout"`
}

// TrendFollowingAnalyticsConfig enables the trend-following strategy.
type TrendFollowingAnalyticsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// MeanReversionAnalyticsConfig enables the mean-reversion strategy.
type MeanReversionAnalyticsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// BreakoutAnalyticsConfig enables the breakout strategy placeholder.
type BreakoutAnalyticsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// ExecutionConfig groups execution engine settings.
type ExecutionConfig struct {
	Paper PaperExecutionConfig `mapstructure:"paper"`
}

// PortfolioConfig controls the portfolio and PnL engine.
type PortfolioConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	SubscriberBuffer int  `mapstructure:"subscriber_buffer"`
}

// PaperExecutionConfig controls the paper execution engine.
type PaperExecutionConfig struct {
	Enabled          bool    `mapstructure:"enabled"`
	SubscriberBuffer int     `mapstructure:"subscriber_buffer"`
	SlippagePercent  float64 `mapstructure:"slippage_percent"`
	DefaultPrice     string  `mapstructure:"default_price"`
}

// RiskAnalyticsConfig controls the decision and risk engine.
type RiskAnalyticsConfig struct {
	Enabled          bool    `mapstructure:"enabled"`
	SubscriberBuffer int     `mapstructure:"subscriber_buffer"`
	MinConfidence    float64 `mapstructure:"min_confidence"`
	MaxPositions     int     `mapstructure:"max_positions"`
	MaxTradesPerDay  int     `mapstructure:"max_trades_per_day"`
	DefaultQuantity  int     `mapstructure:"default_quantity"`
}

// PerformanceAnalyticsConfig controls the performance analytics engine.
type PerformanceAnalyticsConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	SubscriberBuffer int  `mapstructure:"subscriber_buffer"`
}

// IndicatorAnalyticsConfig controls the indicator computation engine.
type IndicatorAnalyticsConfig struct {
	Enabled          bool                     `mapstructure:"enabled"`
	SubscriberBuffer int                      `mapstructure:"subscriber_buffer"`
	EMA              []IndicatorPeriodConfig  `mapstructure:"ema"`
	SMA              []IndicatorPeriodConfig  `mapstructure:"sma"`
	RSI              []IndicatorPeriodConfig  `mapstructure:"rsi"`
	ATR              []IndicatorPeriodConfig  `mapstructure:"atr"`
	MACD             MACDAnalyticsConfig      `mapstructure:"macd"`
	Bollinger        BollingerAnalyticsConfig `mapstructure:"bollinger"`
}

// MACDAnalyticsConfig configures MACD periods.
type MACDAnalyticsConfig struct {
	FastPeriod   int `mapstructure:"fast_period"`
	SlowPeriod   int `mapstructure:"slow_period"`
	SignalPeriod int `mapstructure:"signal_period"`
}

// BollingerAnalyticsConfig configures Bollinger Bands.
type BollingerAnalyticsConfig struct {
	Period int     `mapstructure:"period"`
	StdDev float64 `mapstructure:"stddev"`
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
	v.SetDefault("analytics.indicator.macd.fast_period", 12)
	v.SetDefault("analytics.indicator.macd.slow_period", 26)
	v.SetDefault("analytics.indicator.macd.signal_period", 9)
	v.SetDefault("analytics.indicator.bollinger.period", 20)
	v.SetDefault("analytics.indicator.bollinger.stddev", 2.0)

	v.SetDefault("analytics.signal.enabled", true)
	v.SetDefault("analytics.signal.subscriber_buffer", 256)
	v.SetDefault("analytics.signal.ema_cross.enabled", true)
	v.SetDefault("analytics.signal.ema_cross.fast_period", 9)
	v.SetDefault("analytics.signal.ema_cross.slow_period", 21)
	v.SetDefault("analytics.signal.macd_cross.enabled", true)
	v.SetDefault("analytics.signal.rsi.enabled", true)
	v.SetDefault("analytics.signal.rsi.oversold", 30)
	v.SetDefault("analytics.signal.rsi.overbought", 70)
	v.SetDefault("analytics.signal.bollinger.enabled", true)

	v.SetDefault("analytics.strategy.enabled", true)
	v.SetDefault("analytics.strategy.subscriber_buffer", 256)
	v.SetDefault("analytics.strategy.min_confidence", 0.5)
	v.SetDefault("analytics.strategy.trend_following.enabled", true)
	v.SetDefault("analytics.strategy.mean_reversion.enabled", true)
	v.SetDefault("analytics.strategy.breakout.enabled", false)

	v.SetDefault("analytics.risk.enabled", true)
	v.SetDefault("analytics.risk.subscriber_buffer", 256)
	v.SetDefault("analytics.risk.min_confidence", 0.70)
	v.SetDefault("analytics.risk.max_positions", 5)
	v.SetDefault("analytics.risk.max_trades_per_day", 20)
	v.SetDefault("analytics.risk.default_quantity", 1)

	v.SetDefault("analytics.performance.enabled", true)
	v.SetDefault("analytics.performance.subscriber_buffer", 256)

	v.SetDefault("execution.paper.enabled", true)
	v.SetDefault("execution.paper.subscriber_buffer", 256)
	v.SetDefault("execution.paper.slippage_percent", 0.05)
	v.SetDefault("execution.paper.default_price", "market")

	v.SetDefault("portfolio.enabled", true)
	v.SetDefault("portfolio.subscriber_buffer", 256)

	v.SetDefault("backtest.enabled", false)
	v.SetDefault("backtest.speed", "1x")
	v.SetDefault("backtest.symbols", []string{"NIFTY"})
	v.SetDefault("backtest.timeframe", "1m")

	v.SetDefault("optimization.enabled", true)
	v.SetDefault("optimization.subscriber_buffer", 256)
	v.SetDefault("optimization.scoring.profit_factor_weight", 0.40)
	v.SetDefault("optimization.scoring.win_rate_weight", 0.30)
	v.SetDefault("optimization.scoring.expectancy_weight", 0.20)
	v.SetDefault("optimization.scoring.drawdown_penalty", 0.10)

	v.SetDefault("experiments.enabled", true)
	v.SetDefault("experiments.parallel_workers", 4)
	v.SetDefault("experiments.max_concurrent_runs", 4)
	v.SetDefault("experiments.subscriber_buffer", 256)
	v.SetDefault("experiments.symbols", []string{"NIFTY"})
	v.SetDefault("experiments.timeframes", []string{"5m"})
	v.SetDefault("experiments.strategy", "trend_following")
	v.SetDefault("experiments.parameter_ranges.ema_fast", []int{5, 9, 12})
	v.SetDefault("experiments.parameter_ranges.ema_slow", []int{21, 34, 50})
	v.SetDefault("experiments.parameter_ranges.rsi_period", []int{14, 21})

	v.SetDefault("walkforward.enabled", false)
	v.SetDefault("walkforward.train_window_days", 30)
	v.SetDefault("walkforward.validation_window_days", 10)
	v.SetDefault("walkforward.step_days", 10)
	v.SetDefault("walkforward.subscriber_buffer", 256)
	v.SetDefault("walkforward.parallel_workers", 2)
	v.SetDefault("walkforward.max_concurrent_runs", 2)

	v.SetDefault("montecarlo.enabled", true)
	v.SetDefault("montecarlo.simulations", 1000)
	v.SetDefault("montecarlo.confidence_level", 0.95)
	seed := int64(42)
	v.SetDefault("montecarlo.random_seed", seed)
	v.SetDefault("montecarlo.subscriber_buffer", 256)
	v.SetDefault("montecarlo.ruin_drawdown_pct", 1.0)

	v.SetDefault("research.enabled", true)
	v.SetDefault("research.export_directory", "./reports")
	v.SetDefault("research.formats", []string{"json", "csv"})
	v.SetDefault("research.subscriber_buffer", 256)

	v.SetDefault("scanner.enabled", true)
	v.SetDefault("scanner.symbols", []string{"NIFTY"})
	v.SetDefault("scanner.subscriber_buffer", 256)
	v.SetDefault("scanner.min_confidence", 0.5)
	v.SetDefault("scanner.scanners.ema", true)
	v.SetDefault("scanner.scanners.rsi", true)
	v.SetDefault("scanner.scanners.macd", true)
	v.SetDefault("scanner.scanners.trend", true)
	v.SetDefault("scanner.scanners.ranking", true)

	v.SetDefault("intelligence.opportunity.enabled", true)
	v.SetDefault("intelligence.opportunity.top_n", 20)
	v.SetDefault("intelligence.opportunity.subscriber_buffer", 256)
	v.SetDefault("intelligence.opportunity.buy_threshold", 0.70)
	v.SetDefault("intelligence.opportunity.watch_threshold", 0.40)
	v.SetDefault("intelligence.opportunity.weights.signal", 0.20)
	v.SetDefault("intelligence.opportunity.weights.strategy", 0.20)
	v.SetDefault("intelligence.opportunity.weights.performance", 0.15)
	v.SetDefault("intelligence.opportunity.weights.optimization", 0.15)
	v.SetDefault("intelligence.opportunity.weights.walkforward", 0.15)
	v.SetDefault("intelligence.opportunity.weights.montecarlo", 0.15)
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
	case "backtest":
		return c.Backtest.ToProviderConfig()
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
	MACD             MACDAnalyticsConfig
	Bollinger        BollingerAnalyticsConfig
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
		MACD:             c.Analytics.Indicator.MACD,
		Bollinger:        c.Analytics.Indicator.Bollinger,
	}
}

// SignalEngineConfig is the validated signal configuration used by DI wiring.
type SignalEngineConfig struct {
	Enabled          bool
	SubscriberBuffer int
	EMACross         EMACrossAnalyticsConfig
	MACDCross        MACDCrossAnalyticsConfig
	RSI              RSISignalAnalyticsConfig
	Bollinger        BollingerSignalConfig
}

// SignalEngineSettings maps analytics signal settings.
func (c *Config) SignalEngineSettings() SignalEngineConfig {
	return SignalEngineConfig{
		Enabled:          c.Analytics.Signal.Enabled,
		SubscriberBuffer: c.Analytics.Signal.SubscriberBuffer,
		EMACross:         c.Analytics.Signal.EMACross,
		MACDCross:        c.Analytics.Signal.MACDCross,
		RSI:              c.Analytics.Signal.RSI,
		Bollinger:        c.Analytics.Signal.Bollinger,
	}
}

// StrategyEngineConfig is the validated strategy configuration used by DI wiring.
type StrategyEngineConfig struct {
	Enabled          bool
	SubscriberBuffer int
	MinConfidence    float64
	TrendFollowing   TrendFollowingAnalyticsConfig
	MeanReversion    MeanReversionAnalyticsConfig
	Breakout         BreakoutAnalyticsConfig
}

// StrategyEngineSettings maps analytics strategy settings.
func (c *Config) StrategyEngineSettings() StrategyEngineConfig {
	return StrategyEngineConfig{
		Enabled:          c.Analytics.Strategy.Enabled,
		SubscriberBuffer: c.Analytics.Strategy.SubscriberBuffer,
		MinConfidence:    c.Analytics.Strategy.MinConfidence,
		TrendFollowing:   c.Analytics.Strategy.TrendFollowing,
		MeanReversion:    c.Analytics.Strategy.MeanReversion,
		Breakout:         c.Analytics.Strategy.Breakout,
	}
}

// RiskEngineConfig is the validated risk configuration used by DI wiring.
type RiskEngineConfig struct {
	Enabled          bool
	SubscriberBuffer int
	MinConfidence    float64
	MaxPositions     int
	MaxTradesPerDay  int
	DefaultQuantity  int
	DayResetTimezone string
}

// RiskEngineSettings maps analytics risk settings.
func (c *Config) RiskEngineSettings() RiskEngineConfig {
	return RiskEngineConfig{
		Enabled:          c.Analytics.Risk.Enabled,
		SubscriberBuffer: c.Analytics.Risk.SubscriberBuffer,
		MinConfidence:    c.Analytics.Risk.MinConfidence,
		MaxPositions:     c.Analytics.Risk.MaxPositions,
		MaxTradesPerDay:  c.Analytics.Risk.MaxTradesPerDay,
		DefaultQuantity:  c.Analytics.Risk.DefaultQuantity,
		DayResetTimezone: c.Calendar.Timezone,
	}
}

// PaperExecutionEngineConfig is the validated paper execution configuration used by DI wiring.
type PaperExecutionEngineConfig struct {
	Enabled          bool
	SubscriberBuffer int
	SlippagePercent  float64
	DefaultPrice     string
}

// PaperExecutionSettings maps execution paper settings.
func (c *Config) PaperExecutionSettings() PaperExecutionEngineConfig {
	return PaperExecutionEngineConfig{
		Enabled:          c.Execution.Paper.Enabled,
		SubscriberBuffer: c.Execution.Paper.SubscriberBuffer,
		SlippagePercent:  c.Execution.Paper.SlippagePercent,
		DefaultPrice:     c.Execution.Paper.DefaultPrice,
	}
}

// PortfolioEngineConfig is the validated portfolio configuration used by DI wiring.
type PortfolioEngineConfig struct {
	Enabled          bool
	SubscriberBuffer int
}

// PortfolioEngineSettings maps portfolio settings.
func (c *Config) PortfolioEngineSettings() PortfolioEngineConfig {
	return PortfolioEngineConfig{
		Enabled:          c.Portfolio.Enabled,
		SubscriberBuffer: c.Portfolio.SubscriberBuffer,
	}
}

// PerformanceEngineConfig is the validated performance analytics configuration used by DI wiring.
type PerformanceEngineConfig struct {
	Enabled          bool
	SubscriberBuffer int
}

// PerformanceEngineSettings maps analytics performance settings.
func (c *Config) PerformanceEngineSettings() PerformanceEngineConfig {
	return PerformanceEngineConfig{
		Enabled:          c.Analytics.Performance.Enabled,
		SubscriberBuffer: c.Analytics.Performance.SubscriberBuffer,
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
