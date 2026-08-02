package walkforward

import (
	"context"
	"sync"
)

// WindowHandler processes a single walk-forward window.
type WindowHandler func(ctx context.Context, window Window) error

// Scheduler dispatches walk-forward windows sequentially.
type Scheduler struct {
	queue chan Window

	mu      sync.Mutex
	active  map[int]struct{}
	pending int
}

// NewScheduler creates a window scheduler.
func NewScheduler(buffer int) *Scheduler {
	if buffer < 1 {
		buffer = 4
	}
	return &Scheduler{
		queue:  make(chan Window, buffer),
		active: make(map[int]struct{}),
	}
}

// Enqueue adds windows to the scheduler queue.
func (s *Scheduler) Enqueue(windows []Window) {
	s.mu.Lock()
	s.pending += len(windows)
	s.mu.Unlock()
	for _, window := range windows {
		s.queue <- window
	}
}

// ActiveWindows returns the number of windows currently executing.
func (s *Scheduler) ActiveWindows() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.active)
}

// QueueDepth returns the number of windows waiting in the queue.
func (s *Scheduler) QueueDepth() int {
	return len(s.queue)
}

// Start launches a worker that processes windows via the handler.
func (s *Scheduler) Start(ctx context.Context, handler WindowHandler) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case window, ok := <-s.queue:
				if !ok {
					return
				}
				if !s.tryStart(window.Index) {
					continue
				}
				if handler != nil {
					_ = handler(ctx, window)
				}
				s.finish(window.Index)
			}
		}
	}()
	return wg
}

func (s *Scheduler) tryStart(index int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.active[index]; exists {
		return false
	}
	s.active[index] = struct{}{}
	return true
}

func (s *Scheduler) finish(index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, index)
	if s.pending > 0 {
		s.pending--
	}
}

// PendingCount returns windows not yet completed.
func (s *Scheduler) PendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pending
}

// Close closes the scheduler queue.
func (s *Scheduler) Close() {
	close(s.queue)
}
