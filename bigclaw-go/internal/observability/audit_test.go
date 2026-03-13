package observability

import (
	"os"
	"path/filepath"
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
