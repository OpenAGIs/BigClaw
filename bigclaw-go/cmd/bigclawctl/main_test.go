package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/refill"
)

func TestNormalizeWorkspaceArgsLeavesBootstrapUntouched(t *testing.T) {
	args := []string{"--workspace", "/tmp/demo", "--issue", "BIG-GOM-307"}
	got := normalizeWorkspaceArgs("bootstrap", args)
	if strings.Join(got, "\x00") != strings.Join(args, "\x00") {
		t.Fatalf("expected bootstrap args unchanged, got %#v", got)
	}
}

func TestNormalizeWorkspaceArgsSupportsLegacyValidateFlags(t *testing.T) {
	got := normalizeWorkspaceArgs("validate", []string{
		"--repo-url", "git@github.com:OpenAGIs/BigClaw.git",
		"--workspace-root", "/tmp/workspaces",
		"--issues", "BIG-GOM-302", "BIG-GOM-307",
		"--report-file", "reports/bootstrap.json",
		"--no-cleanup",
	})
	want := []string{
		"--repo-url", "git@github.com:OpenAGIs/BigClaw.git",
		"--workspace-root", "/tmp/workspaces",
		"--issues=BIG-GOM-302,BIG-GOM-307",
		"--report", "reports/bootstrap.json",
		"--cleanup=false",
	}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected normalized args: got %#v want %#v", got, want)
	}
}

func TestLoadWorkspaceDefaultsUsesEnvironment(t *testing.T) {
	t.Setenv("SYMPHONY_BOOTSTRAP_REPO_URL", "ssh://mirror/repo.git")
	t.Setenv("SYMPHONY_BOOTSTRAP_GITHUB_URL", "git@github.com:OpenAGIs/BigClaw.git")
	t.Setenv("SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH", "release")
	t.Setenv("SYMPHONY_BOOTSTRAP_CACHE_ROOT", "/tmp/cache-root")
	t.Setenv("SYMPHONY_BOOTSTRAP_CACHE_BASE", "/tmp/cache-base")
	t.Setenv("SYMPHONY_BOOTSTRAP_CACHE_KEY", "bigclaw-cache")

	defaults := loadWorkspaceDefaults()
	if defaults.repoURL != "ssh://mirror/repo.git" || defaults.githubURL != "git@github.com:OpenAGIs/BigClaw.git" || defaults.defaultBranch != "release" || defaults.cacheRoot != "/tmp/cache-root" || defaults.cacheBase != "/tmp/cache-base" || defaults.cacheKey != "bigclaw-cache" {
		t.Fatalf("unexpected workspace defaults: %+v", defaults)
	}
}

func TestLoadWorkspaceDefaultsFallsBackWhenEnvMissing(t *testing.T) {
	defaults := loadWorkspaceDefaults()
	if defaults.repoURL != "" || defaults.githubURL != "" || defaults.cacheRoot != "" || defaults.cacheKey != "" {
		t.Fatalf("expected empty optional defaults, got %+v", defaults)
	}
	if defaults.defaultBranch != "main" || defaults.cacheBase != "~/.cache/symphony/repos" {
		t.Fatalf("unexpected fallback defaults: %+v", defaults)
	}
}

func captureStdout(t *testing.T, fn func() error) ([]byte, error) {
	t.Helper()
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()
	runErr := fn()
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	return output, runErr
}

