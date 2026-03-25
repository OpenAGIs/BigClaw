package refill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeLocalIssueMapsSkipsNonMapEntries(t *testing.T) {
	items := []any{
		map[string]any{"identifier": "BIG-PAR-399"},
		"skip-me",
		42,
		map[string]any{"identifier": "BIG-PAR-400"},
	}

	issues := normalizeLocalIssueMaps(items)
	if len(issues) != 2 {
		t.Fatalf("expected only map entries to survive, got %+v", issues)
	}
	if issues[0]["identifier"] != "BIG-PAR-399" || issues[1]["identifier"] != "BIG-PAR-400" {
		t.Fatalf("unexpected normalized issues: %+v", issues)
	}
}

func TestIssueCommentListNormalizesCommentCollections(t *testing.T) {
	comments := issueCommentList([]any{
		map[string]any{"body": "kept"},
		"skip-me",
		map[string]any{"body": "also kept"},
	})
	if len(comments) != 2 {
		t.Fatalf("expected only map comments, got %+v", comments)
	}
	if comments[0]["body"] != "kept" || comments[1]["body"] != "also kept" {
		t.Fatalf("unexpected normalized comments: %+v", comments)
	}

	empty := issueCommentList("not-a-list")
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected non-list comments to normalize to empty slice, got %+v", empty)
	}
}

func TestLocalIssueScalarHelpersNormalizeTypes(t *testing.T) {
	issue := map[string]any{
		"priority_float":       float64(3),
		"priority_int":         4,
		"priority_bad":         "high",
		"assigned_true":        true,
		"assigned_bad":         "yes",
		"labels_any":           []any{" ops ", "", 12},
		"labels_strings":       []string{" one ", "", "two"},
		"labels_bad":           99,
		"labels_nil_explicit":  nil,
	}

	if got := mapInt(issue, "priority_float"); got != 3 {
		t.Fatalf("expected float priority to coerce to 3, got %d", got)
	}
	if got := mapInt(issue, "priority_int"); got != 4 {
		t.Fatalf("expected int priority to round-trip, got %d", got)
	}
	if got := mapInt(issue, "priority_bad"); got != 0 {
		t.Fatalf("expected invalid priority to return zero, got %d", got)
	}
	if got := mapInt(issue, "missing_priority"); got != 0 {
		t.Fatalf("expected missing priority to return zero, got %d", got)
	}

	if !mapBool(issue, "assigned_true") {
		t.Fatal("expected bool helper to preserve true")
	}
	if mapBool(issue, "assigned_bad") {
		t.Fatal("expected non-bool value to return false")
	}
	if mapBool(issue, "missing_assigned") {
		t.Fatal("expected missing bool value to return false")
	}

	labelsAny := mapStringSlice(issue, "labels_any")
	if len(labelsAny) != 2 || labelsAny[0] != "ops" || labelsAny[1] != "12" {
		t.Fatalf("expected []any labels to normalize, got %+v", labelsAny)
	}
	labelsStrings := mapStringSlice(issue, "labels_strings")
	if len(labelsStrings) != 2 || labelsStrings[0] != "one" || labelsStrings[1] != "two" {
		t.Fatalf("expected []string labels to normalize, got %+v", labelsStrings)
	}
	if got := mapStringSlice(issue, "labels_bad"); got != nil {
		t.Fatalf("expected invalid label type to return nil, got %+v", got)
	}
	if got := mapStringSlice(issue, "labels_nil_explicit"); got != nil {
		t.Fatalf("expected explicit nil label value to return nil, got %+v", got)
	}
	if got := mapStringSlice(issue, "missing_labels"); got != nil {
		t.Fatalf("expected missing labels to return nil, got %+v", got)
	}
}

