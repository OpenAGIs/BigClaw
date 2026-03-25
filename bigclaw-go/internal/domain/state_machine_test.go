package domain

import (
	"testing"
	"time"
)

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    TaskState
		to      TaskState
		wantErr bool
	}{
		{name: "queued to leased", from: TaskQueued, to: TaskLeased},
		{name: "leased to running", from: TaskLeased, to: TaskRunning},
		{name: "running to success", from: TaskRunning, to: TaskSucceeded},
		{name: "queued to success invalid", from: TaskQueued, to: TaskSucceeded, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTransition(test.from, test.to)
			if test.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestCanTransitionAndTimestampFallbackHelpers(t *testing.T) {
	if !CanTransition(TaskQueued, TaskLeased) {
		t.Fatal("expected queued -> leased to be allowed")
	}
	if CanTransition(TaskSucceeded, TaskRunning) {
		t.Fatal("expected succeeded -> running to stay disallowed")
	}

	if err := ValidateTransition(TaskSucceeded, TaskRunning); err == nil {
		t.Fatal("expected invalid terminal transition to return an error")
	}
	if err := ValidateTransition(TaskRunning, TaskRunning); err != nil {
		t.Fatalf("expected same-state validation to succeed, got %v", err)
	}

	fallback := time.Unix(1_700_000_300, 0).UTC()
	if got := timestampOrFallback(time.Time{}, fallback); !got.Equal(fallback) {
		t.Fatalf("expected zero timestamp to fall back, got %s", got)
	}

	value := fallback.Add(5 * time.Minute)
	if got := timestampOrFallback(value, fallback); !got.Equal(value) {
		t.Fatalf("expected non-zero timestamp to win, got %s", got)
	}
}
