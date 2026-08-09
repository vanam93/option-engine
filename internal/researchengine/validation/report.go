package validation

import (
	"fmt"
	"strings"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// CheckResult is one validation check outcome.
type CheckResult struct {
	Name    string   `json:"name"`
	Passed  bool     `json:"passed"`
	Details string   `json:"details"`
	Issues  []string `json:"issues,omitempty"`
}

// Report is the full validation suite output.
type Report struct {
	Passed  bool          `json:"passed"`
	Checks  []CheckResult `json:"checks"`
	Warnings []string     `json:"warnings,omitempty"`
}

// RunSuite executes all research engine validation checks.
func RunSuite() Report {
	report := Report{Passed: true}

	report.Checks = append(report.Checks, checkSimulatorLongPnL())
	report.Checks = append(report.Checks, checkSimulatorPositionRules())
	report.Checks = append(report.Checks, checkStatisticsDrawdown())
	report.Checks = append(report.Checks, checkStatisticsKnownTrades())
	report.Checks = append(report.Checks, checkStrategyEMAcross())
	report.Checks = append(report.Checks, checkStrategyMACD())
	report.Checks = append(report.Checks, checkStrategyDonchian())
	report.Checks = append(report.Checks, checkStrategyOpeningRangeSessions())
	report.Checks = append(report.Checks, checkJournalInvariants())

	for _, c := range report.Checks {
		if !c.Passed {
			report.Passed = false
		}
	}
	return report
}

// RunSuiteWithDataset adds dataset-specific warnings (e.g. zero volume).
func RunSuiteWithDataset(candles []market.Candle) Report {
	report := RunSuite()
	if len(candles) == 0 {
		report.Warnings = append(report.Warnings, "dataset is empty")
		return report
	}
	zeroVol := 0
	for _, c := range candles {
		if c.Volume <= 0 {
			zeroVol++
		}
	}
	if zeroVol == len(candles) {
		report.Warnings = append(report.Warnings, "all candles have zero volume — VWAP strategies will not trade")
	}
	return report
}

// Format renders a human-readable validation report.
func Format(report Report) string {
	var b strings.Builder
	b.WriteString("Research Engine Validation Report\n")
	b.WriteString("=================================\n\n")
	if report.Passed {
		b.WriteString("Overall: PASSED\n\n")
	} else {
		b.WriteString("Overall: FAILED\n\n")
	}

	for _, c := range report.Checks {
		status := "PASS"
		if !c.Passed {
			status = "FAIL"
		}
		b.WriteString(fmt.Sprintf("[%s] %s\n", status, c.Name))
		if c.Details != "" {
			b.WriteString("  " + c.Details + "\n")
		}
		for _, issue := range c.Issues {
			b.WriteString("  - " + issue + "\n")
		}
		b.WriteString("\n")
	}

	if len(report.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, w := range report.Warnings {
			b.WriteString("  - " + w + "\n")
		}
	}
	return b.String()
}
