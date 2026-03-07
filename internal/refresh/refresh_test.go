package refresh

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/liuerfire/sieve/internal/engine"
)

func TestCoordinator_TriggerUpdatesStatus(t *testing.T) {
	c := NewCoordinator(func(ctx context.Context) (*engine.EngineResult, error) {
		return &engine.EngineResult{
			SourcesProcessed:  2,
			ItemsProcessed:    5,
			ItemsHighInterest: 1,
		}, nil
	})

	status, err := c.Trigger(t.Context(), "manual")
	if err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	if status.Running {
		t.Fatal("expected status not running after trigger completes")
	}
	if status.LastTrigger != "manual" {
		t.Fatalf("expected last trigger manual, got %q", status.LastTrigger)
	}
	if status.LastCompletedAt == nil {
		t.Fatal("expected completion timestamp")
	}
	if status.LastSuccessAt == nil {
		t.Fatal("expected success timestamp")
	}
	if status.LastResult == nil || status.LastResult.ItemsProcessed != 5 {
		t.Fatalf("expected result counts, got %#v", status.LastResult)
	}
}

func TestCoordinator_TriggerRejectsOverlap(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})

	c := NewCoordinator(func(ctx context.Context) (*engine.EngineResult, error) {
		close(started)
		<-release
		return &engine.EngineResult{}, nil
	})

	errCh := make(chan error, 1)
	go func() {
		_, err := c.Trigger(context.Background(), "schedule")
		errCh <- err
	}()

	<-started

	status, err := c.Trigger(t.Context(), "manual")
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
	if !status.Running {
		t.Fatal("expected status to report running")
	}

	close(release)

	if err := <-errCh; err != nil {
		t.Fatalf("expected first trigger to finish cleanly, got %v", err)
	}
}

func TestCoordinator_TriggerTracksFailure(t *testing.T) {
	c := NewCoordinator(func(ctx context.Context) (*engine.EngineResult, error) {
		return nil, errors.New("boom")
	})

	status, err := c.Trigger(t.Context(), "manual")
	if err == nil {
		t.Fatal("expected trigger error")
	}
	if status.LastError != "boom" {
		t.Fatalf("expected last error recorded, got %q", status.LastError)
	}
	if status.LastCompletedAt == nil {
		t.Fatal("expected completion timestamp on failure")
	}
	if status.LastSuccessAt != nil {
		t.Fatal("did not expect success timestamp on failure")
	}
}

func TestSnapshotClonesTimestamps(t *testing.T) {
	now := time.Now()
	status := Status{LastStartedAt: &now}
	snapshot := status.snapshot()
	if snapshot.LastStartedAt == status.LastStartedAt {
		t.Fatal("expected snapshot to clone timestamp pointer")
	}
}
