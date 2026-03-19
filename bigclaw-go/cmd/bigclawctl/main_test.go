package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/refill"
)

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

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runRefillOnce(queue, client, true, "", nil)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
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

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runRefillOnce(queue, client, true, "", nil)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
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

func TestRunRefillOnceTriggersRefreshAfterPromotion(t *testing.T) {
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

	var refreshCalls int
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalls++
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST refresh, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer refreshServer.Close()

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

	if err := runRefillOnce(queue, client, true, refreshServer.URL, nil); err != nil {
		t.Fatalf("run refill once: %v", err)
	}
	if refreshCalls != 1 {
		t.Fatalf("expected exactly one refresh call, got %d", refreshCalls)
	}
}

func TestRunRefillOnceFailsWhenRefreshEndpointReturnsError(t *testing.T) {
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

	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "dashboard unavailable", http.StatusBadGateway)
	}))
	defer refreshServer.Close()

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

	err = runRefillOnce(queue, client, true, refreshServer.URL, nil)
	if err == nil {
		t.Fatal("expected refresh endpoint failure to abort refill")
	}
	if !strings.Contains(err.Error(), "refresh HTTP 502") {
		t.Fatalf("expected refresh HTTP status in error, got %v", err)
	}
}

func TestRunLocalIssueCloseoutMarksDoneAndAppendsComment(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Workflow, bootstrap, and GitHub sync toolchain migration",
      "state": "In Progress",
      "comments": []
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runLocalIssue([]string{
		"closeout",
		"--local-issues", storePath,
		"--issue", "BIG-GOM-307",
		"--summary", "Moved local tracker closeout into Go.",
		"--validation", "go test ./cmd/bigclawctl ./internal/refill",
		"--commit", "deadbeef",
		"--pr-url", "https://github.com/OpenAGIs/BigClaw/pull/307",
		"--json",
	})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run local issue closeout: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"state": "Done"`)) {
		t.Fatalf("expected done state in output, got %s", string(output))
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"state": "Done"`) {
		t.Fatalf("expected done state, got %s", text)
	}
	if !strings.Contains(text, `What changed:\nMoved local tracker closeout into Go.`) {
		t.Fatalf("expected closeout summary in comment, got %s", text)
	}
	if !strings.Contains(text, `"commit_sha": "deadbeef"`) {
		t.Fatalf("expected commit sha metadata, got %s", text)
	}
}

