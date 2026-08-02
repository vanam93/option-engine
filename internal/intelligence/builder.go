package intelligence

import "time"

// Builder assembles intelligence documents from state updates.
type Builder struct {
	cfg      Config
	fmt      *Formatter
	explainer *Explainer
	summary  *SummaryBuilder
}

// NewBuilder creates an intelligence document builder.
func NewBuilder(cfg Config) *Builder {
	fmt := NewFormatter()
	return &Builder{
		cfg:       cfg.withDefaults(),
		fmt:       fmt,
		explainer: NewExplainer(cfg, fmt),
		summary:   NewSummaryBuilder(fmt),
	}
}

// Build constructs a complete intelligence document.
func (b *Builder) Build(update StateUpdate, timeline []TimelineEntry, previous *storedSnapshot, at time.Time) IntelligenceDocument {
	level := b.explainer.ResolveLevel(update)
	evidence := b.explainer.BuildEvidence(update)
	if b.cfg.IncludeResearch {
		evidence.Freshness = freshnessLabel(UpdatedAt(timeline, at), at)
	}

	upgrade, downgrade := b.explainer.DetectChange(update, previous)
	explanation := b.explainer.BuildExplanation(update, level, evidence, upgrade, downgrade)

	doc := IntelligenceDocument{
		RecommendationID:             update.RecommendationID,
		Symbol:                       update.Symbol,
		Timeframe:                    update.Timeframe,
		Strategy:                     update.Strategy,
		RecommendationLevel:          level,
		Confidence:                   update.Confidence,
		CurrentStatus:                update.CurrentStatus,
		CurrentRecommendationState:   b.fmt.StatusLabel(update.CurrentStatus),
		DecisionSummary:              b.summary.DecisionSummary(update, level),
		Explanation:                  explanation,
		SupportingFactors:            b.explainer.SupportingFactors(update, evidence, level),
		RiskFactors:                  b.explainer.RiskFactors(update, evidence),
		ReasonForUpgrade:             upgrade,
		ReasonForDowngrade:           downgrade,
		ResearchEvidence:             evidence,
		GeneratedAt:                  at,
	}

	if b.cfg.IncludeResearch {
		doc.ResearchSummary = b.summary.ResearchSummary(update, evidence)
	}
	if b.cfg.IncludeTimeline {
		doc.TimelineSummary = b.summary.TimelineSummary(timeline)
		doc.RecommendationHistory = append([]TimelineEntry(nil), timeline...)
	}
	if b.cfg.IncludeConfidenceBreakdown {
		doc.ConfidenceBreakdown = b.explainer.BuildConfidenceBreakdown(update)
	} else {
		doc.ConfidenceBreakdown = ConfidenceBreakdown{Overall: update.Confidence}
	}

	return doc
}
