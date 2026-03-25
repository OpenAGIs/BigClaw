package refill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParallelIssueQueueSaveMarkdownNoopAndReadErrorPaths(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-410"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-410", Title: "Add refill markdown helper coverage", Status: "In Progress"},
			},
		},
	}

	markdownPath := filepath.Join(t.TempDir(), "queue.md")
	generatedAt := time.Date(2026, 3, 25, 18, 50, 0, 0, time.UTC)
	written, err := queue.SaveMarkdown(markdownPath, generatedAt)
	if err != nil {
		t.Fatalf("initial save markdown: %v", err)
	}
	if !written {
		t.Fatal("expected first markdown save to write output")
	}

	written, err = queue.SaveMarkdown(markdownPath, generatedAt)
	if err != nil {
		t.Fatalf("repeat save markdown: %v", err)
	}
	if written {
		t.Fatal("expected identical markdown save to be a no-op")
	}

	dirPath := filepath.Join(t.TempDir(), "existing-directory")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("mkdir markdown directory fixture: %v", err)
	}
	if written, err := queue.SaveMarkdown(dirPath, generatedAt); err == nil || written {
		t.Fatalf("expected directory read error from save markdown, got written=%v err=%v", written, err)
	}
}

func TestQueueMarkdownHelperFunctions(t *testing.T) {
	records := map[string]IssueRecord{
		"BIG-PAR-409": {Identifier: "BIG-PAR-409"},
		"BIG-PAR-410": {Identifier: "BIG-PAR-410", Title: "Add refill markdown helper coverage"},
	}

	var out strings.Builder
	writeIssueBucket(&out, "active slices", []string{"BIG-PAR-409", "BIG-PAR-410"}, records)
	text := out.String()
	if !strings.Contains(text, "active slices") {
		t.Fatalf("expected label in bucket output, got %s", text)
	}
	if !strings.Contains(text, "`BIG-PAR-409`") {
		t.Fatalf("expected identifier-only item in bucket output, got %s", text)
	}
	if !strings.Contains(text, "`BIG-PAR-410` — Add refill markdown helper coverage") {
		t.Fatalf("expected titled item in bucket output, got %s", text)
	}

	identifiers := []string{"BIG-PAR-407", "BIG-PAR-408", "BIG-PAR-409", "BIG-PAR-410"}
	if got := tailIdentifiers(identifiers, 2); !equalStringSlices(got, []string{"BIG-PAR-409", "BIG-PAR-410"}) {
		t.Fatalf("unexpected tail identifiers: %+v", got)
	}
	if got := tailIdentifiers(identifiers, 0); !equalStringSlices(got, identifiers) {
		t.Fatalf("expected zero-limit tail to return full slice copy, got %+v", got)
	}
	if got := headIdentifiers(identifiers, 2); !equalStringSlices(got, []string{"BIG-PAR-407", "BIG-PAR-408"}) {
		t.Fatalf("unexpected head identifiers: %+v", got)
	}
	if got := headIdentifiers(identifiers, 0); !equalStringSlices(got, identifiers) {
		t.Fatalf("expected zero-limit head to return full slice copy, got %+v", got)
	}
}
