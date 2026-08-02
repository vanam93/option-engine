package console

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/delivery"
)

// Formatter produces structured terminal output for delivery documents.
type Formatter struct {
	deliveryFmt *delivery.Formatter
}

// NewFormatter creates a console formatter.
func NewFormatter() *Formatter {
	return &Formatter{deliveryFmt: delivery.NewFormatter()}
}

// FormatBlock renders a delivery document as a multi-line terminal block.
func (f *Formatter) FormatBlock(doc delivery.DeliveryDocument, at time.Time) []string {
	lines := make([]string, 0, 32)
	lines = append(lines, strings.Repeat("═", 72))
	lines = append(lines, fmt.Sprintf("  RECOMMENDATION  %s", doc.RecommendationID))
	lines = append(lines, strings.Repeat("─", 72))

	lines = append(lines, f.pair("Time", formatTime(at)))
	lines = append(lines, f.pair("Symbol", doc.Symbol))
	lines = append(lines, f.pair("Option", optionLabel(doc)))
	lines = append(lines, f.pair("Strategy", doc.Strategy))
	lines = append(lines, f.pair("Recommendation Level", f.deliveryFmt.LevelLabel(doc.CurrentRecommendationLevel)))
	lines = append(lines, f.pair("Confidence", formatPercent(doc.CurrentConfidence)))
	lines = append(lines, f.pair("Current Status", f.deliveryFmt.StatusLabel(doc.CurrentStatus)))

	lines = append(lines, "")
	lines = append(lines, "  Research Scores")
	lines = append(lines, f.pair("Optimization Score", formatScore(doc.OptimizationScore)))
	lines = append(lines, f.pair("Walk Forward", formatScore(doc.WalkForwardResult)))
	lines = append(lines, f.pair("Monte Carlo", formatScore(doc.MonteCarloResult)))
	lines = append(lines, f.pair("Historical Win Rate", formatWinRate(doc)))

	lines = append(lines, "")
	lines = append(lines, "  Price & PnL")
	lines = append(lines, f.pair("Entry", formatPrice(doc.EntryPrice)))
	lines = append(lines, f.pair("Current Price", formatPrice(doc.LatestPrice)))
	lines = append(lines, f.pair("Target", unavailableLabel()))
	lines = append(lines, f.pair("Stop Loss", unavailableLabel()))
	lines = append(lines, f.pair("PnL", formatPnL(doc.CurrentPnL, doc.CurrentReturn)))

	if doc.QualityEvaluation != nil {
		lines = append(lines, "")
		lines = append(lines, "  Quality")
		q := doc.QualityEvaluation
		lines = append(lines, f.pair("Outcome", q.Outcome))
		lines = append(lines, f.pair("Classification", q.Classification))
		lines = append(lines, f.pair("Quality Score", formatScore(q.QualityScore)))
	}

	if doc.FeedbackMetrics != nil {
		lines = append(lines, "")
		lines = append(lines, "  Feedback")
		fm := doc.FeedbackMetrics
		lines = append(lines, f.pair("Strategy Success Rate", formatPercent(fm.StrategySuccessRate)))
		lines = append(lines, f.pair("Strategy Win Rate", formatPercent(fm.StrategyWinRate)))
		lines = append(lines, f.pair("Symbol Success Rate", formatPercent(fm.SymbolSuccessRate)))
		lines = append(lines, f.pair("Timeframe Win Rate", formatPercent(fm.TimeframeWinRate)))
		lines = append(lines, f.pair("Platform Success Rate", formatPercent(fm.PlatformSuccessRate)))
	}

	if len(doc.Timeline) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  Timeline")
		for _, entry := range doc.Timeline {
			lines = append(lines, f.timelineLine(entry))
		}
	}

	if strings.TrimSpace(doc.ResearchSummary) != "" {
		lines = append(lines, "")
		lines = append(lines, "  Research Summary")
		for _, line := range wrapText(doc.ResearchSummary, 68) {
			lines = append(lines, "    "+line)
		}
	}

	lines = append(lines, strings.Repeat("═", 72))
	return lines
}

func (f *Formatter) pair(label, value string) string {
	if value == "" {
		value = unavailableLabel()
	}
	return fmt.Sprintf("  %-22s %s", label+":", value)
}

func (f *Formatter) timelineLine(entry delivery.TimelineEntry) string {
	ts := formatTime(entry.Timestamp)
	event := string(entry.Event)
	detail := strings.TrimSpace(entry.Reason)
	if entry.PreviousValue != "" || entry.NewValue != "" {
		transition := strings.TrimSpace(entry.PreviousValue)
		if transition != "" {
			transition += " → "
		}
		transition += strings.TrimSpace(entry.NewValue)
		if detail != "" {
			detail = transition + " — " + detail
		} else {
			detail = transition
		}
	}
	if detail == "" {
		return fmt.Sprintf("    %s  %s", ts, event)
	}
	return fmt.Sprintf("    %s  %s — %s", ts, event, detail)
}

func optionLabel(doc delivery.DeliveryDocument) string {
	if strings.TrimSpace(doc.Recommendation) != "" {
		return doc.Recommendation
	}
	if doc.Symbol != "" && doc.Timeframe != "" {
		return doc.Symbol + " " + doc.Timeframe
	}
	return unavailableLabel()
}

func formatTime(at time.Time) string {
	if at.IsZero() {
		return unavailableLabel()
	}
	return at.UTC().Format("2006-01-02 15:04:05 UTC")
}

func formatPrice(price float64) string {
	if price == 0 {
		return unavailableLabel()
	}
	return strconv.FormatFloat(price, 'f', 2, 64)
}

func formatScore(score float64) string {
	if score == 0 {
		return unavailableLabel()
	}
	return formatPercent(score)
}

func formatPercent(value float64) string {
	if value == 0 {
		return unavailableLabel()
	}
	return strconv.FormatFloat(value*100, 'f', 1, 64) + "%"
}

func formatWinRate(doc delivery.DeliveryDocument) string {
	if doc.FeedbackMetrics != nil && doc.FeedbackMetrics.StrategyWinRate > 0 {
		return formatPercent(doc.FeedbackMetrics.StrategyWinRate)
	}
	return unavailableLabel()
}

func formatPnL(absolute, pct float64) string {
	if absolute == 0 && pct == 0 {
		return unavailableLabel()
	}
	return fmt.Sprintf("%s (%s)", strconv.FormatFloat(absolute, 'f', 2, 64), formatPercent(pct))
}

func unavailableLabel() string {
	return "—"
}

func wrapText(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	var current strings.Builder
	for _, word := range words {
		if current.Len() == 0 {
			current.WriteString(word)
			continue
		}
		if current.Len()+1+len(word) > width {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
			continue
		}
		current.WriteString(" ")
		current.WriteString(word)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}
