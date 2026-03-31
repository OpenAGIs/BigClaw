package observability

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestJSONLAuditSinkWritesEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	sink, err := NewJSONLAuditSink(path)
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	if err := sink.Write(domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", Timestamp: time.Now()}); err != nil {
		t.Fatalf("write event: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(contents), "evt-1") {
		t.Fatalf("expected event id in audit file, got %s", string(contents))
	}
}

func TestMissingRequiredFieldsForEventUsesTopLevelAuditIdentifiers(t *testing.T) {
	missing := MissingRequiredFieldsForEvent(domain.Event{
		ID:     "evt-approval-1",
		Type:   domain.EventType(ApprovalRecordedEvent),
		TaskID: "task-1",
		RunID:  "run-1",
		Payload: map[string]any{
			"approvals":         []string{"eng_lead"},
			"approval_count":    1,
			"acceptance_status": "approved",
		},
	})
	if len(missing) != 0 {
		t.Fatalf("expected top-level task/run identifiers to satisfy audit spec, got %+v", missing)
	}
}

func TestMissingRequiredFieldsForManualTakeoverEventUsesTopLevelAuditIdentifiers(t *testing.T) {
	missing := MissingRequiredFieldsForEvent(domain.Event{
		ID:     "evt-takeover-1",
		Type:   domain.EventType(ManualTakeoverEvent),
		TaskID: "task-takeover-1",
		RunID:  "run-takeover-1",
		Payload: map[string]any{
			"target_team":        "security",
			"reason":             "manual review required",
			"requested_by":       "scheduler",
			"required_approvals": []string{"security-review"},
		},
	})
	if len(missing) != 0 {
		t.Fatalf("expected manual takeover event to satisfy audit spec, got %+v", missing)
	}
}

func TestMissingRequiredFieldsForBudgetOverrideAndFlowHandoffEvents(t *testing.T) {
	cases := []domain.Event{
		{
			ID:     "evt-budget-1",
			Type:   domain.EventType(BudgetOverrideEvent),
			TaskID: "task-budget-1",
			RunID:  "run-budget-1",
			Payload: map[string]any{
				"requested_budget": 120.0,
				"approved_budget":  150.0,
				"override_actor":   "finance-controller",
				"reason":           "approved additional analytics validation spend",
			},
		},
		{
			ID:     "evt-handoff-1",
			Type:   domain.EventType(FlowHandoffEvent),
			TaskID: "task-handoff-1",
			RunID:  "run-handoff-1",
			Payload: map[string]any{
				"source_stage":       "scheduler",
				"target_team":        "operations",
				"reason":             "premium tier required",
				"collaboration_mode": "tier-limited",
			},
		},
	}

	for _, event := range cases {
		if missing := MissingRequiredFieldsForEvent(event); len(missing) != 0 {
			t.Fatalf("expected %s to satisfy audit spec, got %+v", event.Type, missing)
		}
	}
}

func TestMissingRequiredFieldsForEventReturnsSpecGaps(t *testing.T) {
	missing := MissingRequiredFieldsForEvent(domain.Event{
		ID:     "evt-approval-2",
		Type:   domain.EventType(ApprovalRecordedEvent),
		TaskID: "task-2",
		RunID:  "run-2",
		Payload: map[string]any{
			"approvals": []string{"eng_lead"},
		},
	})
	if !reflect.DeepEqual(missing, []string{"approval_count", "acceptance_status"}) {
		t.Fatalf("unexpected missing fields: %+v", missing)
	}
}

func TestJSONLAuditSinkRejectsMalformedKnownAuditEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	sink, err := NewJSONLAuditSink(path)
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	err = sink.Write(domain.Event{
		ID:     "evt-approval-3",
		Type:   domain.EventType(ApprovalRecordedEvent),
		TaskID: "task-3",
		RunID:  "run-3",
		Payload: map[string]any{
			"approvals": []string{"eng_lead"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "approval_count") || !strings.Contains(err.Error(), "acceptance_status") {
		t.Fatalf("expected required field validation error, got %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("expected malformed event to avoid file writes, stat err=%v", statErr)
	}
}
