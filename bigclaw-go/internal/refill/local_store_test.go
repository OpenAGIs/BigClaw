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

func TestLocalIssueStoreCloseIssueAppendsCommentMetadata(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "comments": [
        {"body": "existing", "created_at": "2026-03-18T09:00:00Z"}
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

	updatedState, err := store.CloseIssue("BIG-GOM-307", "Done", LocalIssueComment{
		Body:      "What changed:\nMoved workflow sync into Go\n\nValidation:\ngo test ./...",
		CreatedAt: time.Date(2026, 3, 18, 16, 45, 0, 0, time.UTC),
		Metadata: map[string]any{
			"commit_sha": "abc123",
			"pr_url":     "https://github.com/OpenAGIs/BigClaw/pull/307",
		},
	}, time.Date(2026, 3, 18, 16, 45, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("close issue: %v", err)
	}
	if updatedState != "Done" {
		t.Fatalf("expected Done, got %q", updatedState)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"state": "Done"`) {
		t.Fatalf("expected done state, got %s", text)
	}
	if strings.Count(text, `"body":`) != 2 {
		t.Fatalf("expected appended comment, got %s", text)
	}
	if !strings.Contains(text, `"commit_sha": "abc123"`) {
		t.Fatalf("expected commit metadata, got %s", text)
	}
	if !strings.Contains(text, `"pr_url": "https://github.com/OpenAGIs/BigClaw/pull/307"`) {
		t.Fatalf("expected PR metadata, got %s", text)
	}
}

func TestLocalIssueStoreUpdateIssueAppendsCommentWithoutStateChange(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "comments": []
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	updatedState, err := store.UpdateIssue("BIG-GOM-307", "", LocalIssueComment{
		Body:      "Validated branch sync after the latest automation patch.",
		CreatedAt: time.Date(2026, 3, 18, 17, 0, 0, 0, time.UTC),
		Metadata: map[string]any{
			"commit_sha": "1234abcd",
		},
	}, time.Date(2026, 3, 18, 17, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("update issue: %v", err)
	}
	if updatedState != "In Progress" {
		t.Fatalf("expected state to remain In Progress, got %q", updatedState)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"state": "In Progress"`) {
		t.Fatalf("expected state to be preserved, got %s", text)
	}
	if !strings.Contains(text, `Validated branch sync after the latest automation patch.`) {
		t.Fatalf("expected appended comment, got %s", text)
	}
	if !strings.Contains(text, `"updated_at": "2026-03-18T17:00:00Z"`) {
		t.Fatalf("expected updated_at refresh, got %s", text)
	}
}
