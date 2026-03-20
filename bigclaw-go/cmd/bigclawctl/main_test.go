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

func gitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}

func initGitRepo(t *testing.T, repo string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
}

func gitCommitFile(t *testing.T, repo string, name string, content string, message string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", name}, {"commit", "-m", message}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
		}
	}
	return gitOutput(t, repo, "rev-parse", "HEAD")
}

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
	for _, key := range []string{
		"SYMPHONY_BOOTSTRAP_REPO_URL",
		"SYMPHONY_BOOTSTRAP_GITHUB_URL",
		"SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH",
		"SYMPHONY_BOOTSTRAP_CACHE_ROOT",
		"SYMPHONY_BOOTSTRAP_CACHE_BASE",
		"SYMPHONY_BOOTSTRAP_CACHE_KEY",
	} {
		t.Setenv(key, "")
	}
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

func TestRunLocalIssueUpdateReadsCommentFromFile(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	commentPath := filepath.Join(tempDir, "comment.md")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "updated_at": "2026-03-18T09:00:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}
	commentBody := "Validation:\n- `go test ./...` -> ok\n- `github-sync status` -> ok\n"
	if err := os.WriteFile(commentPath, []byte(commentBody), 0o644); err != nil {
		t.Fatalf("write comment file: %v", err)
	}

	output, runErr := captureStdout(t, func() error {
		return runLocalIssue([]string{
			"update",
			"--local-issues", storePath,
			"--issue", "BIG-GOM-307",
			"--comment-file", commentPath,
			"--json",
		})
	})
	if runErr != nil {
		t.Fatalf("run local issue update with file: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"comment_added": true`)) {
		t.Fatalf("unexpected update output: %s", string(output))
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"body": "Validation:\n- `)) || !bytes.Contains(body, []byte("github-sync status")) {
		t.Fatalf("expected multiline comment in store, got %s", string(body))
	}
}

func TestRunLocalIssueUpdateReadsCommentFromStdin(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "updated_at": "2026-03-18T09:00:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	originalStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	os.Stdin = reader
	defer func() {
		os.Stdin = originalStdin
	}()
	stdinBody := "from stdin\nwith exact punctuation: `[]{}()`\n"
	if _, err := writer.WriteString(stdinBody); err != nil {
		t.Fatalf("write stdin body: %v", err)
	}
	_ = writer.Close()

	output, runErr := captureStdout(t, func() error {
		return runLocalIssue([]string{
			"update",
			"--local-issues", storePath,
			"--issue", "BIG-GOM-307",
			"--comment-file", "-",
			"--json",
		})
	})
	if runErr != nil {
		t.Fatalf("run local issue update with stdin: %v (stdout=%s)", runErr, string(output))
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"body": "from stdin\nwith exact punctuation:`)) || !bytes.Contains(body, []byte("[]{}()")) {
		t.Fatalf("expected stdin comment in store, got %s", string(body))
	}
}

func TestRunLocalIssueUpdateUsesSingleTimestampForStateAndComment(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "Todo",
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
			"--comment", "single timestamp update",
			"--json",
		})
	})
	if runErr != nil {
		t.Fatalf("run local issue update: %v (stdout=%s)", runErr, string(output))
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	updatedAtNeedle := []byte(`"updated_at": "`)
	updatedAtStart := bytes.Index(body, updatedAtNeedle)
	if updatedAtStart == -1 {
		t.Fatalf("expected updated_at in store, got %s", string(body))
	}
	updatedAtValueStart := updatedAtStart + len(updatedAtNeedle)
	updatedAtValueEnd := bytes.IndexByte(body[updatedAtValueStart:], '"')
	if updatedAtValueEnd == -1 {
		t.Fatalf("expected updated_at closing quote in store, got %s", string(body))
	}
	updatedAt := body[updatedAtValueStart : updatedAtValueStart+updatedAtValueEnd]
	createdAtLine := []byte(`"created_at": "` + string(updatedAt) + `"`)
	if !bytes.Contains(body, createdAtLine) {
		t.Fatalf("expected comment created_at to match updated_at %s, got %s", string(updatedAt), string(body))
	}
}

