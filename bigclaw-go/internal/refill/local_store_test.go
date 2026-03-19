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

func TestLocalIssueStoreAddCommentAppendsAndRefreshesUpdatedAt(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "comments": [
        {"body": "seed comment", "created_at": "2026-03-18T09:00:00Z"}
      ],
      "updated_at": "2026-03-18T09:00:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	comment, err := store.AddComment("BIG-GOM-307", "added progress note", time.Date(2026, 3, 18, 16, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}
	if comment.Body != "added progress note" || comment.CreatedAt != "2026-03-18T16:30:00Z" {
		t.Fatalf("unexpected comment payload: %+v", comment)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !strings.Contains(string(body), `"body": "seed comment"`) || !strings.Contains(string(body), `"body": "added progress note"`) {
		t.Fatalf("expected both comments in store, got %s", string(body))
	}
	if !strings.Contains(string(body), `"updated_at": "2026-03-18T16:30:00Z"`) {
		t.Fatalf("expected updated_at refresh, got %s", string(body))
	}
}
