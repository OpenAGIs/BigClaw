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

func TestLocalIssueStoreCreateIssueAppendsStructuredRecord(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "created_at": "2026-03-18T09:00:00Z",
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
	created, err := store.CreateIssue(LocalIssueCreateInput{
		Identifier:       "BIG-GOM-309",
		Title:            "Go-native local issue creation",
		Description:      "Add a Go CLI create flow for local tracker slices",
		Priority:         2,
		State:            "Todo",
		Labels:           []string{"go-mainline", "tooling"},
		AssignedToWorker: true,
	}, time.Date(2026, 3, 20, 10, 15, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if created.ID != "big-gom-309" || created.Identifier != "BIG-GOM-309" || created.StateName != "Todo" {
		t.Fatalf("unexpected created issue payload: %+v", created)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"identifier": "BIG-GOM-309"`) {
		t.Fatalf("expected identifier in local issue store, got %s", text)
	}
	if !strings.Contains(text, `"id": "big-gom-309"`) {
		t.Fatalf("expected derived issue id in local issue store, got %s", text)
	}
	if !strings.Contains(text, "\"labels\": [\n        \"go-mainline\",\n        \"tooling\"\n      ]") {
		t.Fatalf("expected labels in local issue store, got %s", text)
	}
	if !strings.Contains(text, `"assigned_to_worker": true`) {
		t.Fatalf("expected assigned_to_worker in local issue store, got %s", text)
	}
	if !strings.Contains(text, `"created_at": "2026-03-20T10:15:00Z"`) {
		t.Fatalf("expected created_at timestamp, got %s", text)
	}
}

func TestLocalIssueStoreCreateIssueRejectsDuplicateIdentifier(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
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
	_, err = store.CreateIssue(LocalIssueCreateInput{
		Identifier:       "BIG-GOM-307",
		Title:            "Duplicate",
		Priority:         3,
		AssignedToWorker: true,
	}, time.Date(2026, 3, 20, 10, 15, 0, 0, time.UTC))
	if !errors.Is(err, ErrLocalIssueAlreadyExists) {
		t.Fatalf("expected duplicate issue error, got %v", err)
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

func TestLocalIssueStoreListIssuesFiltersAndDecodesFields(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-301",
      "identifier": "BIG-GOM-301",
      "title": "Domain parity",
      "priority": 1,
      "state": "In Progress",
      "labels": ["go-mainline", "domain"],
      "created_at": "2026-03-18T09:00:00Z",
      "updated_at": "2026-03-18T09:30:00Z"
    },
    {
      "id": "big-gom-305",
      "identifier": "BIG-GOM-305",
      "title": "Triage migration",
      "priority": 2,
      "state": "Backlog",
      "labels": ["go-mainline", "triage"]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	issues := store.ListIssues([]string{"In Progress"})
	if len(issues) != 1 {
		t.Fatalf("expected one filtered issue, got %+v", issues)
	}
	if issues[0].Identifier != "BIG-GOM-301" || issues[0].Title != "Domain parity" || issues[0].Priority.(float64) != 1 {
		t.Fatalf("unexpected listed issue payload: %+v", issues[0])
	}
	if len(issues[0].Labels) != 2 || issues[0].Labels[0] != "go-mainline" || issues[0].Labels[1] != "domain" {
		t.Fatalf("unexpected labels: %+v", issues[0])
	}
}

func TestLocalIssueStoreAppendIssueCommentPreservesExistingComments(t *testing.T) {
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
	if err := store.AppendIssueComment("BIG-GOM-307", "added closeout evidence", time.Date(2026, 3, 18, 16, 45, 0, 0, time.UTC)); err != nil {
		t.Fatalf("append issue comment: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"body": "seed comment"`) {
		t.Fatalf("expected existing comment to remain, got %s", text)
	}
	if !strings.Contains(text, `"body": "added closeout evidence"`) {
		t.Fatalf("expected appended comment to be written, got %s", text)
	}
	if !strings.Contains(text, `"created_at": "2026-03-18T16:45:00Z"`) {
		t.Fatalf("expected appended comment timestamp, got %s", text)
	}
	if !strings.Contains(text, `"updated_at": "2026-03-18T16:45:00Z"`) {
		t.Fatalf("expected updated_at refresh, got %s", text)
	}
}

func TestLocalIssueStoreAppendIssueCommentDoesNotEscapeBodyText(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "description": "Keep Go && GitHub aligned -> no Python fallback",
      "priority": 1,
      "state": "In Progress",
      "labels": ["go", "tooling"],
      "assigned_to_worker": true,
      "comments": [],
      "created_at": "2026-03-18T09:00:00Z",
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
	if err := store.AppendIssueComment("BIG-GOM-307", "ran go test ./internal/refill && go test ./cmd/bigclawctl -> passed", time.Date(2026, 3, 18, 17, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("append issue comment: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if strings.Contains(text, `\u0026`) || strings.Contains(text, `\u003e`) {
		t.Fatalf("expected literal comment/body text, got %s", text)
	}
	if !strings.Contains(text, `"description": "Keep Go && GitHub aligned -> no Python fallback"`) {
		t.Fatalf("expected existing description literal text, got %s", text)
	}
	if !strings.Contains(text, `"body": "ran go test ./internal/refill && go test ./cmd/bigclawctl -> passed"`) {
		t.Fatalf("expected literal appended comment, got %s", text)
	}
}
