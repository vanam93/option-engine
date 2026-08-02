package execution

import (
	"context"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

// ExecutionAdapter is the broker-independent contract for order execution.
// Implementations translate approved trade intents into execution reports.
type ExecutionAdapter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Execute(ctx context.Context, intent ApprovedTradeIntent) (ExecutionReport, error)
	Health() health.Report
	Capabilities() Capabilities
}
