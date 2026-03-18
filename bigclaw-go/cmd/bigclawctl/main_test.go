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
