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

func TestParallelIssueQueueRenderMarkdownInfersRecentBatchesAndZeroTime(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			Policy: struct {
				TargetInProgress  int      `json:"target_in_progress"`
				ActivateStateID   string   `json:"activate_state_id"`
				ActivateStateName string   `json:"activate_state_name"`
				RefillStates      []string `json:"refill_states"`
				BlockedReason     string   `json:"blocked_reason,omitempty"`
			}{
				TargetInProgress:  2,
				ActivateStateName: "In Progress",
				RefillStates:      []string{"Todo", "Backlog"},
			},
			IssueOrder: []string{"BIG-PAR-414", "BIG-PAR-415", "BIG-PAR-416"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-414", Title: "queue edge coverage", Status: "Done"},
				{Identifier: "BIG-PAR-415", Title: "local tracker helper edge coverage", Status: "In Progress"},
				{Identifier: "BIG-PAR-416", Status: "Todo"},
			},
		},
	}

	text := queue.RenderMarkdown(time.Time{})
	if !strings.Contains(text, "Current repo tranche status as of ") {
		t.Fatalf("expected generated date header in markdown, got %s", text)
	}
	if !strings.Contains(text, "`BIG-PAR-415` — local tracker helper edge coverage") {
		t.Fatalf("expected inferred active slice in markdown, got %s", text)
	}
	if !strings.Contains(text, "`BIG-PAR-416`") {
		t.Fatalf("expected inferred standby slice in markdown, got %s", text)
	}
	if !strings.Contains(text, "`BIG-PAR-414` — queue edge coverage") {
		t.Fatalf("expected inferred completed slice in markdown, got %s", text)
	}
	if !strings.Contains(text, "`queue_runnable=2`") {
		t.Fatalf("expected runnable count in markdown, got %s", text)
	}
}

func TestParallelIssueQueueRenderMarkdownUsesExplicitRecentBatches(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			Policy: struct {
				TargetInProgress  int      `json:"target_in_progress"`
				ActivateStateID   string   `json:"activate_state_id"`
				ActivateStateName string   `json:"activate_state_name"`
				RefillStates      []string `json:"refill_states"`
				BlockedReason     string   `json:"blocked_reason,omitempty"`
			}{
				TargetInProgress:  2,
				ActivateStateName: "In Progress",
				RefillStates:      []string{"Todo", "Backlog"},
			},
			IssueOrder: []string{"BIG-PAR-425", "BIG-PAR-426", "BIG-PAR-427", "BIG-PAR-428"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-425", Title: "Active primer", Status: "Todo"},
				{Identifier: "BIG-PAR-426", Title: "Active sequel", Status: "Todo"},
				{Identifier: "BIG-PAR-427", Title: "Standby future", Status: "Todo"},
				{Identifier: "BIG-PAR-428", Title: "Completed legacy", Status: "Todo"},
			},
			RecentBatches: struct {
				Completed []string `json:"completed"`
				Active    []string `json:"active"`
				Standby   []string `json:"standby"`
			}{
				Completed: []string{"BIG-PAR-428"},
				Active:    []string{"BIG-PAR-425", "BIG-PAR-426"},
				Standby:   []string{"BIG-PAR-427"},
			},
		},
	}

	text := queue.RenderMarkdown(time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC))
	if !strings.Contains(text, "  - active slices: `BIG-PAR-425` — Active primer; `BIG-PAR-426` — Active sequel") {
		t.Fatalf("expected explicit active slices bucket, got %s", text)
	}
	if !strings.Contains(text, "  - standby slices: `BIG-PAR-427` — Standby future") {
		t.Fatalf("expected explicit standby slices bucket, got %s", text)
	}
	if !strings.Contains(text, "  - recently completed slices: `BIG-PAR-428` — Completed legacy") {
		t.Fatalf("expected explicit recently completed bucket, got %s", text)
	}
}

func TestParallelIssueQueueRenderMarkdownCompletedHistoryHighlightsTerminalStatuses(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-429", "BIG-PAR-430"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-429", Title: "Terminal history", Status: "Done"},
				{Identifier: "BIG-PAR-430", Title: "Mutable next", Status: "In Progress"},
			},
		},
	}

	text := queue.RenderMarkdown(time.Time{})
	if !strings.Contains(text, "- Completed slices:\n  - `BIG-PAR-429` — Terminal history\n") {
		t.Fatalf("expected completed history entry, got %s", text)
	}
	if strings.Contains(text, "- Completed slices:\n  - `BIG-PAR-430`") {
		t.Fatalf("did not expect non-terminal identifier in completed history, got %s", text)
	}
}

func TestParallelIssueQueueSaveMarkdownFailsWhenParentPathIsAFile(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "markdown-parent")
	if err := os.WriteFile(parentFile, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("write markdown parent fixture: %v", err)
	}

	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-418"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-418", Title: "Add refill save-parent failure coverage", Status: "In Progress"},
			},
		},
	}

	written, err := queue.SaveMarkdown(filepath.Join(parentFile, "queue.md"), time.Date(2026, 3, 25, 19, 35, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected save markdown to fail when parent path is a file")
	}
	if written {
		t.Fatal("expected failed markdown save to report written=false")
	}
}
