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
