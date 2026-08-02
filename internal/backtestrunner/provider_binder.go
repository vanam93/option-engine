package backtestrunner

import (
	"context"

	"github.com/vanam-gangireddy/option-engine/internal/providers"
	providerapi "github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// ManagerBinder adapts providers.Manager to ProviderBinder.
type ManagerBinder struct {
	Manager *providers.Manager
}

// BindProvider wires a replay provider into the runtime market pipeline.
func (b *ManagerBinder) BindProvider(provider providerapi.Provider) error {
	if b == nil || b.Manager == nil {
		return ErrNilRunner
	}
	return b.Manager.InitWithProvider(provider)
}

// Connect connects the active provider.
func (b *ManagerBinder) Connect(ctx context.Context) error {
	if b == nil || b.Manager == nil {
		return ErrNilRunner
	}
	return b.Manager.Connect(ctx)
}

// Disconnect disconnects the active provider.
func (b *ManagerBinder) Disconnect(ctx context.Context) error {
	if b == nil || b.Manager == nil {
		return ErrNilRunner
	}
	return b.Manager.Disconnect(ctx)
}

// Subscribe subscribes the active provider to symbols.
func (b *ManagerBinder) Subscribe(ctx context.Context, symbols []string) error {
	if b == nil || b.Manager == nil {
		return ErrNilRunner
	}
	return b.Manager.Subscribe(ctx, symbols)
}
