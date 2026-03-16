package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readReport(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "reports", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestWorkerRuntimeCloseoutDigestCoversTaskContract(t *testing.T) {
	digest := readReport(t, "worker-runtime-closeout-digest.md")
	taskProtocol := readReport(t, "task-protocol-spec.md")
	stateReport := readReport(t, "state-machine-validation-report.md")
	lifecycleReport := readReport(t, "worker-lifecycle-validation-report.md")

	for _, state := range []TaskState{
		TaskQueued,
		TaskLeased,
		TaskRunning,
		TaskBlocked,
		TaskRetrying,
		TaskSucceeded,
		TaskFailed,
		TaskCancelled,
		TaskDeadLetter,
	} {
		quoted := "`" + string(state) + "`"
		if !strings.Contains(digest, quoted) {
			t.Fatalf("expected digest to mention state %s", state)
		}
		if !strings.Contains(taskProtocol, quoted) {
			t.Fatalf("expected task protocol report to mention state %s", state)
		}
	}

	for _, transition := range []string{
		"`queued -> leased`",
		"`queued -> cancelled`",
		"`leased -> running`",
		"`leased -> retrying`",
		"`leased -> cancelled`",
		"`running -> blocked`",
		"`running -> dead_letter`",
		"`blocked -> retrying`",
		"`retrying -> queued`",
		"`failed -> dead_letter`",
	} {
		if !strings.Contains(digest, transition) {
			t.Fatalf("expected digest to mention transition %s", transition)
		}
		if !strings.Contains(stateReport, transition) {
			t.Fatalf("expected state-machine report to mention transition %s", transition)
		}
	}

	for _, surface := range []string{
		"`GET /tasks/{id}`",
		"`GET /events?task_id=...`",
		"`GET /events?trace_id=...`",
		"`GET /replay/{id}`",
		"`GET /stream/events`",
		"`GET /debug/status`",
	} {
		if !strings.Contains(digest, surface) {
			t.Fatalf("expected digest to mention surface %s", surface)
		}
	}

	for _, field := range []string{
		"`current_task_id`",
		"`current_trace_id`",
		"`lease_renewals`",
		"`successful_runs`",
		"`cancelled_runs`",
		"`preemption_active`",
		"`last_transition`",
	} {
		if !strings.Contains(digest, field) {
			t.Fatalf("expected digest to mention debug field %s", field)
		}
	}

	for _, report := range []string{taskProtocol, stateReport, lifecycleReport} {
		if !strings.Contains(report, "worker-runtime-closeout-digest.md") {
			t.Fatal("expected focused report to point back to canonical closeout digest")
		}
	}
}
