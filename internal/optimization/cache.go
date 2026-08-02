package optimization

import (
	"sort"
	"sync"
	"time"
)

type applyResult struct {
	Record  EvaluationRecord
	Ranking []EvaluationRecord
}

type evaluationState struct {
	totalTrades     int
	winRate         float64
	realizedPnL     float64
	unrealizedPnL   float64
	grossProfit     float64
	grossLoss       float64
	winningTrades   int
	avgWin          float64
	avgLoss         float64
	maxDrawdown     float64
	score           float64
	metrics         EvaluationMetrics
	lastTotalTrades int
	lastRealizedPnL float64
	updatedAt       time.Time
}

// Cache stores evaluations, rankings, and historical scores.
type Cache struct {
	mu sync.Mutex

	evaluations map[EvaluationKey]*evaluationState
	rankings    []EvaluationRecord
	scoreHist   map[EvaluationKey][]float64
}

// NewCache creates optimization state storage.
func NewCache() *Cache {
	return &Cache{
		evaluations: make(map[EvaluationKey]*evaluationState),
		scoreHist:   make(map[EvaluationKey][]float64),
	}
}

// Apply processes a performance update and returns the latest evaluation with rankings.
func (c *Cache) Apply(update InputUpdate, weights ScoringConfig) applyResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := EvaluationKey{
		Strategy:   update.Strategy,
		Symbol:     update.Symbol,
		Timeframe:  update.Timeframe,
		Parameters: update.Parameters,
	}

	state, ok := c.evaluations[key]
	if !ok {
		state = &evaluationState{}
		c.evaluations[key] = state
	}

	c.updateTradeState(state, update)

	maxDD := state.maxDrawdown
	if update.MaxDrawdown > maxDD {
		maxDD = update.MaxDrawdown
	}
	state.maxDrawdown = maxDD

	metrics := ComputeMetrics(
		state.totalTrades,
		state.winRate,
		state.realizedPnL,
		state.unrealizedPnL,
		state.grossProfit,
		state.grossLoss,
		state.avgWin,
		state.avgLoss,
		maxDD,
	)

	if update.ProfitFactor > 0 {
		metrics.ProfitFactor = update.ProfitFactor
	}

	score := Score(metrics, weights)
	at := update.Timestamp
	if at.IsZero() {
		at = time.Now().UTC()
	}
	state.score = score
	state.metrics = metrics
	state.updatedAt = at

	c.scoreHist[key] = append(c.scoreHist[key], score)
	c.rebuildRankings()

	record := EvaluationRecord{
		Key:       key,
		Metrics:   metrics,
		Score:     score,
		Rank:      c.rankForKey(key),
		UpdatedAt: at,
	}

	return applyResult{
		Record:  record,
		Ranking: append([]EvaluationRecord(nil), c.rankings...),
	}
}

func (c *Cache) updateTradeState(state *evaluationState, update InputUpdate) {
	if update.TotalTrades > state.lastTotalTrades {
		delta := update.RealizedPnL - state.lastRealizedPnL
		if delta > 0 {
			state.grossProfit += delta
			state.winningTrades++
			wins := state.winningTrades
			state.avgWin = ((state.avgWin * float64(wins-1)) + delta) / float64(wins)
		} else if delta < 0 {
			loss := -delta
			state.grossLoss += loss
			losses := update.TotalTrades - state.winningTrades
			if losses > 0 {
				state.avgLoss = ((state.avgLoss * float64(losses-1)) + loss) / float64(losses)
			}
		}
	}

	state.totalTrades = update.TotalTrades
	state.winRate = update.WinRate
	state.realizedPnL = update.RealizedPnL
	state.unrealizedPnL = update.UnrealizedPnL
	state.lastTotalTrades = update.TotalTrades
	state.lastRealizedPnL = update.RealizedPnL

	if update.Drawdown > state.maxDrawdown {
		state.maxDrawdown = update.Drawdown
	}
}

func (c *Cache) rebuildRankings() {
	records := make([]EvaluationRecord, 0, len(c.evaluations))
	for k, st := range c.evaluations {
		records = append(records, EvaluationRecord{
			Key:       k,
			Metrics:   st.metrics,
			Score:     st.score,
			UpdatedAt: st.updatedAt,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Score != records[j].Score {
			return records[i].Score > records[j].Score
		}
		return records[i].Key.Strategy < records[j].Key.Strategy
	})

	for i := range records {
		records[i].Rank = i + 1
	}
	c.rankings = records
}

func (c *Cache) rankForKey(key EvaluationKey) int {
	for _, r := range c.rankings {
		if r.Key == key {
			return r.Rank
		}
	}
	return 0
}

// Snapshot returns an immutable copy of current optimization state.
func (c *Cache) Snapshot() StateSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	return StateSnapshot{
		Evaluations: append([]EvaluationRecord(nil), c.rankings...),
		Rankings:    append([]EvaluationRecord(nil), c.rankings...),
	}
}

func (c *Cache) strategiesEvaluated() int {
	return len(c.evaluations)
}

func (c *Cache) rankingsCount() int {
	return len(c.rankings)
}
