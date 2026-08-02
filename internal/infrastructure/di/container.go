package di

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/adapters/http"
	"github.com/vanam-gangireddy/option-engine/internal/adapters/ws"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/candle"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/performance"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/risk"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/signal"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/strategy"
	"github.com/vanam-gangireddy/option-engine/internal/application/ports"
	"github.com/vanam-gangireddy/option-engine/internal/backtest"
	"github.com/vanam-gangireddy/option-engine/internal/core/calendar"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/core/metrics"
	"github.com/vanam-gangireddy/option-engine/internal/execution/paper"
	"github.com/vanam-gangireddy/option-engine/internal/experiments"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/config"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/logger"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/postgres"
	"github.com/vanam-gangireddy/option-engine/internal/market/cache"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/market/gateway"
	"github.com/vanam-gangireddy/option-engine/internal/market/normalizer"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
	"github.com/vanam-gangireddy/option-engine/internal/market/snapshot"
	"github.com/vanam-gangireddy/option-engine/internal/market/subscription"
	"github.com/vanam-gangireddy/option-engine/internal/market/validator"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
	"github.com/vanam-gangireddy/option-engine/internal/portfolio"
	"github.com/vanam-gangireddy/option-engine/internal/providers"
)

// Container holds all wired dependencies.
type Container struct {
	Config           *config.Config
	Logger           *slog.Logger
	Clock            clock.Clock
	Calendar         *calendar.Calendar
	Metrics          metrics.Registry
	SymbolRegistry   *symbolregistry.Registry
	ProviderRegistry *providers.Registry
	ProviderManager  *providers.Manager
	Cache            *cache.Cache
	EventBus         *eventbus.Bus
	Gateway          *gateway.Engine
	Normalizer       *normalizer.Normalizer
	Validator        *validator.Validator
	Snapshot         func(time.Time) snapshot.Market
	Subscription     *subscription.Manager
	CandleEngine      *candle.Engine
	IndicatorEngine   *indicator.Engine
	SignalEngine      *signal.Engine
	StrategyEngine    *strategy.Engine
	RiskEngine        *risk.Engine
	PaperEngine         *paper.Engine
	PortfolioEngine     *portfolio.Engine
	PerformanceEngine   *performance.Engine
	OptimizationEngine  *optimization.Engine
	ExperimentEngine    *experiments.Engine
	BacktestEngine      *backtest.Engine
	Postgres            *postgres.Pool
	HTTPServer       *http.Server
	WSServer         *ws.Hub
}

