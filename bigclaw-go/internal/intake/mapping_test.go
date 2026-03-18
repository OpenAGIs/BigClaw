package intake

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestMapPriority(t *testing.T) {
	if got := MapPriority("P0"); got != int(domain.PriorityP0) {
		t.Fatalf("expected P0 -> %d, got %d", domain.PriorityP0, got)
	}
	if got := MapPriority("P1"); got != int(domain.PriorityP1) {
		t.Fatalf("expected P1 -> %d, got %d", domain.PriorityP1, got)
	}
	if got := MapPriority("UNKNOWN"); got != int(domain.PriorityP2) {
		t.Fatalf("expected unknown priority -> %d, got %d", domain.PriorityP2, got)
	}
}

func TestMapSourceState(t *testing.T) {
	cases := map[string]domain.TaskState{
		"Todo":         domain.TaskQueued,
		"In Progress":  domain.TaskRunning,
		"Blocked":      domain.TaskBlocked,
		"Closed":       domain.TaskSucceeded,
		"resolved":     domain.TaskSucceeded,
		"failed check": domain.TaskFailed,
	}
	for input, want := range cases {
		if got := MapSourceState(input); got != want {
			t.Fatalf("expected %q -> %q, got %q", input, want, got)
		}
	}
}

func TestMapSourceIssueToTask(t *testing.T) {
	issue := SourceIssue{
		Source:      "linear",
		SourceID:    "BIG-102",
		Title:       "Implement prod model",
		Description: "desc",
		Labels:      []string{"p0"},
		Priority:    "P0",
		State:       "Todo",
		Links:       map[string]string{"issue": "https://linear.app/openagi/issue/BIG-102"},
	}
	task := MapSourceIssueToTask(issue)
	if task.ID != "BIG-102" {
		t.Fatalf("expected mapped task ID BIG-102, got %q", task.ID)
	}
	if task.Priority != int(domain.PriorityP0) {
		t.Fatalf("expected mapped task priority %d, got %d", domain.PriorityP0, task.Priority)
	}
	if task.Source != "linear" {
		t.Fatalf("expected mapped task source linear, got %q", task.Source)
	}
	if task.State != domain.TaskQueued {
		t.Fatalf("expected mapped task state queued, got %q", task.State)
	}
	if task.RiskLevel != domain.RiskHigh {
		t.Fatalf("expected prod title to raise risk, got %q", task.RiskLevel)
	}
	if task.Metadata["source_state"] != "Todo" {
		t.Fatalf("expected source state metadata Todo, got %q", task.Metadata["source_state"])
	}
	if task.Metadata["source_id"] != "BIG-102" {
		t.Fatalf("expected source ID metadata BIG-102, got %q", task.Metadata["source_id"])
	}
	if task.Metadata["issue_url"] != "https://linear.app/openagi/issue/BIG-102" {
		t.Fatalf("expected issue_url metadata to be preserved, got %q", task.Metadata["issue_url"])
	}
	if task.TraceID != "BIG-102" {
		t.Fatalf("expected trace ID BIG-102, got %q", task.TraceID)
	}
}

func TestMapSourceIssueToTaskAppliesFallbacksAndGitHubTooling(t *testing.T) {
	issue := SourceIssue{
		Source:      "",
		SourceID:    "",
		Title:       "Fix repo drift",
		Description: "desc",
		State:       "Resolved",
		Links:       map[string]string{"issue": "https://github.com/OpenAGIs/BigClaw/issues/7"},
	}
	task := MapSourceIssueToTask(issue)
	if task.ID != "unknown:fix-repo-drift" {
		t.Fatalf("expected fallback task ID unknown:fix-repo-drift, got %q", task.ID)
	}
	if task.Source != "unknown" {
		t.Fatalf("expected blank source to normalize to unknown, got %q", task.Source)
	}
	if task.TraceID != task.ID {
		t.Fatalf("expected trace ID to match fallback ID, got %q", task.TraceID)
	}
	if task.State != domain.TaskSucceeded {
		t.Fatalf("expected resolved state to map to succeeded, got %q", task.State)
	}
	if task.RequiredTools[0] != "connector" {
		t.Fatalf("expected unknown source to use connector tooling, got %#v", task.RequiredTools)
	}
}

func TestMapSourceIssueToTaskUsesGitHubTooling(t *testing.T) {
	issue := SourceIssue{
		Source:   "github",
		SourceID: "OpenAGIs/BigClaw#8",
		Title:    "Fix sync script",
		State:    "In Progress",
	}
	task := MapSourceIssueToTask(issue)
	if task.RequiredTools[0] != "github" {
		t.Fatalf("expected github source to require github tooling, got %#v", task.RequiredTools)
	}
	if task.State != domain.TaskRunning {
		t.Fatalf("expected in-progress source state to map to running, got %q", task.State)
	}
}
