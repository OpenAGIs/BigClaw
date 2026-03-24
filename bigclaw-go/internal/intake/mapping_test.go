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
		"running":      domain.TaskRunning,
		"Blocked":      domain.TaskBlocked,
		"stopped":      domain.TaskBlocked,
		"Closed":       domain.TaskSucceeded,
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
}

func TestMapClawHostSourceIssueToTaskPreservesInventoryMetadata(t *testing.T) {
	issue := SourceIssue{
		Source:      "clawhost",
		SourceID:    "openagi/claw-sales-west",
		Title:       "ClawHost claw sales-west (running)",
		Description: "Provider hetzner plan cpx11 in ash with 2 agents.",
		Labels:      []string{"clawhost", "inventory"},
		Priority:    "P1",
		State:       "running",
		Links: map[string]string{
			"issue":     "https://clawhost.cloud/claws/claw-sales-west",
			"dashboard": "https://clawhost.cloud/claws/claw-sales-west",
		},
		Metadata: map[string]string{
			"tenant_id":           "openagi-owner-a",
			"inventory_kind":      "claw",
			"provider":            "hetzner",
			"provider_server_id":  "srv-hetzner-431",
			"domain":              "sales-west.clawhost.cloud",
			"agent_count":         "2",
			"running_agent_count": "2",
			"control_plane":       "clawhost",
		},
	}
	task := MapSourceIssueToTask(issue)
	if task.Source != "clawhost" || task.State != domain.TaskRunning {
		t.Fatalf("unexpected mapped clawhost task: %+v", task)
	}
	if task.TenantID != "openagi-owner-a" {
		t.Fatalf("expected clawhost tenant mapped, got %+v", task)
	}
	if len(task.RequiredTools) != 2 || task.RequiredTools[0] != "clawhost" {
		t.Fatalf("expected clawhost required tools, got %+v", task.RequiredTools)
	}
	if task.Metadata["domain"] != "sales-west.clawhost.cloud" || task.Metadata["provider_server_id"] != "srv-hetzner-431" || task.Metadata["control_plane"] != "clawhost" {
		t.Fatalf("expected clawhost inventory metadata preserved, got %+v", task.Metadata)
	}
}
