package paper

import "github.com/vanam-gangireddy/option-engine/internal/execution"

// ExecutionStatus identifies the outcome of a simulated order.
type ExecutionStatus = execution.ExecutionStatus

const (
	Filled   = execution.Filled
	Rejected = execution.Rejected
)

// InputIntent mirrors the ApprovedTradeIntent payload consumed by the paper engine.
type InputIntent = execution.ApprovedTradeIntent

// ExecutionReport is the payload published on ExecutionReport events.
type ExecutionReport = execution.ExecutionReport
