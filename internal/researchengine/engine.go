package researchengine

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// RunResult is the output of a single-strategy research run.
type RunResult struct {
	Strategy   string
	Symbol     string
	Timeframe  string
	Journal    *Journal
	Statistics Statistics
}

// Engine orchestrates strategy simulation over historical candles.
type Engine struct {
	simCfg SimulatorConfig
}

// NewEngine creates a research engine.
func NewEngine(simCfg SimulatorConfig) *Engine {
	return &Engine{simCfg: simCfg.withDefaults()}
}

// RunStrategy simulates one strategy over candles.
func (e *Engine) RunStrategy(strategy strategylib.Strategy, candles []market.Candle) RunResult {
	sim := NewSimulator(e.simCfg)
	journal := sim.Run(strategy, candles)

	symbol := ""
	tf := ""
	if len(candles) > 0 {
		symbol = candles[0].Symbol
		tf = string(candles[0].Timeframe)
	}

	stats := ComputeStatistics(journal, e.simCfg.InitialCapital)
	return RunResult{
		Strategy:   strategy.Name(),
		Symbol:     symbol,
		Timeframe:  tf,
		Journal:    journal,
		Statistics: stats,
	}
}

// RunAll simulates every strategy on the same candle stream.
func (e *Engine) RunAll(strategies []strategylib.Strategy, candles []market.Candle) []RunResult {
	out := make([]RunResult, 0, len(strategies))
	for _, s := range strategies {
		if s == nil {
			continue
		}
		out = append(out, e.RunStrategy(s, candles))
	}
	return out
}
