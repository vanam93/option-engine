package feedback

// Learner applies completed recommendation quality reports to the aggregator.
type Learner struct {
	aggregator *Aggregator
}

// NewLearner creates a learner bound to an aggregator.
func NewLearner(aggregator *Aggregator) *Learner {
	return &Learner{aggregator: aggregator}
}

// Learn records one completed recommendation when it passes validation.
func (l *Learner) Learn(input QualityInput) FeedbackSnapshot {
	return l.aggregator.Apply(input)
}

// IsLearnable returns true when the input should be aggregated.
func IsLearnable(input QualityInput) bool {
	return input.RecommendationID != "" && input.Completed
}