// NewContainer wires all application dependencies.
func NewContainer(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Container, error) {
	clk := clock.NewSystem()

	backtestSettings, err := config.BuildBacktestConfig(cfg.Backtest)
	if err != nil {
		return nil, fmt.Errorf("backtest config: %w", err)
	}
	if backtestSettings.Enabled {
		cfg.Market.Provider = "backtest"
	}

	var backtestEngine *backtest.Engine
	if backtestSettings.Enabled {
		backtestEngine, err = backtest.New(backtestSettings, clk)
		if err != nil {
			return nil, fmt.Errorf("backtest engine: %w", err)
		}
		clk = backtestEngine.Clock()
	}

	metricReg := metrics.NewNoopRegistry()

	symbols := symbolregistry.New()
	if cfg.Symbols.File != "" {
		instruments, err := symbolregistry.LoadFromFile(cfg.Symbols.File)
		if err != nil {
			log.Warn("symbol registry load failed; continuing with empty registry",
				"file", cfg.Symbols.File, "error", err)
		} else if err := symbols.Load(instruments); err != nil {
			return nil, fmt.Errorf("symbol registry: %w", err)
		}
	}

	cal, err := calendar.New(toCalendarConfig(cfg), clk)
	if err != nil {
		return nil, fmt.Errorf("calendar: %w", err)
	}

	providerReg := providers.DefaultRegistry()

	manager := providers.NewManager(providerReg, providers.ManagerConfig{
		ActiveProvider: cfg.Market.Provider,
		ProviderCfg:    cfg.ActiveProviderConfig(),
		Reconnect: providers.ReconnectConfig{
			Interval:   cfg.Provider.Reconnect.Interval,
			MaxRetries: cfg.Provider.Reconnect.MaxRetries,
		},
		Subscription: providers.SubscriptionConfig{
			BatchSize: cfg.Subscription.BatchSize,
		},
		Heartbeat: providers.HeartbeatConfig{
			Interval: cfg.Heartbeat.Interval,
		},
	})

	// Inject shared deps into factory config via manager init path
	if err := manager.InitWithDeps(providers.FactoryConfig{
		ProviderCfg: cfg.ActiveProviderConfig(),
		Reconnect: providers.ReconnectConfig{
			Interval:   cfg.Provider.Reconnect.Interval,
			MaxRetries: cfg.Provider.Reconnect.MaxRetries,
		},
		Subscription: providers.SubscriptionConfig{BatchSize: cfg.Subscription.BatchSize},
		Heartbeat:    providers.HeartbeatConfig{Interval: cfg.Heartbeat.Interval},
		Deps: providers.Dependencies{
			Clock:          clk,
			SymbolRegistry: symbols,
			Metrics:        metricReg,
		},
	}); err != nil {
		return nil, fmt.Errorf("provider manager: %w", err)
	}
	if backtestEngine != nil {
		if err := manager.InitWithProvider(backtestEngine.Provider()); err != nil {
			return nil, fmt.Errorf("backtest provider: %w", err)
		}
	}

	provider, err := manager.Provider()
	if err != nil {
		return nil, fmt.Errorf("provider runtime: %w", err)
	}

	cacheStore := cache.New()
	bus := eventbus.New()
	validatorSvc := validator.New(validator.Config{MaxAge: cfg.Validation.MaxTickAge, RequireRegisteredSymbol: true}, symbols)
	normalizerSvc := normalizer.New(clk.Now)
	gatewayEngine := gateway.New(manager.Session(), cacheStore, bus, validatorSvc, normalizerSvc, clk.Now)
	subManager := subscription.New(provider, cfg.Subscription.BatchSize)
	snapshotBuilder := func(at time.Time) snapshot.Market {
		return snapshot.New(cacheStore, at)
	}

	candleSettings, err := cfg.CandleEngineConfig()
	if err != nil {
		return nil, fmt.Errorf("analytics candle config: %w", err)
	}
	candleCfg, err := config.BuildCandleEngineConfig(candleSettings)
	if err != nil {
		return nil, fmt.Errorf("analytics candle config: %w", err)
	}
	candleEngine, err := candle.New(candleCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("analytics candle engine: %w", err)
	}

	indicatorCfg, err := config.BuildIndicatorEngineConfig(cfg.IndicatorEngineSettings())
	if err != nil {
		return nil, fmt.Errorf("analytics indicator config: %w", err)
	}
	indicatorEngine, err := indicator.New(indicatorCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("analytics indicator engine: %w", err)
	}

	signalCfg, err := config.BuildSignalEngineConfig(cfg.SignalEngineSettings())
	if err != nil {
		return nil, fmt.Errorf("analytics signal config: %w", err)
	}
	signalEngine, err := signal.New(signalCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("analytics signal engine: %w", err)
	}

	strategyCfg, err := config.BuildStrategyEngineConfig(cfg.StrategyEngineSettings())
	if err != nil {
		return nil, fmt.Errorf("analytics strategy config: %w", err)
	}
	strategyEngine, err := strategy.New(strategyCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("analytics strategy engine: %w", err)
	}

	riskCfg, err := config.BuildRiskEngineConfig(cfg.RiskEngineSettings())
	if err != nil {
		return nil, fmt.Errorf("analytics risk config: %w", err)
	}
	riskEngine, err := risk.New(riskCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("analytics risk engine: %w", err)
	}

	paperCfg, err := config.BuildPaperExecutionConfig(cfg.PaperExecutionSettings())
	if err != nil {
		return nil, fmt.Errorf("paper execution config: %w", err)
	}
	paperEngine, err := paper.New(paperCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("paper execution engine: %w", err)
	}

	portfolioCfg, err := config.BuildPortfolioEngineConfig(cfg.PortfolioEngineSettings())
	if err != nil {
		return nil, fmt.Errorf("portfolio config: %w", err)
	}
	portfolioEngine, err := portfolio.New(portfolioCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("portfolio engine: %w", err)
	}

	performanceCfg, err := config.BuildPerformanceEngineConfig(cfg.PerformanceEngineSettings())
	if err != nil {
		return nil, fmt.Errorf("performance config: %w", err)
	}
	performanceEngine, err := performance.New(performanceCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("performance engine: %w", err)
	}

	optimizationCfg, err := config.BuildOptimizationEngineConfig(cfg.OptimizationEngineSettings())
	if err != nil {
		return nil, fmt.Errorf("optimization config: %w", err)
	}
	optimizationEngine, err := optimization.New(optimizationCfg, bus, clk)
	if err != nil {
		return nil, fmt.Errorf("optimization engine: %w", err)
	}

	experimentsCfg, err := config.BuildExperimentsEngineConfig(cfg.ExperimentsEngineSettings())
	if err != nil {
		return nil, fmt.Errorf("experiments config: %w", err)
	}
	var experimentRunner experiments.BacktestRunner
	if backtestEngine != nil {
		experimentRunner = experiments.NewSharedEngineRunner(backtestEngine)
	}
	experimentEngine, err := experiments.New(experimentsCfg, bus, clk, experimentRunner)
	if err != nil {
		return nil, fmt.Errorf("experiment engine: %w", err)
	}

	pool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	var healthCheckers []ports.HealthChecker
	healthCheckers = append(healthCheckers, pool)
	providerHealth := &providerHealthAdapter{manager: manager}
	healthCheckers = append(healthCheckers, providerHealth)

	var healthReporters []ports.HealthReporter
	healthReporters = append(healthReporters, providerHealth)
	healthReporters = append(healthReporters, candleEngine)
	healthReporters = append(healthReporters, indicatorEngine)
	healthReporters = append(healthReporters, signalEngine)
	healthReporters = append(healthReporters, strategyEngine)
	healthReporters = append(healthReporters, riskEngine)
	healthReporters = append(healthReporters, paperEngine)
	healthReporters = append(healthReporters, portfolioEngine)
	healthReporters = append(healthReporters, performanceEngine)
	healthReporters = append(healthReporters, optimizationEngine)
	healthReporters = append(healthReporters, experimentEngine)
	if backtestEngine != nil {
		healthReporters = append(healthReporters, backtestEngine)
	}
	healthReporters = append(healthReporters, &postgresHealthAdapter{pool: pool})

	moduleLog := logger.WithModule(log, "bootstrap")

	httpServer := http.NewServer(cfg, moduleLog, healthCheckers, healthReporters)
	wsHub := ws.NewHub(cfg, logger.WithModule(log, "websocket"))

	return &Container{
		Config:           cfg,
		Logger:           log,
		Clock:            clk,
		Calendar:         cal,
		Metrics:          metricReg,
		SymbolRegistry:   symbols,
		ProviderRegistry: providerReg,
		ProviderManager:  manager,
		Cache:            cacheStore,
		EventBus:         bus,
		Gateway:          gatewayEngine,
		Normalizer:       normalizerSvc,
		Validator:        validatorSvc,
		Snapshot:         snapshotBuilder,
		Subscription:     subManager,
		CandleEngine:      candleEngine,
		IndicatorEngine:   indicatorEngine,
		SignalEngine:      signalEngine,
		StrategyEngine:    strategyEngine,
		RiskEngine:        riskEngine,
		PaperEngine:         paperEngine,
		PortfolioEngine:     portfolioEngine,
		PerformanceEngine:   performanceEngine,
		OptimizationEngine:  optimizationEngine,
		ExperimentEngine:    experimentEngine,
		BacktestEngine:      backtestEngine,
		Postgres:            pool,
		HTTPServer:       httpServer,
		WSServer:         wsHub,
	}, nil
}

