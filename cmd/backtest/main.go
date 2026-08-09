package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine/validation"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/catalog"
)

func main() {
	runAll := flagBool("all", false)
	strategyName := flagString("strategy", "ema_cross", "strategy name")
	csvPath := flagString("csv", "data/raw/nifty50/5min.csv", "CSV data path")
	symbol := flagString("symbol", "NIFTY50", "symbol")
	timeframe := flagString("timeframe", "5m", "timeframe")
	capital := flagFloat("capital", 100000, "initial capital")
	quantity := flagInt("quantity", 1, "trade quantity")
	commission := flagFloat("commission", 20, "commission per side")
	slippage := flagFloat("slippage", 0.01, "slippage percent")
	exportJSON := flagString("export-json", "output/leaderboard.json", "JSON export path")
	exportCSV := flagString("export-csv", "output/leaderboard.csv", "CSV export path")

	catalog.RegisterAll()

	tf := market.Timeframe(timeframe)
	candles, err := researchengine.LoadCSVCandles(csvPath, symbol, tf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load csv: %v\n", err)
		os.Exit(1)
	}

	simCfg := researchengine.SimulatorConfig{
		InitialCapital: capital,
		Quantity:       quantity,
		Commission:     commission,
		SlippagePct:    slippage,
	}

	if runAll {
		runChampionship(candles, csvPath, simCfg, exportJSON, exportCSV)
		return
	}

	strategy, ok := strategylib.Get(strategyName)
	if !ok {
		fmt.Fprintf(os.Stderr, "strategy not found: %s\n", strategyName)
		os.Exit(1)
	}

	engine := researchengine.NewEngine(simCfg)
	result := engine.RunStrategy(strategy, candles)
	report := researchengine.FormatReport(result.Strategy, result.Symbol, result.Timeframe, result.Statistics)
	fmt.Println(report)
	qual := researchengine.Qualify(result.Statistics)
	fmt.Println()
	fmt.Print(researchengine.FormatQualification(qual))
	fmt.Printf("\nWarmup bars skipped: %d\n", strategy.WarmupBars())
	fmt.Printf("Trades recorded: %d\n", result.Journal.Len())
}

func runChampionship(candles []market.Candle, csvPath string, simCfg researchengine.SimulatorConfig, jsonPath, csvPathOut string) {
	validationReport := validation.RunSuiteWithDataset(candles)
	fmt.Print(validation.Format(validationReport))
	if !validationReport.Passed {
		fmt.Fprintln(os.Stderr, "validation suite failed — fix research engine before trusting championship results")
		os.Exit(1)
	}

	strategies := catalog.AllFresh()
	diags := make([]validation.StrategyDiagnostics, 0, len(strategies))
	for _, s := range strategies {
		if s == nil {
			continue
		}
		diags = append(diags, validation.DiagnoseStrategy(s, candles, simCfg))
	}
	fmt.Print(validation.FormatDiagnostics(diags))

	champ := researchengine.NewChampionshipEngine(simCfg, researchengine.DefaultRankingWeights())

	meta := researchengine.ChampionshipMeta{
		DataSource: csvPath,
	}
	if len(candles) > 0 {
		meta.Symbol = candles[0].Symbol
		meta.Timeframe = string(candles[0].Timeframe)
	}

	board := champ.Run(strategies, candles, meta)
	fmt.Print(researchengine.FormatLeaderboard(board))

	if jsonPath != "" {
		if err := researchengine.ExportLeaderboardJSON(board, jsonPath); err != nil {
			fmt.Fprintf(os.Stderr, "export json: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nExported JSON: %s\n", jsonPath)
	}
	if csvPathOut != "" {
		if err := researchengine.ExportLeaderboardCSV(board, csvPathOut); err != nil {
			fmt.Fprintf(os.Stderr, "export csv: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Exported CSV: %s\n", csvPathOut)
	}
}

func flagBool(name string, def bool) bool {
	for _, arg := range os.Args {
		if arg == "--"+name {
			return true
		}
		if strings.HasPrefix(arg, "--"+name+"=") {
			v := strings.TrimPrefix(arg, "--"+name+"=")
			return v == "true" || v == "1"
		}
	}
	return def
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
