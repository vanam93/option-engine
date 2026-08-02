package paper

import (
	"context"

	"github.com/vanam-gangireddy/option-engine/internal/execution"
)

var _ execution.ExecutionAdapter = (*Engine)(nil)

// Stop gracefully shuts down the paper adapter.
func (e *Engine) Stop(ctx context.Context) error {
	_ = ctx
	return e.Close()
}

// Execute simulates an approved trade intent and returns an execution report.
func (e *Engine) Execute(ctx context.Context, intent execution.ApprovedTradeIntent) (execution.ExecutionReport, error) {
	if err := ctx.Err(); err != nil {
		return execution.ExecutionReport{}, err
	}
	return e.executor.Execute(intent, e.cache), nil
}

// Capabilities reports paper execution features.
func (e *Engine) Capabilities() execution.Capabilities {
	return execution.Capabilities{
		Mode:              execution.CapabilityModeSimulated,
		Simulated:         true,
		MarketOrders:      true,
		LimitOrders:       false,
		PartialFills:      false,
		RealTimeExecution: false,
	}
}
