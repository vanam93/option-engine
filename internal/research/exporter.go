package research

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Exporter writes unified research reports to disk.
type Exporter interface {
	Format() string
	Export(report UnifiedReport, directory string) (string, error)
}

// JSONExporter writes reports as JSON files.
type JSONExporter struct{}

func (JSONExporter) Format() string { return "json" }

func (JSONExporter) Export(report UnifiedReport, directory string) (string, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_v%d.json", report.ExperimentID, report.Version)
	path := filepath.Join(directory, filename)
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// CSVExporter writes reports as CSV files.
type CSVExporter struct{}

func (CSVExporter) Format() string { return "csv" }

func (CSVExporter) Export(report UnifiedReport, directory string) (string, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_v%d.csv", report.ExperimentID, report.Version)
	path := filepath.Join(directory, filename)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	_ = writer.Write([]string{"section", "key", "value"})
	_ = writer.Write([]string{"experiment", "research_id", report.ResearchID})
	_ = writer.Write([]string{"experiment", "experiment_id", report.ExperimentID})
	_ = writer.Write([]string{"experiment", "strategy", report.Strategy})
	_ = writer.Write([]string{"experiment", "symbol", report.Symbol})
	_ = writer.Write([]string{"experiment", "timeframe", report.Timeframe})
	_ = writer.Write([]string{"summary", "best_score", strconv.FormatFloat(report.Summary.BestScore, 'f', 4, 64)})
	_ = writer.Write([]string{"summary", "latest_validation_score", strconv.FormatFloat(report.Summary.LatestValidation, 'f', 4, 64)})
	_ = writer.Write([]string{"summary", "probability_of_profit", strconv.FormatFloat(report.Summary.ProbabilityOfProfit, 'f', 4, 64)})
	_ = writer.Write([]string{"summary", "risk_of_ruin", strconv.FormatFloat(report.Summary.RiskOfRuin, 'f', 4, 64)})

	for i, opt := range report.Optimization {
		prefix := fmt.Sprintf("optimization_%d", i)
		_ = writer.Write([]string{prefix, "score", strconv.FormatFloat(opt.Score, 'f', 4, 64)})
		_ = writer.Write([]string{prefix, "win_rate", strconv.FormatFloat(opt.WinRate, 'f', 4, 64)})
		_ = writer.Write([]string{prefix, "expectancy", strconv.FormatFloat(opt.Expectancy, 'f', 4, 64)})
		_ = writer.Write([]string{prefix, "profit_factor", strconv.FormatFloat(opt.ProfitFactor, 'f', 4, 64)})
		_ = writer.Write([]string{prefix, "drawdown", strconv.FormatFloat(opt.Drawdown, 'f', 4, 64)})
	}
	for i, wf := range report.WalkForward {
		prefix := fmt.Sprintf("walkforward_%d", i)
		_ = writer.Write([]string{prefix, "walkforward_id", wf.WalkForwardID})
		_ = writer.Write([]string{prefix, "train_score", strconv.FormatFloat(wf.TrainScore, 'f', 4, 64)})
		_ = writer.Write([]string{prefix, "validation_score", strconv.FormatFloat(wf.ValidationScore, 'f', 4, 64)})
	}
	for i, mc := range report.MonteCarlo {
		prefix := fmt.Sprintf("montecarlo_%d", i)
		_ = writer.Write([]string{prefix, "simulation_id", mc.SimulationID})
		_ = writer.Write([]string{prefix, "probability_of_profit", strconv.FormatFloat(mc.ProbabilityOfProfit, 'f', 4, 64)})
		_ = writer.Write([]string{prefix, "probability_of_loss", strconv.FormatFloat(mc.ProbabilityOfLoss, 'f', 4, 64)})
		_ = writer.Write([]string{prefix, "risk_of_ruin", strconv.FormatFloat(mc.RiskOfRuin, 'f', 4, 64)})
	}

	return path, nil
}

// NewExporters returns exporters for configured formats.
func NewExporters(formats []string) []Exporter {
	out := make([]Exporter, 0, len(formats))
	for _, format := range formats {
		switch format {
		case "json":
			out = append(out, JSONExporter{})
		case "csv":
			out = append(out, CSVExporter{})
		}
	}
	return out
}
