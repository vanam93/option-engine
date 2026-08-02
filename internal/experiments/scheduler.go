package experiments

import (
	"context"
	"sync"
)

// BacktestRunner executes a single experiment run through the existing backtest pipeline.
type BacktestRunner interface {
	Execute(ctx context.Context, run ExperimentRun) error
}

// Scheduler dispatches experiment runs to workers with concurrency control.
type Scheduler struct {
	workers   int
	semaphore chan struct{}
	queue     chan ExperimentRun

	mu      sync.Mutex
	running map[string]struct{}
}

// NewScheduler creates a run scheduler.
func NewScheduler(workers, maxConcurrent int) *Scheduler {
	if workers < 1 {
		workers = 1
	}
	if maxConcurrent < 1 {
		maxConcurrent = workers
	}
	return &Scheduler{
		workers:   workers,
		semaphore: make(chan struct{}, maxConcurrent),
		queue:     make(chan ExperimentRun, workers*4),
		running:   make(map[string]struct{}),
	}
}

// Enqueue adds runs to the scheduler queue.
func (s *Scheduler) Enqueue(runs []ExperimentRun) {
	for _, run := range runs {
		s.queue <- run
	}
}

// QueueDepth returns the number of runs waiting in the queue.
func (s *Scheduler) QueueDepth() int {
	return len(s.queue)
}

// ActiveWorkers returns the number of runs currently executing.
func (s *Scheduler) ActiveWorkers() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.running)
}

// Start launches worker goroutines that execute runs via the runner.
func (s *Scheduler) Start(ctx context.Context, runner BacktestRunner, onStart func(ExperimentRun), onComplete func(ExperimentRun, error)) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx, runner, onStart, onComplete)
		}()
	}
	return wg
}

func (s *Scheduler) worker(ctx context.Context, runner BacktestRunner, onStart func(ExperimentRun), onComplete func(ExperimentRun, error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case run, ok := <-s.queue:
			if !ok {
				return
			}
			if !s.tryStart(run.RunID) {
				continue
			}

			select {
			case s.semaphore <- struct{}{}:
			case <-ctx.Done():
				s.finish(run.RunID)
				return
			}

			if onStart != nil {
				onStart(run)
			}

			err := runner.Execute(ctx, run)

			<-s.semaphore
			s.finish(run.RunID)

			if onComplete != nil {
				onComplete(run, err)
			}
		}
	}
}

func (s *Scheduler) tryStart(runID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.running[runID]; exists {
		return false
	}
	s.running[runID] = struct{}{}
	return true
}

func (s *Scheduler) finish(runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, runID)
}

// TryStartForTest exposes duplicate-run protection for tests.
func (s *Scheduler) TryStartForTest(runID string) bool {
	return s.tryStart(runID)
}

// FinishForTest releases a run slot for tests.
func (s *Scheduler) FinishForTest(runID string) {
	s.finish(runID)
}

// Close closes the scheduler queue.
func (s *Scheduler) Close() {
	close(s.queue)
}
