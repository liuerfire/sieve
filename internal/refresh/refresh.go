package refresh

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/liuerfire/sieve/internal/engine"
)

var ErrAlreadyRunning = errors.New("refresh already running")

type Result struct {
	SourcesProcessed  int `json:"sources_processed"`
	SourcesFailed     int `json:"sources_failed"`
	ItemsProcessed    int `json:"items_processed"`
	ItemsHighInterest int `json:"items_high_interest"`
}

type Status struct {
	Running         bool       `json:"running"`
	LastTrigger     string     `json:"last_trigger,omitempty"`
	LastStartedAt   *time.Time `json:"last_started_at,omitempty"`
	LastCompletedAt *time.Time `json:"last_completed_at,omitempty"`
	LastSuccessAt   *time.Time `json:"last_success_at,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	LastResult      *Result    `json:"last_result,omitempty"`
}

func (s Status) snapshot() Status {
	clone := s
	clone.LastStartedAt = cloneTime(s.LastStartedAt)
	clone.LastCompletedAt = cloneTime(s.LastCompletedAt)
	clone.LastSuccessAt = cloneTime(s.LastSuccessAt)
	if s.LastResult != nil {
		result := *s.LastResult
		clone.LastResult = &result
	}
	return clone
}

type Runner func(context.Context) (*engine.EngineResult, error)

type Coordinator struct {
	mu     sync.Mutex
	status Status
	run    Runner
}

func NewCoordinator(run Runner) *Coordinator {
	return &Coordinator{run: run}
}

func (c *Coordinator) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status.snapshot()
}

func (c *Coordinator) Trigger(ctx context.Context, source string) (Status, error) {
	c.mu.Lock()
	if c.status.Running {
		status := c.status.snapshot()
		c.mu.Unlock()
		return status, ErrAlreadyRunning
	}

	now := time.Now()
	c.status.Running = true
	c.status.LastTrigger = source
	c.status.LastStartedAt = &now
	c.status.LastError = ""
	c.mu.Unlock()

	result, err := c.run(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()

	finishedAt := time.Now()
	c.status.Running = false
	c.status.LastCompletedAt = &finishedAt
	if err != nil {
		c.status.LastError = err.Error()
		return c.status.snapshot(), err
	}

	c.status.LastSuccessAt = &finishedAt
	c.status.LastResult = summarizeResult(result)
	return c.status.snapshot(), nil
}

func summarizeResult(result *engine.EngineResult) *Result {
	if result == nil {
		return nil
	}
	return &Result{
		SourcesProcessed:  result.SourcesProcessed,
		SourcesFailed:     len(result.SourcesFailed),
		ItemsProcessed:    result.ItemsProcessed,
		ItemsHighInterest: result.ItemsHighInterest,
	}
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}
