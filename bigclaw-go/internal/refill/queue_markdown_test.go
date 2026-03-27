package refill

import (
	"errors"
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
	if !strings.Contains(text, "`bash scripts/ops/bigclawctl issue list`") {
		t.Fatalf("expected direct bigclawctl issue command in markdown, got %s", text)
	}
	if !strings.Contains(text, "`bash scripts/ops/bigclawctl symphony`") {
		t.Fatalf("expected direct bigclawctl symphony command in markdown, got %s", text)
	}
	if !strings.Contains(text, "`bash scripts/ops/bigclaw-issue ...`") {
		t.Fatalf("expected compatibility wrapper note in markdown, got %s", text)
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

func TestParallelIssueQueueSaveMarkdownPropagatesAbsPathFailure(t *testing.T) {
	queue := &ParallelIssueQueue{}

	originalAbsPath := queueMarkdownAbsPath
	originalMkdirAll := queueMarkdownMkdirAll
	t.Cleanup(func() {
		queueMarkdownAbsPath = originalAbsPath
		queueMarkdownMkdirAll = originalMkdirAll
	})

	absErr := errors.New("markdown abs failure")
	queueMarkdownAbsPath = func(path string) (string, error) {
		return "", absErr
	}
	queueMarkdownMkdirAll = func(path string, perm os.FileMode) error {
		t.Fatal("did not expect mkdir after abs-path failure")
		return nil
	}

	if written, err := queue.SaveMarkdown("queue.md", time.Date(2026, 3, 25, 19, 50, 0, 0, time.UTC)); !errors.Is(err, absErr) || written {
		t.Fatalf("expected abs-path failure, got written=%v err=%v", written, err)
	}
}

func TestParallelIssueQueueSaveMarkdownPropagatesMkdirAllFailure(t *testing.T) {
	queue := &ParallelIssueQueue{}

	originalAbsPath := queueMarkdownAbsPath
	originalMkdirAll := queueMarkdownMkdirAll
	originalCreateTemp := queueMarkdownCreateTemp
	t.Cleanup(func() {
		queueMarkdownAbsPath = originalAbsPath
		queueMarkdownMkdirAll = originalMkdirAll
		queueMarkdownCreateTemp = originalCreateTemp
	})

	mkdirErr := errors.New("markdown mkdir failure")
	queueMarkdownAbsPath = func(path string) (string, error) {
		return filepath.Join(t.TempDir(), "nested", "queue.md"), nil
	}
	queueMarkdownMkdirAll = func(path string, perm os.FileMode) error {
		return mkdirErr
	}
	queueMarkdownCreateTemp = func(dir string, pattern string) (tempFile, error) {
		t.Fatal("did not expect temp file creation after mkdir failure")
		return nil, nil
	}

	if written, err := queue.SaveMarkdown("queue.md", time.Date(2026, 3, 25, 19, 51, 0, 0, time.UTC)); !errors.Is(err, mkdirErr) || written {
		t.Fatalf("expected mkdir failure, got written=%v err=%v", written, err)
	}
}

func TestParallelIssueQueueSaveMarkdownPropagatesWriteAndRenameFailures(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-425"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-425", Title: "Make refill queue save failures testable", Status: "In Progress"},
			},
		},
	}

	originalCreateTemp := queueMarkdownCreateTemp
	originalRename := queueMarkdownRename
	t.Cleanup(func() {
		queueMarkdownCreateTemp = originalCreateTemp
		queueMarkdownRename = originalRename
	})

	writeErr := errors.New("write markdown temp file")
	queueMarkdownCreateTemp = func(dir string, pattern string) (tempFile, error) {
		return &stubTempFile{name: filepath.Join(dir, "queue-md-write.tmp"), writeErr: writeErr}, nil
	}
	queueMarkdownRename = func(oldPath string, newPath string) error {
		t.Fatal("did not expect rename after write failure")
		return nil
	}
	if written, err := queue.SaveMarkdown(filepath.Join(t.TempDir(), "queue.md"), time.Date(2026, 3, 25, 20, 0, 0, 0, time.UTC)); !errors.Is(err, writeErr) || written {
		t.Fatalf("expected write failure with written=false, got written=%v err=%v", written, err)
	}

	renameErr := errors.New("rename markdown file")
	targetPath := filepath.Join(t.TempDir(), "queue-rename.md")
	queueMarkdownCreateTemp = func(dir string, pattern string) (tempFile, error) {
		return &stubTempFile{name: filepath.Join(dir, "queue-md-rename.tmp")}, nil
	}
	queueMarkdownRename = func(oldPath string, newPath string) error {
		if oldPath != filepath.Join(filepath.Dir(targetPath), "queue-md-rename.tmp") || newPath != targetPath {
			t.Fatalf("unexpected rename paths old=%q new=%q", oldPath, newPath)
		}
		return renameErr
	}
	if written, err := queue.SaveMarkdown(targetPath, time.Date(2026, 3, 25, 20, 1, 0, 0, time.UTC)); !errors.Is(err, renameErr) || written {
		t.Fatalf("expected rename failure with written=false, got written=%v err=%v", written, err)
	}
}