func TestLocalIssueStoreUpdateIssueStatePreservesExtraFields(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "Todo",
      "labels": ["go", "tooling"],
      "assigned_to_worker": true,
      "branch_name": "feat/big-gom-307",
      "comments": [
        {"body": "seed comment", "created_at": "2026-03-18T09:00:00Z"}
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	updatedState, err := store.UpdateIssueState("BIG-GOM-307", "In Progress", time.Date(2026, 3, 18, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("update issue state: %v", err)
	}
	if updatedState != "In Progress" {
		t.Fatalf("expected updated state, got %q", updatedState)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !strings.Contains(string(body), `"branch_name": "feat/big-gom-307"`) {
		t.Fatalf("expected extra fields to be preserved, got %s", string(body))
	}
	if !strings.Contains(string(body), `"comments": [`) {
		t.Fatalf("expected comments to be preserved, got %s", string(body))
	}
	if !strings.Contains(string(body), `"updated_at": "2026-03-18T15:00:00Z"`) {
		t.Fatalf("expected updated_at to refresh, got %s", string(body))
	}
}

func TestLocalIssueStoreIssueStatesFiltersRequestedStates(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-gom-301", "identifier": "BIG-GOM-301", "state": "In Progress"},
    {"id": "big-gom-303", "identifier": "BIG-GOM-303", "state": "Todo"},
    {"id": "big-gom-305", "identifier": "BIG-GOM-305", "state": "Backlog"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	issues := store.IssueStates([]string{"In Progress", "Todo"})
	if len(issues) != 2 {
		t.Fatalf("expected 2 active issues, got %d", len(issues))
	}
	if issues[0].Identifier != "BIG-GOM-301" || issues[1].Identifier != "BIG-GOM-303" {
		t.Fatalf("unexpected filtered issues: %+v", issues)
	}
}

func TestLocalIssueStoreAddCommentAppendsAndUpdatesTimestamp(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "comments": [
        {"author": "codex", "created_at": "2026-03-18T09:00:00Z", "body": "seed comment"}
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	if err := store.AddComment("BIG-GOM-307", LocalIssueComment{
		Author:    "codex",
		Body:      "validation passed",
		CreatedAt: time.Date(2026, 3, 20, 10, 45, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("add comment: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if strings.Count(text, `"body":`) != 2 {
		t.Fatalf("expected appended comment list, got %s", text)
	}
	if !strings.Contains(text, `"body": "validation passed"`) {
		t.Fatalf("expected appended comment body, got %s", text)
	}
	if !strings.Contains(text, `"updated_at": "2026-03-20T10:45:00Z"`) {
		t.Fatalf("expected updated_at refresh, got %s", text)
	}
}

func TestLocalIssueStoreSaveDoesNotEscapeArrowTokens(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "Todo",
      "comments": [
        {"author": "codex", "created_at": "2026-03-18T09:00:00Z", "body": "seed -> ok"}
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	if _, err := store.UpdateIssueState("BIG-GOM-307", "In Progress", time.Date(2026, 3, 18, 15, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("update issue state: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "seed -> ok") {
		t.Fatalf("expected arrow token to persist, got %s", text)
	}
	if strings.Contains(text, `\\u003e`) {
		t.Fatalf("expected no HTML escaping, got %s", text)
	}
}

func TestLocalIssueStoreReloadRefreshesInMemorySnapshot(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-par-243", "identifier": "BIG-PAR-243", "state": "Todo"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	if issues := store.IssueStates([]string{"Todo"}); len(issues) != 1 {
		t.Fatalf("expected initial todo issue, got %+v", issues)
	}

	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-par-243", "identifier": "BIG-PAR-243", "state": "Done"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("rewrite local issue store: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("reload local issue store: %v", err)
	}

	if issues := store.IssueStates([]string{"Todo"}); len(issues) != 0 {
		t.Fatalf("expected no todo issues after reload, got %+v", issues)
	}
	if issues := store.IssueStates([]string{"Done"}); len(issues) != 1 || issues[0].Identifier != "BIG-PAR-243" {
		t.Fatalf("expected reloaded done issue, got %+v", issues)
	}
}

func TestLocalIssueStoreAddCommentReloadsLatestStateBeforeSaving(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-241",
      "identifier": "BIG-PAR-241",
      "title": "Serialize local tracker writes with an explicit lock",
      "state": "In Progress",
      "comments": []
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	storeA, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store A: %v", err)
	}
	storeB, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store B: %v", err)
	}

	if err := storeA.AddComment("BIG-PAR-241", LocalIssueComment{
		Author:    "codex",
		Body:      "first writer",
		CreatedAt: time.Date(2026, 3, 23, 2, 45, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("add comment with store A: %v", err)
	}
	if err := storeB.AddComment("BIG-PAR-241", LocalIssueComment{
		Author:    "codex",
		Body:      "second writer",
		CreatedAt: time.Date(2026, 3, 23, 2, 45, 1, 0, time.UTC),
	}); err != nil {
		t.Fatalf("add comment with store B: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"body": "first writer"`) || !strings.Contains(text, `"body": "second writer"`) {
		t.Fatalf("expected both comments to persist after stale reload protection, got %s", text)
	}
}

func TestLocalIssueStoreAddCommentRetriesTransientLockFile(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-241",
      "identifier": "BIG-PAR-241",
      "title": "Serialize local tracker writes with an explicit lock",
      "state": "In Progress"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	lockPath := storePath + ".lock"
	if err := os.WriteFile(lockPath, []byte("held"), 0o644); err != nil {
		t.Fatalf("write transient lock file: %v", err)
	}
	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = os.Remove(lockPath)
	}()

	if err := store.AddComment("BIG-PAR-241", LocalIssueComment{
		Author:    "codex",
		Body:      "lock released",
		CreatedAt: time.Date(2026, 3, 23, 2, 46, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("expected transient lock retry to succeed, got %v", err)
	}
}
