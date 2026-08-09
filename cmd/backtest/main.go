package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/catalog"
)

func main() {
	strategyName := flagString("strategy", "ema_cross", "strategy name")
	csvPath := flagString("csv", "data/raw/nifty50/5min.csv", "CSV data path")
	symbol := flagString("symbol", "NIFTY50", "symbol")
	timeframe := flagString("timeframe", "5m", "timeframe")
	capital := flagFloat("capital", 100000, "initial capital")
	quantity := flagInt("quantity", 1, "trade quantity")
	commission := flagFloat("commission", 20, "commission per side")
	slippage := flagFloat("slippage", 0.01, "slippage percent")

	catalog.RegisterAll()
	strategy, ok := strategylib.Get(strategyName)
	if !ok {
		fmt.Fprintf(os.Stderr, "strategy not found: %s\n", strategyName)
		os.Exit(1)
	}

	tf := market.Timeframe(timeframe)
	candles, err := researchengine.LoadCSVCandles(csvPath, symbol, tf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load csv: %v\n", err)
		os.Exit(1)
	}

	engine := researchengine.NewEngine(researchengine.SimulatorConfig{
		InitialCapital: capital,
		Quantity:       quantity,
		Commission:     commission,
		SlippagePct:    slippage,
	})

	result := engine.RunStrategy(strategy, candles)
	report := researchengine.FormatReport(result.Strategy, result.Symbol, result.Timeframe, result.Statistics)
	fmt.Println(report)
	fmt.Printf("\nWarmup bars skipped: %d\n", strategy.WarmupBars())
	fmt.Printf("Trades recorded: %d\n", result.Journal.Len())
}

func flagString(name, def, usage string) string {
	for i, arg := range os.Args {
		if arg == "--"+name && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, "--"+name+"=") {
			return strings.TrimPrefix(arg, "--"+name+"=")
		}
	}
	_ = def
	_ = usage
	return def
}

func flagFloat(name string, def float64, usage string) float64 {
	s := flagString(name, fmt.Sprintf("%v", def), usage)
	v, err := parseFloat(s)
	if err != nil {
		return def
	}
	return v
}

func flagInt(name string, def int, usage string) int {
	s := flagString(name, fmt.Sprintf("%d", def), usage)
	v, err := parseInt(s)
	if err != nil {
		return def
	}
	return v
}

func parseFloat(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}

func parseInt(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}
