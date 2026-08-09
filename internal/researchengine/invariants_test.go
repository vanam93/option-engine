package researchengine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
)

func TestStatisticsMaxDrawdownCapped(t *testing.T) {
	journal := researchengine.NewJournal()
	for i := 0; i < 20; i++ {
		journal.Add(researchengine.SimulatedTrade{NetProfit: -100})
	}
	stats := researchengine.ComputeStatistics(journal, 100)
	require.LessOrEqual(t, stats.MaxDrawdownPercent, 100.0)
}

func TestValidateJournalValidTrade(t *testing.T) {
	journal := researchengine.NewJournal()
	now := time.Now()
	journal.Add(researchengine.SimulatedTrade{
		EntryTime:  now,
		ExitTime:   now.Add(5 * time.Minute),
		EntryPrice: 100,
		ExitPrice:  110,
		Quantity:   1,
		BarsHeld:   5,
	})
	issues := researchengine.ValidateJournal(journal)
	require.Empty(t, issues)
}
