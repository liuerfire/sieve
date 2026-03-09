package refresh

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/liuerfire/sieve/internal/engine"
)

func TestCoordinator_TriggerUpdatesStatus(t *testing.T) {
	c := NewCoordinator(func(ctx context.Context, report func(engine.ProgressEvent)) (*engine.EngineResult, error) {
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

	c := NewCoordinator(func(ctx context.Context, report func(engine.ProgressEvent)) (*engine.EngineResult, error) {
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
	c := NewCoordinator(func(ctx context.Context, report func(engine.ProgressEvent)) (*engine.EngineResult, error) {
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

func TestCoordinator_SubscribeReplaysBufferedEvents(t *testing.T) {
	progressSent := make(chan struct{})
	release := make(chan struct{})

	var c *Coordinator
	c = NewCoordinator(func(ctx context.Context, report func(engine.ProgressEvent)) (*engine.EngineResult, error) {
		c.publishProgress(engine.ProgressEvent{Type: "source_start", Source: "feed-a"})
		close(progressSent)
		<-release
		return &engine.EngineResult{}, nil
	})

	done := make(chan error, 1)
	go func() {
		_, err := c.Trigger(context.Background(), "manual")
		done <- err
	}()

	<-progressSent

	replay, events, cancel := c.Subscribe()
	defer cancel()

	if len(replay) < 2 {
		t.Fatalf("expected buffered lifecycle and progress events, got %d", len(replay))
	}
	if replay[0].Kind != EventKindRefreshStarted {
		t.Fatalf("expected first replay event to be refresh_started, got %q", replay[0].Kind)
	}
	if replay[1].Kind != EventKindProgress {
		t.Fatalf("expected second replay event to be progress, got %q", replay[1].Kind)
	}
	if replay[1].Source != "feed-a" {
		t.Fatalf("expected replayed progress source feed-a, got %q", replay[1].Source)
	}

	close(release)

	ev := waitForEvent(t, events)
	if ev.Kind != EventKindRefreshFinished {
		t.Fatalf("expected live finish event, got %q", ev.Kind)
	}

	if err := <-done; err != nil {
		t.Fatalf("expected refresh to complete cleanly, got %v", err)
	}
}

func TestCoordinator_SubscribeReplaysMostRecentRun(t *testing.T) {
	var c *Coordinator
	c = NewCoordinator(func(ctx context.Context, report func(engine.ProgressEvent)) (*engine.EngineResult, error) {
		c.publishProgress(engine.ProgressEvent{Type: "source_done", Source: "feed-a"})
		return &engine.EngineResult{}, nil
	})

	if _, err := c.Trigger(t.Context(), "manual"); err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	replay, _, cancel := c.Subscribe()
	defer cancel()

	if len(replay) < 3 {
		t.Fatalf("expected replay for completed run, got %d events", len(replay))
	}
	if replay[len(replay)-1].Kind != EventKindRefreshFinished {
		t.Fatalf("expected last replay event to be refresh_finished, got %q", replay[len(replay)-1].Kind)
	}
}

func TestCoordinator_NewRunResetsBufferAndRunID(t *testing.T) {
	run := 0
	var c *Coordinator
	c = NewCoordinator(func(ctx context.Context, report func(engine.ProgressEvent)) (*engine.EngineResult, error) {
		run++
		c.publishProgress(engine.ProgressEvent{
			Type:   "item_done",
			Source: fmt.Sprintf("feed-%d", run),
			Item:   fmt.Sprintf("item-%d", run),
		})
		return &engine.EngineResult{}, nil
	})

	if _, err := c.Trigger(t.Context(), "manual"); err != nil {
		t.Fatalf("first Trigger failed: %v", err)
	}
	firstReplay, _, cancelFirst := c.Subscribe()
	defer cancelFirst()

	if _, err := c.Trigger(t.Context(), "manual"); err != nil {
		t.Fatalf("second Trigger failed: %v", err)
	}
	secondReplay, _, cancelSecond := c.Subscribe()
	defer cancelSecond()

	if len(firstReplay) == 0 || len(secondReplay) == 0 {
		t.Fatal("expected non-empty replays")
	}
	if firstReplay[0].RunID == secondReplay[0].RunID {
		t.Fatalf("expected distinct run ids, got %q", firstReplay[0].RunID)
	}
	for _, ev := range secondReplay {
		if ev.Source == "feed-1" || ev.Item == "item-1" {
			t.Fatalf("expected second run buffer to reset, found stale event %#v", ev)
		}
	}
}

func TestCoordinator_TriggerForwardsProgressCallback(t *testing.T) {
	replayReady := make(chan []Event, 1)

	var c *Coordinator
	c = NewCoordinator(func(ctx context.Context, report func(engine.ProgressEvent)) (*engine.EngineResult, error) {
		report(engine.ProgressEvent{
			Type:   "item_done",
			Source: "feed-a",
			Item:   "story-a",
			Level:  "interest",
		})

		replay, _, cancel := c.Subscribe()
		defer cancel()
		replayReady <- replay

		return &engine.EngineResult{}, nil
	})

	if _, err := c.Trigger(t.Context(), "manual"); err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	replay := <-replayReady
	if len(replay) < 2 {
		t.Fatalf("expected replay to include progress callback output, got %d events", len(replay))
	}
	progress := replay[1]
	if progress.Kind != EventKindProgress {
		t.Fatalf("expected progress event, got %q", progress.Kind)
	}
	if progress.Item != "story-a" || progress.Level != "interest" {
		t.Fatalf("expected forwarded progress payload, got %#v", progress)
	}
}

func waitForEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()

	select {
	case ev := <-ch:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}
