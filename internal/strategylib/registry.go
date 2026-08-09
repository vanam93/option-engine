package strategylib

import (
	"sort"
	"sync"
)

// Registry indexes research strategies by metadata for search and discovery.
type Registry struct {
	mu sync.RWMutex

	strategies []Strategy
	byName     map[string]Strategy
	byCategory map[Category][]Strategy
	byRegime   map[Regime][]Strategy
	byTimeframe map[string][]Strategy
	byRisk     map[RiskLevel][]Strategy
}

// NewRegistry creates an empty strategy registry.
func NewRegistry() *Registry {
	return &Registry{
		byName:      make(map[string]Strategy),
		byCategory:  make(map[Category][]Strategy),
		byRegime:    make(map[Regime][]Strategy),
		byTimeframe: make(map[string][]Strategy),
		byRisk:      make(map[RiskLevel][]Strategy),
	}
}

var defaultRegistry = NewRegistry()

// Register adds a strategy to the default registry.
func Register(s Strategy) {
	defaultRegistry.Register(s)
}

// All returns every registered strategy from the default registry.
func All() []Strategy {
	return defaultRegistry.All()
}

// ByCategory returns strategies matching a category from the default registry.
func ByCategory(category Category) []Strategy {
	return defaultRegistry.ByCategory(category)
}

// ByRegime returns strategies preferring a regime from the default registry.
func ByRegime(regime Regime) []Strategy {
	return defaultRegistry.ByRegime(regime)
}

// ByTimeframe returns strategies supporting a timeframe from the default registry.
func ByTimeframe(timeframe string) []Strategy {
	return defaultRegistry.ByTimeframe(timeframe)
}

// ByRisk returns strategies with a risk level from the default registry.
func ByRisk(risk RiskLevel) []Strategy {
	return defaultRegistry.ByRisk(risk)
}

// SupportedStrategies returns strategies supporting symbol and timeframe.
func SupportedStrategies(symbol, timeframe string) []Strategy {
	return defaultRegistry.SupportedStrategies(symbol, timeframe)
}

// Get returns a strategy by name from the default registry.
func Get(name string) (Strategy, bool) {
	return defaultRegistry.Get(name)
}

// Names returns sorted strategy names from the default registry.
func Names() []string {
	return defaultRegistry.Names()
}

// Descriptors returns registry descriptors for all strategies.
func Descriptors() []StrategyDescriptor {
	return defaultRegistry.Descriptors()
}

// GetDescriptor returns a strategy descriptor by name.
func GetDescriptor(name string) (StrategyDescriptor, bool) {
	return defaultRegistry.GetDescriptor(name)
}

// Register adds a strategy prototype to the registry.
func (r *Registry) Register(s Strategy) {
	if s == nil {
		return
	}
	meta := s.Metadata()
	name := meta.Name
	if name == "" {
		name = s.Name()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.byName[name]; ok {
		r.removeIndexes(existing)
	}

	found := false
	for i, existing := range r.strategies {
		if existing.Name() == name {
			r.strategies[i] = s
			found = true
			break
		}
	}
	if !found {
		r.strategies = append(r.strategies, s)
	}

	r.byName[name] = s
	r.index(meta, s)
}

// All returns a copy of all registered strategies sorted by name.
func (r *Registry) All() []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := append([]Strategy(nil), r.strategies...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

// ByCategory returns strategies in a category.
func (r *Registry) ByCategory(category Category) []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Strategy(nil), r.byCategory[category]...)
}

// ByRegime returns strategies preferring a regime.
func (r *Registry) ByRegime(regime Regime) []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Strategy(nil), r.byRegime[regime]...)
}

// ByTimeframe returns strategies supporting a timeframe.
func (r *Registry) ByTimeframe(timeframe string) []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Strategy(nil), r.byTimeframe[timeframe]...)
}

// ByRisk returns strategies with a risk level.
func (r *Registry) ByRisk(risk RiskLevel) []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Strategy(nil), r.byRisk[risk]...)
}

// SupportedStrategies returns strategies that support the timeframe.
// Symbol is accepted for future symbol-type filtering; currently all registered strategies apply.
func (r *Registry) SupportedStrategies(symbol, timeframe string) []Strategy {
	_ = symbol
	if timeframe == "" {
		return r.All()
	}
	return r.ByTimeframe(timeframe)
}

// Get returns a strategy by name.
func (r *Registry) Get(name string) (Strategy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byName[name]
	return s, ok
}

// Names returns sorted registered strategy names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byName))
	for name := range r.byName {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Descriptors returns read models for all registered strategies.
func (r *Registry) Descriptors() []StrategyDescriptor {
	all := r.All()
	out := make([]StrategyDescriptor, 0, len(all))
	for _, s := range all {
		out = append(out, r.descriptorFor(s))
	}
	return out
}

// GetDescriptor returns a descriptor for a strategy name.
func (r *Registry) GetDescriptor(name string) (StrategyDescriptor, bool) {
	s, ok := r.Get(name)
	if !ok {
		return StrategyDescriptor{}, false
	}
	return r.descriptorFor(s), true
}

func (r *Registry) descriptorFor(s Strategy) StrategyDescriptor {
	meta := s.Metadata()
	return StrategyDescriptor{
		Name:              s.Name(),
		Version:           meta.Version,
		Category:          meta.Category,
		Metadata:          meta,
		WarmupBars:        s.WarmupBars(),
		DefaultParameters: CloneParams(s.DefaultParameters()),
		ParameterRanges:   append([]ParameterRange(nil), s.ParameterRanges()...),
	}
}

func (r *Registry) index(meta Metadata, s Strategy) {
	r.byCategory[meta.Category] = appendUniqueStrategy(r.byCategory[meta.Category], s)
	r.byRisk[meta.RiskLevel] = appendUniqueStrategy(r.byRisk[meta.RiskLevel], s)
	for _, tf := range meta.SupportedTimeframes {
		r.byTimeframe[tf] = appendUniqueStrategy(r.byTimeframe[tf], s)
	}
	for _, regime := range meta.PreferredRegimes {
		r.byRegime[regime] = appendUniqueStrategy(r.byRegime[regime], s)
	}
}

func (r *Registry) removeIndexes(s Strategy) {
	meta := s.Metadata()
	r.byCategory[meta.Category] = removeStrategy(r.byCategory[meta.Category], s)
	r.byRisk[meta.RiskLevel] = removeStrategy(r.byRisk[meta.RiskLevel], s)
	for _, tf := range meta.SupportedTimeframes {
		r.byTimeframe[tf] = removeStrategy(r.byTimeframe[tf], s)
	}
	for _, regime := range meta.PreferredRegimes {
		r.byRegime[regime] = removeStrategy(r.byRegime[regime], s)
	}
	for i, existing := range r.strategies {
		if existing.Name() == s.Name() {
			r.strategies = append(r.strategies[:i], r.strategies[i+1:]...)
			break
		}
	}
}

func appendUniqueStrategy(list []Strategy, s Strategy) []Strategy {
	for _, existing := range list {
		if existing.Name() == s.Name() {
			return list
		}
	}
	return append(list, s)
}

func removeStrategy(list []Strategy, s Strategy) []Strategy {
	out := make([]Strategy, 0, len(list))
	for _, existing := range list {
		if existing.Name() != s.Name() {
			out = append(out, existing)
		}
	}
	return out
}
