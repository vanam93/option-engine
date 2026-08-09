package researchengine

// SimulatorConfig controls fill simulation and costs.
type SimulatorConfig struct {
	InitialCapital float64
	Quantity       int
	Commission     float64
	TaxRate        float64
	SlippagePct    float64
}

func (c SimulatorConfig) withDefaults() SimulatorConfig {
	out := c
	if out.InitialCapital <= 0 {
		out.InitialCapital = 100000
	}
	if out.Quantity <= 0 {
		out.Quantity = 1
	}
	return out
}
