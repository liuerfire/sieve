package retry

import (
	"errors"
	"testing"
)

func TestWithRetry_RetriesAndReturnsLastError(t *testing.T) {
	attempts := 0
	wantErr := errors.New("still failing")

	_, err := WithRetry(1, func() (int, error) {
		attempts++
		return 0, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
