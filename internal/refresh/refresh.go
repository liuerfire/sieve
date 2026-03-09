package refresh

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/liuerfire/sieve/internal/engine"
)

var ErrAlreadyRunning = errors.New("refresh already running")

const defaultEventBufferSize = 500

type EventKind string

const (
	EventKindRefreshStarted  EventKind = "refresh_started"
	EventKindProgress        EventKind = "progress"
	EventKindRefreshFinished EventKind = "refresh_finished"
)

type Event struct {
	RunID     string    `json:"run_id"`
	Seq       int       `json:"seq"`
	Timestamp time.Time `json:"timestamp"`
	Kind      EventKind `json:"kind"`
	Type      string    `json:"type,omitempty"`
	Source    string    `json:"source,omitempty"`
	Item      string    `json:"item,omitempty"`
	Message   string    `json:"message,omitempty"`
	Level     string    `json:"level,omitempty"`
	Count     int       `json:"count,omitempty"`
	Total     int       `json:"total,omitempty"`
	Error     string    `json:"error,omitempty"`
}

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

type Runner func(context.Context, func(engine.ProgressEvent)) (*engine.EngineResult, error)

type Coordinator struct {
	mu           sync.Mutex
	status       Status
	run          Runner
	runSeq       int
	runID        string
	nextRunID    int64
	nextSubID    int
	events       []Event
	subscribers  map[int]chan Event
	eventBufSize int
}

func NewCoordinator(run Runner) *Coordinator {
	return &Coordinator{
		run:          run,
		subscribers:  make(map[int]chan Event),
		eventBufSize: defaultEventBufferSize,
	}
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
	c.startRunLocked(now)
	c.publishLocked(Event{
		Kind:      EventKindRefreshStarted,
		Timestamp: now,
		Message:   fmt.Sprintf("Refresh triggered by %s", source),
	})
	c.mu.Unlock()

	result, err := c.run(ctx, c.publishProgress)

	c.mu.Lock()
	defer c.mu.Unlock()

	finishedAt := time.Now()
	c.status.Running = false
	c.status.LastCompletedAt = &finishedAt
	if err != nil {
		c.status.LastError = err.Error()
		c.publishLocked(Event{
			Kind:      EventKindRefreshFinished,
			Timestamp: finishedAt,
			Error:     err.Error(),
			Message:   "Refresh failed",
		})
		return c.status.snapshot(), err
	}

	c.status.LastSuccessAt = &finishedAt
	c.status.LastResult = summarizeResult(result)
	c.publishLocked(Event{
		Kind:      EventKindRefreshFinished,
		Timestamp: finishedAt,
		Message:   "Refresh complete",
	})
	return c.status.snapshot(), nil
}

func (c *Coordinator) Subscribe() ([]Event, <-chan Event, func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextSubID
	c.nextSubID++

	ch := make(chan Event, 64)
	c.subscribers[id] = ch
	replay := append([]Event(nil), c.events...)

	cancel := func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		sub, ok := c.subscribers[id]
		if !ok {
			return
		}
		delete(c.subscribers, id)
		close(sub)
	}

	return replay, ch, cancel
}

func (c *Coordinator) publishProgress(ev engine.ProgressEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runID == "" {
		return
	}

	c.publishLocked(Event{
		Kind:      EventKindProgress,
		Timestamp: time.Now(),
		Type:      ev.Type,
		Source:    ev.Source,
		Item:      ev.Item,
		Message:   ev.Message,
		Level:     ev.Level,
		Count:     ev.Count,
		Total:     ev.Total,
	})
}

func (c *Coordinator) startRunLocked(now time.Time) {
	c.nextRunID++
	c.runID = fmt.Sprintf("run-%d", c.nextRunID)
	c.runSeq = 0
	c.events = c.events[:0]
}

func (c *Coordinator) publishLocked(ev Event) {
	c.runSeq++
	ev.RunID = c.runID
	ev.Seq = c.runSeq
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}

	c.events = append(c.events, ev)
	if len(c.events) > c.eventBufSize {
		c.events = append([]Event(nil), c.events[len(c.events)-c.eventBufSize:]...)
	}

	for _, ch := range c.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
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
