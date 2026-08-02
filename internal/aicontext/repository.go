package aicontext

import "sync"

// Repository stores generated AI context packages in memory.
type Repository struct {
	mu    sync.RWMutex
	items map[string]AIContext
	order []string
}

// NewRepository creates an empty context repository.
func NewRepository() *Repository {
	return &Repository{
		items: make(map[string]AIContext),
	}
}

// Save stores an AI context package.
func (r *Repository) Save(ctx AIContext) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[ctx.ContextID]; !exists {
		r.order = append(r.order, ctx.ContextID)
	}
	r.items[ctx.ContextID] = ctx
}

// Get returns a context by ID.
func (r *Repository) Get(id string) (AIContext, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ctx, ok := r.items[id]
	return ctx, ok
}

// Latest returns the most recently stored context.
func (r *Repository) Latest() (AIContext, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.order) - 1; i >= 0; i-- {
		if ctx, ok := r.items[r.order[i]]; ok {
			return ctx, true
		}
	}
	return AIContext{}, false
}

// List returns all contexts in generation order.
func (r *Repository) List() []AIContext {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AIContext, 0, len(r.order))
	for _, id := range r.order {
		if ctx, ok := r.items[id]; ok {
			out = append(out, ctx)
		}
	}
	return out
}

// Count returns the number of stored contexts.
func (r *Repository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items)
}