// StartRuntime connects the provider and starts the market pipeline.
func (c *Container) StartRuntime(ctx context.Context) error {
	if c.ProviderManager != nil {
		if c.Subscription != nil {
			c.ProviderManager.SetSubscriptionManager(c.Subscription)
		}
		if err := c.ProviderManager.Connect(ctx); err != nil {
			return err
		}
	}
	if c.Subscription != nil {
		symbols := c.backtestSymbols()
		if len(symbols) == 0 && c.SymbolRegistry != nil {
			instruments := c.SymbolRegistry.All()
			symbols = make([]string, 0, len(instruments))
			for _, inst := range instruments {
				symbols = append(symbols, inst.Symbol)
			}
		}
		if len(symbols) > 0 {
			if err := c.Subscription.Subscribe(ctx, symbols); err != nil {
				return err
			}
		}
	}
	if c.ExperimentEngine != nil {
		if err := c.ExperimentEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.OptimizationEngine != nil {
		if err := c.OptimizationEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.PerformanceEngine != nil {
		if err := c.PerformanceEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.PortfolioEngine != nil {
		if err := c.PortfolioEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.PaperEngine != nil {
		if err := c.PaperEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.RiskEngine != nil {
		if err := c.RiskEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.StrategyEngine != nil {
		if err := c.StrategyEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.SignalEngine != nil {
		if err := c.SignalEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.IndicatorEngine != nil {
		if err := c.IndicatorEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.CandleEngine != nil {
		if err := c.CandleEngine.Start(ctx); err != nil {
			return err
		}
	}
	if c.Gateway != nil {
		if err := c.Gateway.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Close releases all resources.
func (c *Container) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if c.Gateway != nil {
		_ = c.Gateway.Close()
	}
	if c.CandleEngine != nil {
		_ = c.CandleEngine.Close()
	}
	if c.IndicatorEngine != nil {
		_ = c.IndicatorEngine.Close()
	}
	if c.RiskEngine != nil {
		_ = c.RiskEngine.Close()
	}
	if c.PaperEngine != nil {
		_ = c.PaperEngine.Close()
	}
	if c.PortfolioEngine != nil {
		_ = c.PortfolioEngine.Close()
	}
	if c.PerformanceEngine != nil {
		_ = c.PerformanceEngine.Close()
	}
	if c.OptimizationEngine != nil {
		_ = c.OptimizationEngine.Close()
	}
	if c.ExperimentEngine != nil {
		_ = c.ExperimentEngine.Close()
	}
	if c.StrategyEngine != nil {
		_ = c.StrategyEngine.Close()
	}
	if c.SignalEngine != nil {
		_ = c.SignalEngine.Close()
	}
	if c.ProviderManager != nil {
		_ = c.ProviderManager.Disconnect(ctx)
	}
	if c.BacktestEngine != nil {
		_ = c.BacktestEngine.Close()
	}
	if c.Postgres != nil {
		c.Postgres.Close()
	}
}

func (c *Container) backtestSymbols() []string {
	if c.Config == nil || !c.Config.Backtest.Enabled {
		return nil
	}
	return append([]string(nil), c.Config.Backtest.Symbols...)
}

func toCalendarConfig(cfg *config.Config) calendar.Config {
	c := calendar.Config{
		Timezone:       cfg.Calendar.Timezone,
		RegularOpen:    cfg.Calendar.RegularOpen,
		RegularClose:   cfg.Calendar.RegularClose,
		MuhuratOpen:    cfg.Calendar.MuhuratOpen,
		MuhuratClose:   cfg.Calendar.MuhuratClose,
		EarlyCloseAt:   cfg.Calendar.EarlyCloseAt,
		Holidays:       cfg.Calendar.Holidays,
		MuhuratDays:    cfg.Calendar.MuhuratDays,
		EarlyCloseDays: cfg.Calendar.EarlyCloseDays,
	}
	if cfg.Calendar.ExpiryWeekday >= 0 && cfg.Calendar.ExpiryWeekday <= 6 {
		c.ExpiryWeekday = time.Weekday(cfg.Calendar.ExpiryWeekday)
	}
	return c
}

type providerHealthAdapter struct {
	manager *providers.Manager
}

func (a *providerHealthAdapter) Health() health.Report {
	return a.manager.Health()
}

func (a *providerHealthAdapter) Check(ctx context.Context) error {
	r := a.manager.Health()
	if r.Status == health.StatusUnhealthy {
		return fmt.Errorf("%s", r.Message)
	}
	return nil
}

type postgresHealthAdapter struct {
	pool *postgres.Pool
}

func (a *postgresHealthAdapter) Health() health.Report {
	start := time.Now()
	err := a.pool.Ping(context.Background())
	latency := time.Since(start)

	status := health.StatusHealthy
	msg := "postgres connected"
	if err != nil {
		status = health.StatusUnhealthy
		msg = err.Error()
	}

	return health.Report{
		Component: "postgres",
		Status:    status,
		Latency:   latency.Milliseconds(),
		Connected: err == nil,
		Message:   msg,
	}
}

func (a *postgresHealthAdapter) Check(ctx context.Context) error {
	return a.pool.Check(ctx)
}
