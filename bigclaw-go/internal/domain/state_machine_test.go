package domain

import "testing"

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
