package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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
	markdownPath := filepath.Join(t.TempDir(), "queue.md")
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

	runErr := runRefillOnce(queue, client, true, "", nil, false, queuePath, markdownPath, "")
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

func TestRunRefillOnceLinearBackendUsesConfiguredActivateStateName(t *testing.T) {
	queuePath := filepath.Join(t.TempDir(), "queue.json")
	markdownPath := filepath.Join(t.TempDir(), "queue.md")
	if err := os.WriteFile(queuePath, []byte(`{
  "project": {"slug_id": "project-slug"},
  "policy": {
    "target_in_progress": 1,
    "activate_state_name": "Queued for Review",
    "activate_state_id": "state-review",
    "refill_states": ["Todo", "Backlog"]
  },
  "issue_order": ["BIG-PAR-389", "BIG-PAR-390"],
  "issues": [
    {"identifier": "BIG-PAR-389", "title": "Honor configured activate state in refill fetches", "track": "Automation", "status": "Queued for Review"},
    {"identifier": "BIG-PAR-390", "title": "Normalize local store state updates", "track": "Automation", "status": "Todo"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}
	queue, err := refill.LoadQueue(queuePath)
	if err != nil {
		t.Fatalf("load queue: %v", err)
	}

	var requestedStateNames []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "query RefillIssues") {
			t.Fatalf("unexpected query: %s", request.Query)
		}
		rawStateNames, ok := request.Variables["stateNames"].([]any)
		if !ok {
			t.Fatalf("expected stateNames variable, got %#v", request.Variables["stateNames"])
		}
		for _, stateName := range rawStateNames {
			requestedStateNames = append(requestedStateNames, stateName.(string))
		}
		response := refillResponse{}
		response.Data.Issues.Nodes = []linearIssueNode{
			{
				ID:         "issue-linear-389",
				Identifier: "BIG-PAR-389",
				Title:      "Honor configured activate state in refill fetches",
				State: struct {
					Name string `json:"name"`
				}{Name: "queued for review."},
			},
			{
				ID:         "issue-linear-390",
				Identifier: "BIG-PAR-390",
				Title:      "Normalize local store state updates",
				State: struct {
					Name string `json:"name"`
				}{Name: "Todo"},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
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

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, markdownPath, "")
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !reflect.DeepEqual(requestedStateNames, []string{"Queued for Review", "Todo", "Backlog"}) {
		t.Fatalf("expected configured activate state fetch list, got %#v", requestedStateNames)
	}
	if !bytes.Contains(output, []byte(`"active_in_progress": [`)) || !bytes.Contains(output, []byte(`"BIG-PAR-389"`)) {
		t.Fatalf("expected custom active state issue in payload, got %s", string(output))
	}
}

func TestRunRefillOnceLinearBackendDeduplicatesEquivalentFetchStates(t *testing.T) {
	queuePath := filepath.Join(t.TempDir(), "queue.json")
	markdownPath := filepath.Join(t.TempDir(), "queue.md")
	if err := os.WriteFile(queuePath, []byte(`{
  "project": {"slug_id": "project-slug"},
  "policy": {
    "target_in_progress": 1,
    "activate_state_name": "Queued for Review",
    "activate_state_id": "state-review",
    "refill_states": [" todo. ", "Backlog", "Todo", "backlog."]
  },
  "issue_order": ["BIG-PAR-393"],
  "issues": [
    {"identifier": "BIG-PAR-393", "title": "Stabilize normalized refill fetch state lists", "track": "Automation", "status": "Todo"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}
	queue, err := refill.LoadQueue(queuePath)
	if err != nil {
		t.Fatalf("load queue: %v", err)
	}

	var requestedStateNames []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "query RefillIssues") {
			t.Fatalf("unexpected query: %s", request.Query)
		}
		rawStateNames, ok := request.Variables["stateNames"].([]any)
		if !ok {
			t.Fatalf("expected stateNames variable, got %#v", request.Variables["stateNames"])
		}
		for _, stateName := range rawStateNames {
			requestedStateNames = append(requestedStateNames, stateName.(string))
		}
		response := refillResponse{}
		response.Data.Issues.Nodes = []linearIssueNode{}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &linearClient{apiKey: "test-token", endpoint: server.URL, httpClient: server.Client()}

	if err := runRefillOnce(queue, client, false, "", nil, false, queuePath, markdownPath, ""); err != nil {
		t.Fatalf("run refill once: %v", err)
	}
	if !reflect.DeepEqual(requestedStateNames, []string{"Queued for Review", "Todo", "Backlog"}) {
		t.Fatalf("expected deduplicated fetch states, got %#v", requestedStateNames)
	}
}

func TestRunRefillOncePromotesUsingLocalIssueStore(t *testing.T) {
	tempDir := t.TempDir()
	queuePath := filepath.Join(tempDir, "queue.json")
	markdownPath := filepath.Join(tempDir, "queue.md")
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

	runErr := runRefillOnce(queue, client, true, "", nil, false, queuePath, markdownPath, storePath)
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
	if !bytes.Contains(output, []byte(`"markdown_written": true`)) {
		t.Fatalf("expected refill output to preview markdown write after promotion, got %s", string(output))
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
	markdownBody, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown file: %v", err)
	}
	if !bytes.Contains(markdownBody, []byte("## Canonical refill order")) || !bytes.Contains(markdownBody, []byte("`BIG-GOM-303`")) {
		t.Fatalf("expected markdown sync output, got %s", string(markdownBody))
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

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
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
	if !bytes.Contains(output, []byte(`"next_steps": [`)) {
		t.Fatalf("expected drained queue next_steps guidance, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`docs/parallel-refill-queue.json`)) {
		t.Fatalf("expected queue recovery hint, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`refill seed`)) {
		t.Fatalf("expected refill seed hint, got %s", string(output))
	}
}

func TestRunRefillOnceReportsAbsolutePathsForRelativeInputs(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	queuePath := "queue.json"
	markdownPath := "queue.md"
	storePath := "local-issues.json"
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
    {"identifier": "BIG-GOM-303", "title": "Workflow loop parity", "track": "Control/Workflow", "status": "Done"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}
	queue, err := refill.LoadQueue(queuePath)
	if err != nil {
		t.Fatalf("load queue: %v", err)
	}

	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-gom-303", "identifier": "BIG-GOM-303", "state": "Done"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}
	store, err := refill.LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	client := &localIssueClient{store: store}

	absoluteQueuePath := filepath.Join(tempDir, queuePath)
	absoluteMarkdownPath := filepath.Join(tempDir, markdownPath)
	absoluteStorePath := filepath.Join(tempDir, storePath)

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, markdownPath, storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_path":`)) || !bytes.Contains(output, []byte(absoluteQueuePath)) {
		t.Fatalf("expected absolute queue_path, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"markdown_path":`)) || !bytes.Contains(output, []byte(absoluteMarkdownPath)) {
		t.Fatalf("expected absolute markdown_path, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"local_issues_path":`)) || !bytes.Contains(output, []byte(absoluteStorePath)) {
		t.Fatalf("expected absolute local_issues_path, got %s", string(output))
	}
}

func TestRunHelpAtRootPrintsUsageAndExitsZero(t *testing.T) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	code := run([]string{"--help"})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stdout=%s)", code, string(output))
	}
	if !strings.Contains(string(output), "usage: bigclawctl") {
		t.Fatalf("expected usage in help output, got %s", string(output))
	}
	if !strings.Contains(string(output), "github-sync") || !strings.Contains(string(output), "refill") {
		t.Fatalf("expected command list in help output, got %s", string(output))
	}
}

func TestRunRefillHelpPrintsDefaultsAndExitsZero(t *testing.T) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	code := run([]string{"refill", "--help"})
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stdout=%s)", code, string(output))
	}
	text := string(output)
	if !strings.Contains(text, "usage: bigclawctl refill") {
		t.Fatalf("expected refill usage, got %s", text)
	}
	if !strings.Contains(text, "-sync-queue-status") {
		t.Fatalf("expected sync-queue-status flag in help output, got %s", text)
	}
	if !strings.Contains(text, "seed") {
		t.Fatalf("expected refill seed subcommand in help output, got %s", text)
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

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_runnable": 1`)) {
		t.Fatalf("expected runnable count to use full local state, got %s", string(output))
	}
}

func TestLocalIssueClientFetchIssueStatesReloadsTrackerBetweenReads(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-par-243", "identifier": "BIG-PAR-243", "state": "Todo"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := refill.LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	client := &localIssueClient{store: store}

	issues, err := client.fetchIssueStates("ignored", []string{"Todo"})
	if err != nil {
		t.Fatalf("fetch issue states: %v", err)
	}
	if len(issues) != 1 || issues[0].Identifier != "BIG-PAR-243" {
		t.Fatalf("expected initial todo issue, got %+v", issues)
	}

	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-par-243", "identifier": "BIG-PAR-243", "state": "Done"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("rewrite local issue store: %v", err)
	}

	issues, err = client.fetchIssueStates("ignored", []string{"Done"})
	if err != nil {
		t.Fatalf("fetch reloaded issue states: %v", err)
	}
	if len(issues) != 1 || issues[0].Identifier != "BIG-PAR-243" || issues[0].StateName != "Done" {
		t.Fatalf("expected reloaded done issue, got %+v", issues)
	}
}

func TestRunRefillOnceLocalBackendSyncsQueueStatusFromLocalIssues(t *testing.T) {
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
  "recent_batches": {
    "completed": [],
    "active": ["BIG-GOM-501"],
    "standby": []
  },
  "issue_order": ["BIG-GOM-501", "BIG-GOM-502"],
  "issues": [
    {"identifier": "BIG-GOM-501", "title": "Queue metadata drift", "track": "Automation", "status": "Todo"},
    {"identifier": "BIG-GOM-502", "title": "Standby seed", "track": "Automation", "status": "Todo"}
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
    {"id": "big-gom-501", "identifier": "BIG-GOM-501", "state": "Done"},
    {"id": "big-gom-502", "identifier": "BIG-GOM-502", "state": "Todo"}
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

	runErr := runRefillOnce(queue, client, true, "", nil, true, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
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
	if !bytes.Contains(body, []byte(`"completed": [`)) || !bytes.Contains(body, []byte(`"BIG-GOM-501"`)) {
		t.Fatalf("expected recent batch completion sync, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(`"standby": [`)) || !bytes.Contains(body, []byte(`"BIG-GOM-502"`)) {
		t.Fatalf("expected recent batch standby sync, got %s", string(body))
	}
	if !bytes.Contains(output, []byte(`"queue_status_updates": 1`)) {
		t.Fatalf("expected refill output to include queue status updates, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_recent_batch_updates": 3`)) {
		t.Fatalf("expected refill output to include recent batch updates, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_synced": true`)) {
		t.Fatalf("expected refill output to report queue status synced after apply, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_synced": true`)) {
		t.Fatalf("expected refill output to report recent batches synced after apply, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_written": true`)) {
		t.Fatalf("expected refill output to confirm write, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_written": true`)) {
		t.Fatalf("expected refill output to confirm recent batch write, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"markdown_written": true`)) {
		t.Fatalf("expected refill output to confirm markdown write, got %s", string(output))
	}
}

func TestRunRefillOnceLocalBackendNormalizesActiveStatePayloads(t *testing.T) {
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
  "recent_batches": {
    "completed": [],
    "active": [],
    "standby": []
  },
  "issue_order": ["BIG-PAR-385", "BIG-PAR-386"],
  "issues": [
    {"identifier": "BIG-PAR-385", "title": "Normalize local tracker state filters", "track": "Automation", "status": "Todo"},
    {"identifier": "BIG-PAR-386", "title": "Normalize active detection", "track": "Automation", "status": "Todo"}
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
    {"id": "big-par-385", "identifier": "BIG-PAR-385", "state": "in progress."},
    {"id": "big-par-386", "identifier": "BIG-PAR-386", "state": "TODO"}
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

	runErr := runRefillOnce(queue, client, false, "", nil, true, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"active_in_progress": [`)) || !bytes.Contains(output, []byte(`"BIG-PAR-385"`)) {
		t.Fatalf("expected normalized active issue in payload, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_synced": false`)) {
		t.Fatalf("expected queue status drift detection before sync, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_synced": false`)) {
		t.Fatalf("expected recent batch drift detection before sync, got %s", string(output))
	}
}

func TestRunRefillOnceLocalDryRunIgnoresEquivalentQueueStatusSpellings(t *testing.T) {
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
  "recent_batches": {
    "completed": ["BIG-PAR-388"],
    "active": [],
    "standby": ["BIG-PAR-387"]
  },
  "issue_order": ["BIG-PAR-387", "BIG-PAR-388"],
  "issues": [
    {"identifier": "BIG-PAR-387", "title": "Normalize queue status sync equivalence", "track": "Automation", "status": "todo."},
    {"identifier": "BIG-PAR-388", "title": "Terminal state normalization companion", "track": "Automation", "status": "DONE"}
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
    {"id": "big-par-387", "identifier": "BIG-PAR-387", "state": "Todo"},
    {"id": "big-par-388", "identifier": "BIG-PAR-388", "state": "done."}
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

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_synced": true`)) {
		t.Fatalf("expected equivalent status spellings to avoid drift, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_updates": 0`)) {
		t.Fatalf("expected zero queue status updates for equivalent spellings, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_synced": true`)) {
		t.Fatalf("expected recent batches to stay synced, got %s", string(output))
	}
}

func TestRunRefillOnceLocalBackendReportsRecentBatchWriteWithoutStatusWrite(t *testing.T) {
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
  "recent_batches": {
    "completed": [],
    "active": [],
    "standby": []
  },
  "issue_order": ["BIG-GOM-501", "BIG-GOM-502"],
  "issues": [
    {"identifier": "BIG-GOM-501", "title": "Queue metadata drift", "track": "Automation", "status": "Done"},
    {"identifier": "BIG-GOM-502", "title": "Standby seed", "track": "Automation", "status": "Todo"}
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
    {"id": "big-gom-501", "identifier": "BIG-GOM-501", "state": "Done"},
    {"id": "big-gom-502", "identifier": "BIG-GOM-502", "state": "Todo"}
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

	runErr := runRefillOnce(queue, client, true, "", nil, true, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_updates": 0`)) {
		t.Fatalf("expected zero status updates, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_recent_batch_updates": 2`)) {
		t.Fatalf("expected two recent batch updates, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_written": false`)) {
		t.Fatalf("expected queue status write to remain false, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_written": true`)) {
		t.Fatalf("expected recent batch write to be true, got %s", string(output))
	}
}

func TestRunRefillOnceLocalDryRunReportsQueueStatusSyncedWithoutSyncFlag(t *testing.T) {
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
  "recent_batches": {
    "completed": ["BIG-GOM-501"],
    "active": [],
    "standby": ["BIG-GOM-502"]
  },
  "issue_order": ["BIG-GOM-501", "BIG-GOM-502"],
  "issues": [
    {"identifier": "BIG-GOM-501", "title": "Queue metadata drift", "track": "Automation", "status": "Done"},
    {"identifier": "BIG-GOM-502", "title": "Standby seed", "track": "Automation", "status": "Todo"}
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
    {"id": "big-gom-501", "identifier": "BIG-GOM-501", "state": "Done"},
    {"id": "big-gom-502", "identifier": "BIG-GOM-502", "state": "Todo"}
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

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_synced": true`)) {
		t.Fatalf("expected queue status to report synced in dry-run output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_synced": true`)) {
		t.Fatalf("expected recent batches to report synced in dry-run output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_updates": 0`)) {
		t.Fatalf("expected zero queue status updates in dry-run output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_recent_batch_updates": 0`)) {
		t.Fatalf("expected zero queue recent batch updates in dry-run output, got %s", string(output))
	}
}

func TestRunRefillOnceLocalDryRunReportsQueueDriftWithoutMutatingQueue(t *testing.T) {
	tempDir := t.TempDir()
	queuePath := filepath.Join(tempDir, "queue.json")
	originalQueue := []byte(`{
  "project": {"slug_id": "project-slug"},
  "policy": {
    "target_in_progress": 2,
    "activate_state_name": "In Progress",
    "activate_state_id": "state-in-progress",
    "refill_states": ["Todo", "Backlog"]
  },
  "recent_batches": {
    "completed": [],
    "active": ["BIG-GOM-501"],
    "standby": []
  },
  "issue_order": ["BIG-GOM-501", "BIG-GOM-502"],
  "issues": [
    {"identifier": "BIG-GOM-501", "title": "Queue metadata drift", "track": "Automation", "status": "Todo"},
    {"identifier": "BIG-GOM-502", "title": "Standby seed", "track": "Automation", "status": "Todo"}
  ]
}`)
	if err := os.WriteFile(queuePath, originalQueue, 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}
	queue, err := refill.LoadQueue(queuePath)
	if err != nil {
		t.Fatalf("load queue: %v", err)
	}

	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-gom-501", "identifier": "BIG-GOM-501", "state": "Done"},
    {"id": "big-gom-502", "identifier": "BIG-GOM-502", "state": "Todo"}
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

	runErr := runRefillOnce(queue, client, false, "", nil, false, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if runErr != nil {
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_synced": false`)) {
		t.Fatalf("expected queue status to report drift in dry-run output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_synced": false`)) {
		t.Fatalf("expected recent batches to report drift in dry-run output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_status_updates": 1`)) {
		t.Fatalf("expected one queue status update in dry-run output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"queue_recent_batch_updates": 3`)) {
		t.Fatalf("expected three recent batch updates in dry-run output, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_updated": false`)) {
		t.Fatalf("expected dry-run recent batch update flag to remain false, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"recent_batches_written": false`)) {
		t.Fatalf("expected dry-run recent batch write flag to remain false, got %s", string(output))
	}
	if !bytes.Contains(output, []byte(`"markdown_written": false`)) {
		t.Fatalf("expected dry-run markdown write flag to remain false, got %s", string(output))
	}

	body, err := os.ReadFile(queuePath)
	if err != nil {
		t.Fatalf("read queue file: %v", err)
	}
	if !bytes.Equal(body, originalQueue) {
		t.Fatalf("expected dry-run to leave queue file unchanged, got %s", string(body))
	}
}

func TestRunRefillOnceUpdatesRecentBatchesFromLocalTracker(t *testing.T) {
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
  "recent_batches": {
    "completed": ["BIG-PAR-200"],
    "active": [],
    "standby": ["BIG-PAR-201"]
  },
  "issue_order": ["BIG-PAR-241", "BIG-PAR-242"],
  "issues": [
    {"identifier": "BIG-PAR-241", "title": "tracker lock", "track": "Automation", "status": "Todo"},
    {"identifier": "BIG-PAR-242", "title": "queue sync", "track": "Automation", "status": "Todo"}
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
    {"id": "big-par-241", "identifier": "BIG-PAR-241", "state": "In Progress"},
    {"id": "big-par-242", "identifier": "BIG-PAR-242", "state": "Done"}
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

	runErr := runRefillOnce(queue, client, true, "", nil, false, queuePath, filepath.Join(tempDir, "queue.md"), storePath)
	_ = writer.Close()
	if runErr != nil {
		output, _ := io.ReadAll(reader)
		t.Fatalf("run refill once: %v (stdout=%s)", runErr, string(output))
	}

	body, err := os.ReadFile(queuePath)
	if err != nil {
		t.Fatalf("read updated queue file: %v", err)
	}
	var updated struct {
		RecentBatches struct {
			Active    []string `json:"active"`
			Completed []string `json:"completed"`
			Standby   []string `json:"standby"`
		} `json:"recent_batches"`
	}
	if err := json.Unmarshal(body, &updated); err != nil {
		t.Fatalf("decode updated queue: %v", err)
	}
	if !reflect.DeepEqual(updated.RecentBatches.Active, []string{"BIG-PAR-241"}) {
		t.Fatalf("unexpected active batches: %v", updated.RecentBatches.Active)
	}
	if !reflect.DeepEqual(updated.RecentBatches.Completed, []string{"BIG-PAR-242"}) {
		t.Fatalf("unexpected completed batches: %v", updated.RecentBatches.Completed)
	}
	if len(updated.RecentBatches.Standby) != 0 {
		t.Fatalf("expected empty standby batches, got %v", updated.RecentBatches.Standby)
	}
}

func TestRunRefillSeedCreatesQueueAndLocalIssue(t *testing.T) {
	tempDir := t.TempDir()
	queuePath := filepath.Join(tempDir, "queue.json")
	markdownPath := filepath.Join(tempDir, "queue.md")
	if err := os.WriteFile(queuePath, []byte(`{
  "project": {"slug_id": "project-slug"},
  "policy": {
    "target_in_progress": 2,
    "activate_state_name": "In Progress",
    "activate_state_id": "state-in-progress",
    "refill_states": ["Todo", "Backlog"]
  },
  "recent_batches": {
    "completed": [],
    "active": [],
    "standby": []
  },
  "issue_order": [],
  "issues": []
}`), 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}
	storePath := filepath.Join(tempDir, "local-issues.json")

	if err := runRefillSeed([]string{
		"--repo", tempDir,
		"--queue", queuePath,
		"--markdown", markdownPath,
		"--local-issues", storePath,
		"--identifier", "BIG-PAR-238",
		"--title", "bigclawctl refill: seed queue entries from CLI",
		"--description", "seed queue and tracker metadata from one command",
		"--labels", "parallel,tooling,refill",
		"--state", "Todo",
		"--recent-batch", "active",
		"--created-at", "2026-03-23T01:10:00Z",
		"--json",
	}); err != nil {
		t.Fatalf("run refill seed: %v", err)
	}

	queueBody, err := os.ReadFile(queuePath)
	if err != nil {
		t.Fatalf("read queue file: %v", err)
	}
	if !bytes.Contains(queueBody, []byte(`"identifier": "BIG-PAR-238"`)) {
		t.Fatalf("expected queue issue record, got %s", string(queueBody))
	}
	if !bytes.Contains(queueBody, []byte(`"active": [`)) || !bytes.Contains(queueBody, []byte(`"BIG-PAR-238"`)) {
		t.Fatalf("expected recent batch metadata, got %s", string(queueBody))
	}

	storeBody, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(storeBody, []byte(`"identifier": "BIG-PAR-238"`)) {
		t.Fatalf("expected local issue record, got %s", string(storeBody))
	}
	if !bytes.Contains(storeBody, []byte(`"labels": [`)) || !bytes.Contains(storeBody, []byte(`"refill"`)) {
		t.Fatalf("expected labels in local issue record, got %s", string(storeBody))
	}
	markdownBody, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	if !bytes.Contains(markdownBody, []byte("Current repo tranche status as of March 23, 2026")) {
		t.Fatalf("expected generated current batch date, got %s", string(markdownBody))
	}
	if !bytes.Contains(markdownBody, []byte("`BIG-PAR-238` — bigclawctl refill: seed queue entries from CLI")) {
		t.Fatalf("expected markdown issue summary, got %s", string(markdownBody))
	}
}

func TestRunRefillSeedSetStateIfExistsIgnoresEquivalentSpellings(t *testing.T) {
	tempDir := t.TempDir()
	queuePath := filepath.Join(tempDir, "queue.json")
	markdownPath := filepath.Join(tempDir, "queue.md")
	if err := os.WriteFile(queuePath, []byte(`{
  "project": {"slug_id": "project-slug"},
  "policy": {
    "target_in_progress": 2,
    "activate_state_name": "In Progress",
    "activate_state_id": "state-in-progress",
    "refill_states": ["Todo", "Backlog"]
  },
  "recent_batches": {
    "completed": [],
    "active": ["BIG-PAR-388"],
    "standby": []
  },
  "issue_order": ["BIG-PAR-388"],
  "issues": [
    {"identifier": "BIG-PAR-388", "title": "Normalize seed and ensure state equivalence", "track": "Automation", "status": "todo."}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-388",
      "identifier": "BIG-PAR-388",
      "title": "Normalize seed and ensure state equivalence",
      "state": "todo.",
      "updated_at": "2026-03-26T00:45:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	if err := runRefillSeed([]string{
		"--repo", tempDir,
		"--queue", queuePath,
		"--markdown", markdownPath,
		"--local-issues", storePath,
		"--identifier", "BIG-PAR-388",
		"--title", "Normalize seed and ensure state equivalence",
		"--track", "Automation",
		"--state", "Todo",
		"--recent-batch", "active",
		"--set-state-if-exists",
		"--created-at", "2026-03-26T00:50:00Z",
		"--json",
	}); err != nil {
		t.Fatalf("run refill seed: %v", err)
	}

	queueBody, err := os.ReadFile(queuePath)
	if err != nil {
		t.Fatalf("read queue file: %v", err)
	}
	if !bytes.Contains(queueBody, []byte(`"status": "todo."`)) {
		t.Fatalf("expected equivalent queue status to remain unchanged, got %s", string(queueBody))
	}

	storeBody, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(storeBody, []byte(`"state": "todo."`)) {
		t.Fatalf("expected equivalent local issue state to remain unchanged, got %s", string(storeBody))
	}
	if !bytes.Contains(storeBody, []byte(`"updated_at": "2026-03-26T00:45:00Z"`)) {
		t.Fatalf("expected unchanged timestamp for equivalent local issue state, got %s", string(storeBody))
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

func TestRunLocalIssuesEnsureIgnoresEquivalentStateSpellings(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-388",
      "identifier": "BIG-PAR-388",
      "title": "Normalize seed and ensure state equivalence",
      "state": "in progress.",
      "updated_at": "2026-03-26T00:55:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	if err := runLocalIssues([]string{
		"ensure",
		"--local-issues", storePath,
		"--identifier", "BIG-PAR-388",
		"--state", "In Progress",
		"--set-state-if-exists",
		"--created-at", "2026-03-26T01:00:00Z",
		"--json",
	}); err != nil {
		t.Fatalf("run local-issues ensure equivalent state: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !bytes.Contains(body, []byte(`"state": "in progress."`)) {
		t.Fatalf("expected equivalent state spelling to remain unchanged, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(`"updated_at": "2026-03-26T00:55:00Z"`)) {
		t.Fatalf("expected unchanged timestamp for equivalent state spelling, got %s", string(body))
	}
}

func TestRunLocalIssuesListNormalizesStateFilters(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-385",
      "identifier": "BIG-PAR-385",
      "title": "Normalize local tracker state filters for refill commands",
      "state": "in progress."
    },
    {
      "id": "big-par-386",
      "identifier": "BIG-PAR-386",
      "title": "Normalize refill active and recent-batch state detection",
      "state": "TODO"
    },
    {
      "id": "big-par-387",
      "identifier": "BIG-PAR-387",
      "title": "Unrelated done issue",
      "state": "Done"
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

	if err := runLocalIssues([]string{
		"list",
		"--local-issues", storePath,
		"--states", "In Progress,todo.",
		"--json",
	}); err != nil {
		t.Fatalf("run local-issues list: %v", err)
	}

	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	if !bytes.Contains(output, []byte(`"BIG-PAR-385"`)) || !bytes.Contains(output, []byte(`"BIG-PAR-386"`)) {
		t.Fatalf("expected normalized state matches in output, got %s", string(output))
	}
	if bytes.Contains(output, []byte(`"BIG-PAR-387"`)) {
		t.Fatalf("expected done issue to be filtered out, got %s", string(output))
	}
}
