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

func TestLinearClientFetchIssueStatesPreservesTrackerIDs(t *testing.T) {
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

func TestResolvePathAgainstRepoRootJoinsRelativePaths(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "repo-root")
	if got := resolvePathAgainstRepoRoot(repoRoot, "reports/bootstrap-cache-validation.json"); got != filepath.Join(repoRoot, "reports/bootstrap-cache-validation.json") {
		t.Fatalf("expected join, got %q", got)
	}
	if got := resolvePathAgainstRepoRoot(repoRoot, "/tmp/absolute.json"); got != "/tmp/absolute.json" {
		t.Fatalf("expected absolute path passthrough, got %q", got)
	}
	if got := resolvePathAgainstRepoRoot(repoRoot, "~/relative.json"); got != "~/relative.json" {
		t.Fatalf("expected tilde passthrough, got %q", got)
	}
	if got := resolvePathAgainstRepoRoot("", "reports/bootstrap-cache-validation.json"); got != "reports/bootstrap-cache-validation.json" {
		t.Fatalf("expected repoRoot empty passthrough, got %q", got)
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

	runErr := runRefillOnce(queue, client, true, "", nil, false, queuePath, "")
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

	runErr := runRefillOnce(queue, client, true, "", nil, false, queuePath, storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"backend": "local"`)) {
		t.Fatalf("expected refill output to advertise local backend, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_path":`)) || !bytes.Contains(output, []byte(queuePath)) {
		t.Fatalf("expected refill output to include queue_path, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"local_issues_path":`)) || !bytes.Contains(output, []byte(storePath)) {
		t.Fatalf("expected refill output to include local_issues_path, got %s", string(output))
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

func TestRunRefillOnceLocalIssueStoreDetectsQueueDrainedWhenMetadataStale(t *testing.T) {
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
      "state": "Done",
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
	client := &localIssueClient{store: store}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_drained": true`)) {
		t.Fatalf("expected drained queue warning, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_runnable": 0`)) {
		t.Fatalf("expected runnable count 0, got %s", string(output))
	}
}

func TestRunRefillOnceLocalBackendUsesAllLocalStatesForRunnableCount(t *testing.T) {
	tempDir := t.TempDir()
	queuePath := filepath.Join(tempDir, "queue.json")
	if err := os.WriteFile(queuePath, []byte(`{
  "project": {"slug_id": "project-slug"},
  "policy": {
    "target_in_progress": 2,
    "activate_state_name": "In Progress",
    "activate_state_id": "state-in-progress",
    "refill_states": ["Todo", "Backlog"]
  },
  "issue_order": ["BIG-PAR-229", "BIG-PAR-230", "BIG-PAR-231"],
  "issues": [
    {"identifier": "BIG-PAR-229", "title": "sync queue", "track": "Automation", "status": "In Progress"},
    {"identifier": "BIG-PAR-230", "title": "live runnable count", "track": "Automation", "status": "In Progress"},
    {"identifier": "BIG-PAR-231", "title": "docs", "track": "Automation", "status": "Todo"}
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
    {"id": "big-par-229", "identifier": "BIG-PAR-229", "state": "Done"},
    {"id": "big-par-230", "identifier": "BIG-PAR-230", "state": "Done"},
    {"id": "big-par-231", "identifier": "BIG-PAR-231", "state": "Todo"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}
	store, err := refill.LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	client := &localIssueClient{store: store}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_runnable": 1`)) {
		t.Fatalf("expected runnable count to use full local state, got %s", string(output))
	}
}

func TestRunRefillOnceLocalBackendSyncsQueueStatusFromLocalIssues(t *testing.T) {
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
  "issue_order": ["BIG-GOM-501"],
  "issues": [
    {"identifier": "BIG-GOM-501", "title": "Queue metadata drift", "track": "Automation", "status": "Todo"}
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
    {"id": "big-gom-501", "identifier": "BIG-GOM-501", "state": "Done"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}
	store, err := refill.LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	client := &localIssueClient{store: store}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runRefillOnce(queue, client, true, "", nil, true, queuePath, storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}

	body, err := os.ReadFile(queuePath)
	if err != nil {
		t.Fatalf("read updated queue file: %v", err)
	}
	if !bytes.Contains(body, []byte(`"status": "Done"`)) {
		t.Fatalf("expected queue status update, got %s", string(body))
	}
	if !bytes.Contains(output, []byte(`"queue_status_updates": 1`)) {
		t.Fatalf("expected refill output to include queue status updates, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_written": true`)) {
		t.Fatalf("expected refill output to confirm write, got %s", string(output))
	}
}

func TestRunLocalIssuesSetStateUpdatesStore(t *testing.T) {
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

	if err := runLocalIssues([]string{
		"set-state",
		"--local-issues", storePath,
		"--issue", "BIG-GOM-307",
		"--state", "In Progress",
		"--created-at", "2026-03-20T11:00:00Z",
		"--json",
	}); err != nil {
		t.Fatalf("run local-issues set-state: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"state": "In Progress"`)) {
		t.Fatalf("expected updated state, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(`"updated_at": "2026-03-20T11:00:00Z"`)) {
		t.Fatalf("expected updated timestamp, got %s", string(body))
	}
}

func TestRunLocalIssuesCommentAppendsComment(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "comments": [
        {"author": "codex", "created_at": "2026-03-18T09:00:00Z", "body": "seed"}
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	if err := runLocalIssues([]string{
		"comment",
		"--local-issues", storePath,
		"--issue", "BIG-GOM-307",
		"--author", "codex",
		"--body", "validation passed",
		"--created-at", "2026-03-20T11:05:00Z",
		"--json",
	}); err != nil {
		t.Fatalf("run local-issues comment: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"body": "validation passed"`)) {
		t.Fatalf("expected appended comment, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(`"updated_at": "2026-03-20T11:05:00Z"`)) {
		t.Fatalf("expected updated timestamp, got %s", string(body))
	}
}

func TestRunLocalIssuesEnsureCreatesMissingIssue(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")

	if err := runLocalIssues([]string{
		"ensure",
		"--local-issues", storePath,
		"--identifier", "BIG-PAR-225",
		"--title", "Refill queue follow-up",
		"--description", "seed missing issue",
		"--state", "In Progress",
		"--labels", "parallel,tracker",
		"--created-at", "2026-03-22T10:40:00Z",
		"--json",
	}); err != nil {
		t.Fatalf("run local-issues ensure create: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"identifier": "BIG-PAR-225"`)) {
		t.Fatalf("expected BIG-PAR-225 issue, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(`"state": "In Progress"`)) {
		t.Fatalf("expected in-progress state, got %s", string(body))
	}
}

func TestRunLocalIssuesEnsureUpdatesExistingStateCaseInsensitive(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-225",
      "identifier": "BIG-PAR-225",
      "title": "Refill queue follow-up",
      "state": "Todo",
      "updated_at": "2026-03-22T10:40:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	if err := runLocalIssues([]string{
		"ensure",
		"--local-issues", storePath,
		"--identifier", "big-par-225",
		"--state", "In Progress",
		"--set-state-if-exists",
		"--created-at", "2026-03-22T10:45:00Z",
		"--json",
	}); err != nil {
		t.Fatalf("run local-issues ensure update: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"state": "In Progress"`)) {
		t.Fatalf("expected updated state, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(`"updated_at": "2026-03-22T10:45:00Z"`)) {
		t.Fatalf("expected updated timestamp, got %s", string(body))
	}
}
