package validation_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine/validation"
)

func TestRunSuitePasses(t *testing.T) {
	report := validation.RunSuite()
	require.True(t, report.Passed, validation.Format(report))
}

func TestFormatReport(t *testing.T) {
	report := validation.RunSuite()
	out := validation.Format(report)
	require.Contains(t, out, "Validation Report")
}