func TestParallelIssueQueueSaveMarkdownPropagatesChmodFailure(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-427"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-427", Title: "Add refill temp-save branch coverage", Status: "In Progress"},
			},
		},
	}

	origCreate := queueMarkdownCreateTemp
	origRename := queueMarkdownRename
	defer func() {
		queueMarkdownCreateTemp = origCreate
		queueMarkdownRename = origRename
	}()

	chmodErr := errors.New("markdown chmod failure")
	queueMarkdownCreateTemp = func(dir string, pattern string) (tempFile, error) {
		return &stubTempFile{name: filepath.Join(dir, "queue-md-chmod.tmp"), chmodErr: chmodErr}, nil
	}
	queueMarkdownRename = func(oldPath string, newPath string) error {
		t.Fatalf("did not expect rename after chmod failure")
		return nil
	}

	if written, err := queue.SaveMarkdown(filepath.Join(t.TempDir(), "queue.md"), time.Date(2026, 3, 25, 20, 5, 0, 0, time.UTC)); !errors.Is(err, chmodErr) || written {
		t.Fatalf("expected chmod failure with written=false, got written=%v err=%v", written, err)
	}
}

func TestParallelIssueQueueSaveMarkdownPropagatesCloseFailure(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-427"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-427", Title: "Add refill temp-save branch coverage", Status: "In Progress"},
			},
		},
	}

	originalCreateTemp := queueMarkdownCreateTemp
	originalRename := queueMarkdownRename
	t.Cleanup(func() {
		queueMarkdownCreateTemp = originalCreateTemp
		queueMarkdownRename = originalRename
	})

	closeErr := errors.New("close markdown temp file")
	queueMarkdownCreateTemp = func(dir string, pattern string) (tempFile, error) {
		return &stubTempFile{name: filepath.Join(dir, "queue-md-close.tmp"), closeErr: closeErr}, nil
	}
	queueMarkdownRename = func(oldPath string, newPath string) error {
		t.Fatal("did not expect rename after close failure")
		return nil
	}
	if written, err := queue.SaveMarkdown(filepath.Join(t.TempDir(), "queue-close.md"), time.Date(2026, 3, 25, 20, 2, 0, 0, time.UTC)); !errors.Is(err, closeErr) || written {
		t.Fatalf("expected close failure with written=false, got written=%v err=%v", written, err)
	}
}

func TestParallelIssueQueueSaveMarkdownPropagatesCreateTempFailure(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-431"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-431", Title: "Add refill final save branch coverage", Status: "In Progress"},
			},
		},
	}

	originalCreateTemp := queueMarkdownCreateTemp
	originalRename := queueMarkdownRename
	t.Cleanup(func() {
		queueMarkdownCreateTemp = originalCreateTemp
		queueMarkdownRename = originalRename
	})

	createErr := errors.New("create temp markdown file")
	queueMarkdownCreateTemp = func(dir string, pattern string) (tempFile, error) {
		return nil, createErr
	}
	queueMarkdownRename = func(oldPath string, newPath string) error {
		t.Fatal("did not expect rename after create temp failure")
		return nil
	}
	if written, err := queue.SaveMarkdown(filepath.Join(t.TempDir(), "queue-create.md"), time.Date(2026, 3, 25, 20, 6, 0, 0, time.UTC)); !errors.Is(err, createErr) || written {
		t.Fatalf("expected create temp failure with written=false, got written=%v err=%v", written, err)
	}
}
