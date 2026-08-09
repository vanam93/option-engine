package researchengine

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// ChampionshipEngine runs every strategy on one shared candle stream and ranks results.
type ChampionshipEngine struct {
	engine  *Engine
	weights RankingWeights
}

// NewChampionshipEngine creates a championship runner.
func NewChampionshipEngine(simCfg SimulatorConfig, weights RankingWeights) *ChampionshipEngine {
	return &ChampionshipEngine{
		engine:  NewEngine(simCfg),
		weights: weights.withDefaults(),
	}
}

// Run executes all strategies on identical candles and returns a ranked leaderboard.
func (c *ChampionshipEngine) Run(strategies []strategylib.Strategy, candles []market.Candle, meta ChampionshipMeta) StrategyLeaderboard {
	dataset := NewDataset(candles)
	reports := make([]PerformanceReport, 0, len(strategies))
	at := time.Now().UTC()

	for _, strategy := range strategies {
		if strategy == nil {
			continue
		}
		result := c.engine.RunStrategyOnDataset(strategy, dataset)
		metaDesc := strategy.Metadata()
		reports = append(reports, PerformanceReport{
			Strategy:        result.Strategy,
			StrategyVersion: metaDesc.Version,
			Category:        metaDesc.Category,
			Symbol:          result.Symbol,
			Timeframe:       result.Timeframe,
			Parameters:      strategylib.CloneParams(strategy.Parameters()),
			WarmupBars:      strategy.WarmupBars(),
			Statistics:      result.Statistics,
			Qualification:   Qualify(result.Statistics),
			GeneratedAt:     at,
		})
	}

	if meta.Symbol == "" && len(candles) > 0 {
		meta.Symbol = candles[0].Symbol
		meta.Timeframe = string(candles[0].Timeframe)
	}
	meta.CandleCount = len(candles)

	return BuildLeaderboard(reports, c.weights, meta)
}
