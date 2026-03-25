package refill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

func TestLocalIssueStoreUpdateIssueStateTrimsState(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-391",
      "identifier": "BIG-PAR-391",
      "title": "Trim state test",
      "state": "Todo"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	if _, err := store.UpdateIssueState("BIG-PAR-391", "  Backlog  ", time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("update issue state: %v", err)
	}
	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !strings.Contains(string(body), `"state": "Backlog"`) {
		t.Fatalf("expected trimmed state, got %s", string(body))
	}
}

func TestLocalIssueStoreUpdateIssueStateDefaultsBlank(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-392",
      "identifier": "BIG-PAR-392",
      "title": "Blank state test",
      "state": "Done"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	if _, err := store.UpdateIssueState("BIG-PAR-392", "   ", time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("update issue state: %v", err)
	}
	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !strings.Contains(string(body), `"state": "Todo"`) {
		t.Fatalf("expected defaulted state to Todo, got %s", string(body))
	}
}

func TestLocalIssueStoreUpdateIssueStateIgnoresEquivalentSpellings(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-399",
      "identifier": "BIG-PAR-399",
      "title": "Equivalent state test",
      "state": "in progress.",
      "updated_at": "2026-03-25T18:20:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	updatedState, err := store.UpdateIssueState("BIG-PAR-399", "In Progress", time.Date(2026, 3, 25, 18, 22, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("update issue state: %v", err)
	}
	if updatedState != "in progress." {
		t.Fatalf("expected original equivalent spelling to be preserved, got %q", updatedState)
	}
	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"state": "in progress."`) {
		t.Fatalf("expected equivalent spelling to remain unchanged, got %s", text)
	}
	if !strings.Contains(text, `"updated_at": "2026-03-25T18:20:00Z"`) {
		t.Fatalf("expected unchanged updated_at for equivalent spelling, got %s", text)
	}
}

func TestLocalIssueStoreCreateIssueCanonicalizesEquivalentBuiltInStates(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	created, err := store.CreateIssue(LocalIssueCreateParams{
		Identifier: "BIG-PAR-400",
		Title:      "Canonicalize create state",
		State:      "todo.",
		CreatedAt:  time.Date(2026, 3, 25, 18, 35, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if created.State != "Todo" {
		t.Fatalf("expected canonical Todo state, got %q", created.State)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !strings.Contains(string(body), `"state": "Todo"`) {
		t.Fatalf("expected persisted canonical Todo state, got %s", string(body))
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

func TestLocalIssueStoreIssueStatesNormalizesStateNames(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-par-385", "identifier": "BIG-PAR-385", "state": "in progress."},
    {"id": "big-par-386", "identifier": "BIG-PAR-386", "state": "TODO"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	issues := store.IssueStates([]string{"In Progress", "todo."})
	if len(issues) != 2 {
		t.Fatalf("expected normalized state match, got %+v", issues)
	}
	if issues[0].Identifier != "BIG-PAR-385" || issues[1].Identifier != "BIG-PAR-386" {
		t.Fatalf("unexpected filtered issues after normalization: %+v", issues)
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
