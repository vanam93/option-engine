package di

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/adapters/http"
	"github.com/vanam-gangireddy/option-engine/internal/adapters/ws"
	"github.com/vanam-gangireddy/option-engine/internal/application/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/calendar"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/core/metrics"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/config"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/logger"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/postgres"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
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
	Postgres         *postgres.Pool
	HTTPServer       *http.Server
	WSServer         *ws.Hub
}

// NewContainer wires all application dependencies.
func NewContainer(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Container, error) {
	clk := clock.NewSystem()
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
		Postgres:         pool,
		HTTPServer:       httpServer,
		WSServer:         wsHub,
	}, nil
}

// Close releases all resources.
func (c *Container) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if c.ProviderManager != nil {
		_ = c.ProviderManager.Disconnect(ctx)
	}
	if c.Postgres != nil {
		c.Postgres.Close()
	}
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
