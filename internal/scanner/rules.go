package scanner

import (
	"strings"
	"time"
)

const (
	scannerEMA     = "ema"
	scannerRSI     = "rsi"
	scannerMACD    = "macd"
	scannerTrend   = "trend"
	scannerRanking = "ranking"

	signalStrategyEMACross  = "ema_cross"
	signalStrategyMACDCross = "macd_cross"
	signalStrategyRSI       = "rsi"

	strategyTrendFollowing = "trend_following"
)

// Evaluator applies enabled scanner rules to symbol state.
type Evaluator struct {
	cfg Config
}

// NewEvaluator creates a scanner rule evaluator from configuration.
func NewEvaluator(cfg Config) *Evaluator {
	return &Evaluator{cfg: cfg.WithDefaults()}
}

// EvaluateSignal runs signal-based scanners against the latest signal input.
func (e *Evaluator) EvaluateSignal(state *SymbolState) []ScanResult {
	if state == nil || !state.HasSignal {
		return nil
	}
	input := state.LastSignal
	var results []ScanResult

	if e.cfg.Scanners.EMA && input.Strategy == signalStrategyEMACross {
		if result, ok := e.evalSignalScanner(scannerEMA, input, "ema_crossover"); ok {
			results = append(results, result)
		}
	}
	if e.cfg.Scanners.RSI && input.Strategy == signalStrategyRSI {
		if result, ok := e.evalSignalScanner(scannerRSI, input, "rsi_threshold"); ok {
			results = append(results, result)
		}
	}
	if e.cfg.Scanners.MACD && input.Strategy == signalStrategyMACDCross {
		if result, ok := e.evalSignalScanner(scannerMACD, input, "macd_crossover"); ok {
			results = append(results, result)
		}
	}
	return results
}

func (e *Evaluator) evalSignalScanner(name string, input InputSignal, rule string) (ScanResult, bool) {
	signal := strings.ToUpper(strings.TrimSpace(input.Signal))
	if signal != "BUY" && signal != "SELL" {
		return ScanResult{}, false
	}
	if input.Confidence < e.cfg.MinConfidence {
		return ScanResult{
			Symbol:       input.Symbol,
			Timeframe:    input.Timeframe,
			ScannerName:  name,
			Status:       StatusWatch,
			Score:        input.Confidence,
			Confidence:   input.Confidence,
			MatchedRules: []string{rule + "_low_confidence"},
			Timestamp:    input.Timestamp,
		}, true
	}
	return ScanResult{
		Symbol:       input.Symbol,
		Timeframe:    input.Timeframe,
		ScannerName:  name,
		Status:       StatusMatch,
		Score:        input.Confidence,
		Confidence:   input.Confidence,
		MatchedRules: []string{rule, signal},
		Timestamp:    input.Timestamp,
	}, true
}

// EvaluateDecision runs strategy-based scanners against the latest decision input.
func (e *Evaluator) EvaluateDecision(state *SymbolState) []ScanResult {
	if state == nil || !state.HasDecision || !e.cfg.Scanners.Trend {
		return nil
	}
	input := state.LastDecision
	if input.Strategy != strategyTrendFollowing {
		return nil
	}

	decision := strings.ToUpper(strings.TrimSpace(input.Decision))
	if decision != "LONG_ENTRY" && decision != "SHORT_ENTRY" {
		return nil
	}

	status := StatusWatch
	if input.Confidence >= e.cfg.MinConfidence {
		status = StatusMatch
	}
	return []ScanResult{{
		Symbol:       input.Symbol,
		Timeframe:    input.Timeframe,
		ScannerName:  scannerTrend,
		Status:       status,
		Score:        input.Confidence,
		Confidence:   input.Confidence,
		MatchedRules: []string{"strong_trend", decision},
		Timestamp:    input.Timestamp,
	}}
}

// EvaluateRanking ranks symbols by composite performance score.
func (e *Evaluator) EvaluateRanking(performances []InputPerformance, trigger InputPerformance) []ScanResult {
	if !e.cfg.Scanners.Ranking || len(performances) == 0 {
		return nil
	}

	type ranked struct {
		input InputPerformance
		score float64
	}
	rankedList := make([]ranked, 0, len(performances))
	for _, perf := range performances {
		if !e.cfg.WatchesSymbol(perf.Symbol) {
			continue
		}
		score := perf.WinRate*0.5 + normalizePnL(perf.RealizedPnL+perf.UnrealizedPnL)*0.5
		rankedList = append(rankedList, ranked{input: perf, score: score})
	}
	if len(rankedList) == 0 {
		return nil
	}

	best := rankedList[0]
	for _, item := range rankedList[1:] {
		if item.score > best.score {
			best = item
		}
	}

	if trigger.Symbol != "" && best.input.Symbol != trigger.Symbol {
		return nil
	}

	at := trigger.Timestamp
	if at.IsZero() {
		at = time.Now().UTC()
	}

	return []ScanResult{{
		Symbol:       best.input.Symbol,
		Timeframe:    best.input.Timeframe,
		ScannerName:  scannerRanking,
		Status:       StatusMatch,
		Score:        best.score,
		Confidence:   best.input.WinRate,
		MatchedRules: []string{"top_rank", "performance_score"},
		Timestamp:    at,
	}}
}

func normalizePnL(pnl float64) float64 {
	if pnl <= 0 {
		return 0
	}
	if pnl >= 1000 {
		return 1
	}
	return pnl / 1000
}