func TestRunLocalIssueListFiltersStatesAndPreservesTrackerOrder(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-302",
      "identifier": "BIG-GOM-302",
      "title": "Risk and policy",
      "state": "In Progress",
      "updated_at": "2026-03-18T09:00:00Z"
    },
    {
      "id": "big-gom-303",
      "identifier": "BIG-GOM-303",
      "title": "Workflow loop parity",
      "state": "Todo",
      "updated_at": "2026-03-18T09:00:00Z"
    },
    {
      "id": "big-gom-305",
      "identifier": "BIG-GOM-305",
      "title": "Operations view parity",
      "state": "Backlog",
      "updated_at": "2026-03-18T09:00:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	output, runErr := captureStdout(t, func() error {
		return runLocalIssue([]string{
			"list",
			"--local-issues", storePath,
			"--states", "In Progress,Todo",
			"--json",
		})
	})
	if runErr != nil {
		t.Fatalf("run local issue list: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"count": 2`)) {
		t.Fatalf("expected filtered issue count, got %s", string(output))
	}
	if bytes.Index(output, []byte(`"identifier": "BIG-GOM-302"`)) > bytes.Index(output, []byte(`"identifier": "BIG-GOM-303"`)) {
		t.Fatalf("expected tracker order to be preserved, got %s", string(output))
	}
	if bytes.Contains(output, []byte(`"identifier": "BIG-GOM-305"`)) {
		t.Fatalf("expected backlog issue to be filtered out, got %s", string(output))
	}
}

func TestRunLocalIssueListPrintsTabSeparatedSummaryByDefault(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
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

	output, runErr := captureStdout(t, func() error {
		return runLocalIssue([]string{
			"list",
			"--local-issues", storePath,
		})
	})
	if runErr != nil {
		t.Fatalf("run local issue list: %v (stdout=%s)", runErr, string(output))
	}
	if strings.TrimSpace(string(output)) != "BIG-GOM-307\tIn Progress\tToolchain migration" {
		t.Fatalf("unexpected summary output: %q", string(output))
	}
}

func TestResolveGitHubSyncRemotesPrefersUpstreamThenOriginAndGithub(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)
	for _, remote := range []string{"origin", "github"} {
		remotePath := filepath.Join(tmp, remote+".git")
		if output, err := exec.Command("git", "init", "--bare", remotePath).CombinedOutput(); err != nil {
			t.Fatalf("git init --bare %s failed: %v (%s)", remote, err, string(output))
		}
		cmd := exec.Command("git", "remote", "add", remote, remotePath)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git remote add %s failed: %v (%s)", remote, err, string(output))
		}
	}
	gitCommitFile(t, repo, "README.md", "hello\n", "initial commit")
	cmd := exec.Command("git", "push", "-u", "github", "HEAD")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git push -u github HEAD failed: %v (%s)", err, string(output))
	}

	remotes, err := resolveGitHubSyncRemotes(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(remotes, ",") != "github,origin" {
		t.Fatalf("unexpected remote order: %v", remotes)
	}
}

func TestRunGitHubSyncSyncPushesUpstreamAndOriginByDefault(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)
	for _, remote := range []string{"origin", "github"} {
		remotePath := filepath.Join(tmp, remote+".git")
		if output, err := exec.Command("git", "init", "--bare", remotePath).CombinedOutput(); err != nil {
			t.Fatalf("git init --bare %s failed: %v (%s)", remote, err, string(output))
		}
		cmd := exec.Command("git", "remote", "add", remote, remotePath)
		cmd.Dir = repo
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git remote add %s failed: %v (%s)", remote, err, string(output))
		}
	}
	gitCommitFile(t, repo, "README.md", "hello\n", "initial commit")
	cmd := exec.Command("git", "push", "-u", "github", "HEAD")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git push -u github HEAD failed: %v (%s)", err, string(output))
	}
	localSHA := gitCommitFile(t, repo, "README.md", "hello again\n", "second commit")

	output, runErr := captureStdout(t, func() error {
		return runGitHubSync([]string{"sync", "--repo", repo, "--json"})
	})
	if runErr != nil {
		t.Fatalf("run github-sync sync: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"remote": "github"`)) || !bytes.Contains(output, []byte(`"remote": "origin"`)) {
		t.Fatalf("expected both remotes in output, got %s", string(output))
	}
	for _, remote := range []string{"origin", "github"} {
		sha := gitOutput(t, repo, "ls-remote", "--heads", remote, gitOutput(t, repo, "branch", "--show-current"))
		if !strings.Contains(sha, localSHA) {
			t.Fatalf("expected %s to have %s, got %s", remote, localSHA, sha)
		}
	}
}
