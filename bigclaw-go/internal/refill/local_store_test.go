package refill

import (
	"errors"
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

func TestLocalIssueStoreIssuesFiltersRequestedStatesAndClonesMaps(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-gom-301", "identifier": "BIG-GOM-301", "title": "Domain parity", "state": "In Progress"},
    {"id": "big-gom-303", "identifier": "BIG-GOM-303", "title": "Workflow loop parity", "state": "Todo"},
    {"id": "big-gom-305", "identifier": "BIG-GOM-305", "title": "Operations views", "state": "Backlog"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	issues := store.Issues([]string{"In Progress", "Todo"})
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if mapString(issues[0], "identifier") != "BIG-GOM-301" || mapString(issues[1], "identifier") != "BIG-GOM-303" {
		t.Fatalf("unexpected issues: %+v", issues)
	}

	issues[0]["state"] = "Done"
	original, err := store.Issue("BIG-GOM-301")
	if err != nil {
		t.Fatalf("reload original issue: %v", err)
	}
	if mapString(original, "state") != "In Progress" {
		t.Fatalf("expected returned list to be cloned, got %+v", original)
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

func TestLocalIssueStoreCreateIssueAppendsWithDefaults(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	issue, err := store.CreateIssue(LocalIssueCreateInput{
		ID:               "big-gom-309",
		Identifier:       "BIG-GOM-309",
		Title:            "New tracker issue",
		Description:      "Create issues without Symphony",
		Priority:         2,
		Labels:           []string{"go-mainline", "tooling", "tooling"},
		AssignedToWorker: true,
	}, time.Date(2026, 3, 20, 9, 45, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if mapString(issue, "state") != "Backlog" {
		t.Fatalf("expected default backlog state, got %+v", issue)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !strings.Contains(string(body), `"identifier": "BIG-GOM-309"`) || !strings.Contains(string(body), `"created_at": "2026-03-20T09:45:00Z"`) {
		t.Fatalf("expected created issue in store, got %s", string(body))
	}
	if strings.Count(string(body), `"tooling"`) != 1 {
		t.Fatalf("expected duplicate labels to be collapsed, got %s", string(body))
	}
}

func TestLocalIssueStoreCreateIssueDerivesIDFromIdentifier(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	issue, err := store.CreateIssue(LocalIssueCreateInput{
		Identifier:       "BIG-GOM-310",
		Title:            "Derive issue id",
		AssignedToWorker: true,
	}, time.Date(2026, 3, 20, 10, 15, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if mapString(issue, "id") != "big-gom-310" {
		t.Fatalf("expected derived id, got %+v", issue)
	}
}

func TestLocalIssueStoreCreateIssueRejectsDuplicateIdentifiers(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-gom-307", "identifier": "BIG-GOM-307", "title": "Toolchain migration", "state": "In Progress"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	_, err = store.CreateIssue(LocalIssueCreateInput{
		ID:               "big-gom-307",
		Identifier:       "BIG-GOM-307",
		Title:            "Duplicate issue",
		AssignedToWorker: true,
	}, time.Date(2026, 3, 20, 9, 45, 0, 0, time.UTC))
	if !errors.Is(err, ErrLocalIssueAlreadyExists) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestDefaultIssueIDFallsBackToNormalizedIdentifier(t *testing.T) {
	if got := defaultIssueID("", " BIG/GOM 311 "); got != "big-gom-311" {
		t.Fatalf("unexpected normalized id: %q", got)
	}
}