func TestLinearClientFetchIssueStatesPreservesLinearIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "query RefillIssues") {
			t.Fatalf("unexpected query: %s", request.Query)
		}
		response := refillResponse{}
		response.Data.Issues.Nodes = []linearIssueNode{
			{
				ID:         "issue-linear-1",
				Identifier: "BIG-GOM-301",
				Title:      "Domain parity",
				State: struct {
					Name string `json:"name"`
				}{Name: "Todo"},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &linearClient{apiKey: "test-token", endpoint: server.URL, httpClient: server.Client()}
	issues, err := client.fetchIssueStates("project-slug", []string{"Todo"})
	if err != nil {
		t.Fatalf("fetch issue states: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].ID != "issue-linear-1" || issues[0].Identifier != "BIG-GOM-301" {
		t.Fatalf("unexpected issue payload: %+v", issues[0])
	}
}

func TestRunRefillOncePromotesUsingLinearIssueID(t *testing.T) {
	queuePath := filepath.Join(t.TempDir(), "queue.json")
	if err := os.WriteFile(queuePath, []byte(`{
  "project": {"slug_id": "project-slug"},
  "policy": {
    "target_in_progress": 1,
    "activate_state_name": "In Progress",
    "activate_state_id": "state-in-progress",
    "refill_states": ["Todo", "Backlog"]
  },
  "issue_order": ["BIG-GOM-301"],
  "issues": [
    {"identifier": "BIG-GOM-301", "title": "Domain parity", "track": "Control/Workflow", "status": "Todo"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}
	queue, err := refill.LoadQueue(queuePath)
	if err != nil {
		t.Fatalf("load queue: %v", err)
	}

	var promotedIssueID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		switch {
		case strings.Contains(request.Query, "query RefillIssues"):
			response := refillResponse{}
			response.Data.Issues.Nodes = []linearIssueNode{
				{
					ID:         "issue-linear-301",
					Identifier: "BIG-GOM-301",
					Title:      "Domain parity",
					State: struct {
						Name string `json:"name"`
					}{Name: "Todo"},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		case strings.Contains(request.Query, "mutation PromoteIssue"):
			promotedIssueID = request.Variables["id"].(string)
			response := promoteResponse{}
			response.Data.IssueUpdate.Success = true
			response.Data.IssueUpdate.Issue.Identifier = "BIG-GOM-301"
			response.Data.IssueUpdate.Issue.State.Name = "In Progress"
			_ = json.NewEncoder(w).Encode(response)
		default:
			t.Fatalf("unexpected query: %s", request.Query)
		}
	}))
	defer server.Close()

	client := &linearClient{apiKey: "test-token", endpoint: server.URL, httpClient: server.Client()}

	output, runErr := captureStdout(t, func() error {
		return runRefillOnce(queue, client, true, "", nil)
	})
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if promotedIssueID != "issue-linear-301" {
		t.Fatalf("expected promotion to use Linear issue ID, got %q", promotedIssueID)
	}
	if !bytes.Contains(output, []byte(`"BIG-GOM-301"`)) {
		t.Fatalf("expected refill output to include candidate payload, got %s", string(output))
	}
}

func TestRunRefillOncePromotesUsingLocalIssueStore(t *testing.T) {
	tempDir := t.TempDir()
	queuePath := filepath.Join(tempDir, "queue.json")
	if err := os.WriteFile(queuePath, []byte(`{
  "project": {"slug_id": "project-slug"},
  "policy": {
    "target_in_progress": 1,
    "activate_state_name": "In Progress",
    "activate_state_id": "state-in-progress",
    "refill_states": ["Todo", "Backlog"]
  },
  "issue_order": ["BIG-GOM-303"],
  "issues": [
    {"identifier": "BIG-GOM-303", "title": "Workflow loop parity", "track": "Control/Workflow", "status": "Todo"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}
	queue, err := refill.LoadQueue(queuePath)
	if err != nil {
		t.Fatalf("load queue: %v", err)
	}

	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-303",
      "identifier": "BIG-GOM-303",
      "title": "Workflow loop parity",
      "state": "Todo",
      "labels": ["go", "workflow"],
      "assigned_to_worker": true,
      "created_at": "2026-03-18T09:00:00Z",
      "updated_at": "2026-03-18T09:00:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}
	store, err := refill.LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	client := &localIssueClient{
		store: store,
		now: func() time.Time {
			return time.Date(2026, 3, 18, 12, 34, 56, 0, time.UTC)
		},
	}

	output, runErr := captureStdout(t, func() error {
		return runRefillOnce(queue, client, true, "", nil)
	})
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"backend": "local"`)) {
		t.Fatalf("expected refill output to advertise local backend, got %s", string(output))
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read updated local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"state": "In Progress"`)) {
		t.Fatalf("expected local issue store state update, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(`"updated_at": "2026-03-18T12:34:56Z"`)) {
		t.Fatalf("expected local issue store updated_at refresh, got %s", string(body))
	}
}

func TestRunLocalIssueUpdateAppendsCommentAndState(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "Todo",
      "comments": [
        {"body": "seed comment", "created_at": "2026-03-18T09:00:00Z"}
      ],
      "updated_at": "2026-03-18T09:00:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	output, runErr := captureStdout(t, func() error {
		return runLocalIssue([]string{
			"update",
			"--local-issues", storePath,
			"--issue", "BIG-GOM-307",
			"--state", "In Progress",
			"--comment", "validated the go-first tracker path",
			"--json",
		})
	})
	if runErr != nil {
		t.Fatalf("run local issue update: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"comment_added": true`)) || !bytes.Contains(output, []byte(`"state": "In Progress"`)) {
		t.Fatalf("unexpected update output: %s", string(output))
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"body": "validated the go-first tracker path"`)) {
		t.Fatalf("expected comment in store, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(`"state": "In Progress"`)) {
		t.Fatalf("expected updated state in store, got %s", string(body))
	}
}
