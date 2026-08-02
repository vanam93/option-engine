package airesearch

import (
	"context"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
)

// ReportBuilder assembles research reports from study data via a ResearchAnalyzer.
type ReportBuilder struct {
	analyzer ResearchAnalyzer
	cfg      Config
}

// NewReportBuilder creates a report builder with the configured analyzer.
func NewReportBuilder(cfg Config, analyzer ResearchAnalyzer) *ReportBuilder {
	return &ReportBuilder{
		analyzer: analyzer,
		cfg:      cfg,
	}
}

// Build generates a complete research report for a completed study.
func (b *ReportBuilder) Build(ctx context.Context, study laboratory.Study, at time.Time) (ResearchReport, error) {
	prompt := BuildPrompt(study)
	sections, err := b.analyzer.Analyze(ctx, study, prompt)
	if err != nil {
		return ResearchReport{}, err
	}

	report := ResearchReport{
		ReportID:        generateReportID(at),
		StudyID:         study.StudyID,
		ResearchVersion: study.ResearchVersion,
		Analyzer:        b.cfg.Analyzer,
		Prompt:          prompt,
		Sections:        sections,
		GeneratedAt:     at,
	}
	report.FormattedText = FormatReport(report)
	return report, nil
}
