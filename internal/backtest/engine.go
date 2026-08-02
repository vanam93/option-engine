package backtest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// Engine orchestrates historical candle replay through the market data provider contract.
type Engine struct {
	cfg      Config
	provider *ReplayProvider
	clk      *clock.ReplayClock
	metrics  *replayMetrics

	mu     sync.Mutex
	closed bool
}

// New creates a backtest engine and loads historical candles when a data path is configured.
func New(cfg Config, clk clock.Clock) (*Engine, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	metrics := newReplayMetrics()
	candles, err := loadCandles(cfg)
	if err != nil {
		return nil, err
	}

	replayClk, ok := clk.(*clock.ReplayClock)
	if !ok || replayClk == nil {
		if len(candles) > 0 {
			replayClk = clock.NewReplay(candleTime(candles[0]))
		} else if !cfg.StartTime.IsZero() {
			replayClk = clock.NewReplay(cfg.StartTime)
		} else {
			replayClk = clock.NewReplay(clock.NewSystem().Now())
		}
	}

	provider := NewReplayProvider(replayClk, cfg.Speed, candles, metrics)
	return &Engine{
		cfg:      cfg,
		provider: provider,
		clk:      replayClk,
		metrics:  metrics,
	}, nil
}

// NewWithCandles creates a backtest engine from in-memory historical candles.
func NewWithCandles(cfg Config, candles []market.Candle, clk clock.Clock) (*Engine, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	candles = FilterCandles(candles, LoadOptions{
		Symbols:   cfg.Symbols,
		StartTime: cfg.StartTime,
		EndTime:   cfg.EndTime,
		Timeframe: cfg.Timeframe,
	})

	metrics := newReplayMetrics()
	replayClk, ok := clk.(*clock.ReplayClock)
	if !ok || replayClk == nil {
		if len(candles) > 0 {
			replayClk = clock.NewReplay(candleTime(candles[0]))
		} else {
			replayClk = clock.NewReplay(cfg.StartTime)
		}
	}

	provider := NewReplayProvider(replayClk, cfg.Speed, candles, metrics)
	return &Engine{
		cfg:      cfg,
		provider: provider,
		clk:      replayClk,
		metrics:  metrics,
	}, nil
}

func loadCandles(cfg Config) ([]market.Candle, error) {
	if cfg.DataPath == "" {
		return nil, nil
	}
	return Load(cfg.DataPath, LoadOptions{
		Symbols:   cfg.Symbols,
		StartTime: cfg.StartTime,
		EndTime:   cfg.EndTime,
		Timeframe: cfg.Timeframe,
	})
}

// Provider returns the replay provider for the market pipeline.
func (e *Engine) Provider() api.Provider {
	return e.provider
}

// Clock returns the replay clock driving simulated time.
func (e *Engine) Clock() *clock.ReplayClock {
	return e.clk
}

// Status returns the current replay lifecycle state.
func (e *Engine) Status() ReplayStatus {
	return e.metrics.statusValue()
}

// ProcessedCandles returns the number of replayed candles.
func (e *Engine) ProcessedCandles() uint64 {
	return e.metrics.processed.Load()
}

// Health exposes backtest_engine health for probes.
func (e *Engine) Health() health.Report {
	return e.metrics.report(e.cfg)
}

// Close stops replay and releases background workers.
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	e.mu.Unlock()

	if e.provider == nil {
		return nil
	}
	return e.provider.Disconnect(context.Background())
}

// ProviderConfig maps engine settings into provider factory configuration.
func (e *Engine) ProviderConfig() map[string]any {
	return map[string]any{
		"speed":       e.cfg.Speed,
		"symbols":     append([]string(nil), e.cfg.Symbols...),
		"start_time":  formatTime(e.cfg.StartTime),
		"end_time":    formatTime(e.cfg.EndTime),
		"data_path":   e.cfg.DataPath,
		"timeframe":   string(e.cfg.Timeframe),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ValidateProvider ensures the engine owns a provider instance.
func (e *Engine) ValidateProvider() error {
	if e.provider == nil {
		return fmt.Errorf("%w", ErrNilProvider)
	}
	return nil
}