func TestRunLocalIssueUpdateTransitionsStateAndAppendsComment(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-303",
      "identifier": "BIG-GOM-303",
      "title": "Workflow orchestration and scheduler loop migration",
      "state": "Todo",
      "comments": []
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runLocalIssue([]string{
		"update",
		"--local-issues", storePath,
		"--issue", "BIG-GOM-303",
		"--state", "In Progress",
		"--comment", "Claimed the workflow-loop migration slice for Go implementation.",
		"--commit", "feedface",
		"--pr-url", "https://github.com/OpenAGIs/BigClaw/pull/303",
		"--json",
	})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run local issue update: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"state": "In Progress"`)) {
		t.Fatalf("expected in-progress state in output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"commented": true`)) {
		t.Fatalf("expected commented=true in output, got %s", string(output))
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"state": "In Progress"`) {
		t.Fatalf("expected in-progress state, got %s", text)
	}
	if !strings.Contains(text, `Claimed the workflow-loop migration slice for Go implementation.`) {
		t.Fatalf("expected progress comment, got %s", text)
	}
	if !strings.Contains(text, `"commit_sha": "feedface"`) {
		t.Fatalf("expected commit metadata, got %s", text)
	}
}

func TestRunWorkspaceValidateEmitsJSONReport(t *testing.T) {
	root := t.TempDir()
	remote := initWorkspaceValidationRemote(t, root)
	workspaceRoot := filepath.Join(root, "workspaces")

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runWorkspace([]string{
		"validate",
		"--workspace-root", workspaceRoot,
		"--repo-url", remote,
		"--cache-base", filepath.Join(root, "repos"),
		"--issues", "OPE-401,OPE-402,OPE-403",
		"--json",
	})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run workspace validate: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"workspace_count": 3`)) {
		t.Fatalf("expected workspace count in JSON report, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"cleanup_preserved_cache": true`)) {
		t.Fatalf("expected cleanup summary in JSON report, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"bootstrap_results"`)) {
		t.Fatalf("expected bootstrap results in JSON report, got %s", string(output))
	}
}

func TestRunGitHubSyncStatusRequireCleanReturnsExitErrorForDirtyRepo(t *testing.T) {
	root := t.TempDir()
	repo := initGitHubSyncRepo(t, root)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runGitHubSync([]string{
		"status",
		"--repo", repo,
		"--require-clean",
		"--json",
	})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr == nil {
		t.Fatalf("expected require-clean to fail for dirty repo (stdout=%s)", string(output))
	}
	if _, ok := runErr.(exitError); !ok {
		t.Fatalf("expected exitError, got %T (%v)", runErr, runErr)
	}
	if !bytes.Contains(output, []byte(`"dirty": true`)) {
		t.Fatalf("expected dirty status in output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"synced": true`)) {
		t.Fatalf("expected synced branch state in output, got %s", string(output))
	}
}

func TestRunGitHubSyncSyncAllowDirtyReportsStatus(t *testing.T) {
	root := t.TempDir()
	repo := initGitHubSyncRepo(t, root)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runGitHubSync([]string{
		"sync",
		"--repo", repo,
		"--allow-dirty",
		"--json",
	})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("expected allow-dirty sync to succeed: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"dirty": true`)) {
		t.Fatalf("expected dirty status in output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"status": "ok"`)) {
		t.Fatalf("expected ok status in output, got %s", string(output))
	}
}

func TestRunGitHubSyncStatusRequireSyncedReturnsExitErrorForUnpushedCommit(t *testing.T) {
	root := t.TempDir()
	repo := initGitHubSyncRepo(t, root)
	cmd := exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config user.email failed: %v (%s)", err, string(output))
	}
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config user.name failed: %v (%s)", err, string(output))
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\nnext\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "README.md"},
		{"commit", "-m", "next"},
	} {
		cmd = exec.Command("git", args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
	localSHA, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	remoteSHA, err := exec.Command("git", "-C", repo, "rev-parse", "origin/main").Output()
	if err != nil {
		t.Fatalf("git rev-parse origin/main failed: %v", err)
	}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runGitHubSync([]string{
		"status",
		"--repo", repo,
		"--require-synced",
		"--json",
	})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr == nil {
		t.Fatalf("expected require-synced to fail for unpushed commit (stdout=%s)", string(output))
	}
	if _, ok := runErr.(exitError); !ok {
		t.Fatalf("expected exitError, got %T (%v)", runErr, runErr)
	}
	if !bytes.Contains(output, []byte(`"synced": false`)) {
		t.Fatalf("expected unsynced status in output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"remote_exists": true`)) {
		t.Fatalf("expected remote branch to exist in output, got %s", string(output))
	}
	if !bytes.Contains(output, bytes.TrimSpace(localSHA)) {
		t.Fatalf("expected local SHA in output, got %s", string(output))
	}
	if !bytes.Contains(output, bytes.TrimSpace(remoteSHA)) {
		t.Fatalf("expected remote SHA in output, got %s", string(output))
	}
}

func TestRunGitHubSyncSyncRequireSyncedPublishesMissingIssueBranch(t *testing.T) {
	root := t.TempDir()
	repo := initGitHubSyncRepo(t, root)
	remotePath, err := exec.Command("git", "-C", repo, "remote", "get-url", "origin").Output()
	if err != nil {
		t.Fatalf("git remote get-url failed: %v", err)
	}

	workspace := filepath.Join(root, "workspace")
	cmd := exec.Command("git", "clone", strings.TrimSpace(string(remotePath)), workspace)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (%s)", err, string(output))
	}
	cmd = exec.Command("git", "checkout", "-b", "symphony/BIG-GOM-307")
	cmd.Dir = workspace
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b failed: %v (%s)", err, string(output))
	}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runGitHubSync([]string{
		"sync",
		"--repo", workspace,
		"--require-synced",
		"--json",
	})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("expected sync require-synced to publish missing branch: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"remote_exists": true`)) {
		t.Fatalf("expected published remote branch in output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"synced": true`)) {
		t.Fatalf("expected synced status in output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"pushed": true`)) {
		t.Fatalf("expected pushed status in output, got %s", string(output))
	}

	remoteBranchSHA, err := exec.Command("git", "-C", workspace, "rev-parse", "origin/symphony/BIG-GOM-307").Output()
	if err != nil {
		t.Fatalf("git rev-parse origin/symphony/BIG-GOM-307 failed: %v", err)
	}
	localSHA, err := exec.Command("git", "-C", workspace, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(localSHA), bytes.TrimSpace(remoteBranchSHA)) {
		t.Fatalf("expected published branch SHA to match HEAD, local=%s remote=%s", bytes.TrimSpace(localSHA), bytes.TrimSpace(remoteBranchSHA))
	}
}

func TestRunWorkspaceValidateWritesMarkdownReport(t *testing.T) {
	root := t.TempDir()
	remote := initWorkspaceValidationRemote(t, root)
	reportPath := filepath.Join(root, "reports", "bootstrap-validation.md")

	runErr := runWorkspace([]string{
		"validate",
		"--workspace-root", filepath.Join(root, "workspaces"),
		"--repo-url", remote,
		"--cache-base", filepath.Join(root, "repos"),
		"--issues", "OPE-411,OPE-412,OPE-413",
		"--report", reportPath,
		"--cleanup=true",
	})
	if runErr != nil {
		t.Fatalf("run workspace validate with report: %v", runErr)
	}

	body, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "# Symphony bootstrap cache validation") {
		t.Fatalf("expected markdown heading, got %s", text)
	}
	if !strings.Contains(text, "- Workspaces: `3`") {
		t.Fatalf("expected workspace summary, got %s", text)
	}
	if !strings.Contains(text, "Cleanup preserved cache: `true`") {
		t.Fatalf("expected cleanup summary, got %s", text)
	}
}

func initWorkspaceValidationRemote(t *testing.T, root string) string {
	t.Helper()
	remote := filepath.Join(root, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", "--initial-branch=main", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
		{"remote", "add", "origin", remote},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = source
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "README.md"},
		{"commit", "-m", "initial"},
		{"push", "-u", "origin", "main"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = source
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
	return remote
}

func initGitHubSyncRepo(t *testing.T, root string) string {
	t.Helper()
	remote := filepath.Join(root, "remote.git")
	if output, err := exec.Command("git", "init", "--bare", "--initial-branch=main", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v (%s)", err, string(output))
	}

	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
		{"remote", "add", "origin", remote},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "README.md"},
		{"commit", "-m", "initial"},
		{"push", "-u", "origin", "main"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
	return repo
}
