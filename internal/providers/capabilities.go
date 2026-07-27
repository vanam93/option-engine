package providers

// Capabilities describes what a provider can supply.
// See provider.go for the struct definition and helper predicates.

// AllLive returns capabilities for a full live broker feed.
func AllLive() Capabilities {
	return Capabilities{
		LiveTicks:      true,
		OptionChain:    true,
		HistoricalData: true,
		Replay:         false,
		OrderPlacement: true,
	}
}

// MockCapabilities returns capabilities for the mock provider.
func MockCapabilities() Capabilities {
	return Capabilities{
		LiveTicks:      true,
		OptionChain:    false,
		HistoricalData: false,
		Replay:         false,
	}
}

// ReplayCapabilities returns capabilities for the replay provider.
func ReplayCapabilities() Capabilities {
	return Capabilities{
		LiveTicks:      false,
		OptionChain:    true,
		HistoricalData: true,
		Replay:         true,
	}
}
