package issuebootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSyncDryRunSkipsExistingAndPreviewsMissing(t *testing.T) {
	report, err := Sync(context.Background(), SyncOptions{
		Owner:    "OpenAGIs",
		Repo:     "BigClaw",
		PlanName: "v1",
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("sync dry-run: %v", err)
	}
	if report.Existing != 0 || report.Skipped != 0 {
		t.Fatalf("unexpected existing/skip counts: %+v", report)
	}
	if report.CreatedCount != len(DefaultPlans()["v1"].Issues) {
		t.Fatalf("unexpected created count: %+v", report)
	}
	if report.Created[0].Number != 0 {
		t.Fatalf("expected dry-run preview to avoid issue numbers: %+v", report.Created[0])
	}
}

func TestSyncCreatesMissingIssues(t *testing.T) {
	posted := []map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			posted = append(posted, payload)
			_ = json.NewEncoder(w).Encode(map[string]any{"number": len(posted), "title": payload["title"]})
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()

	report, err := Sync(context.Background(), SyncOptions{
		Owner:      "OpenAGIs",
		Repo:       "BigClaw",
		PlanName:   "v2-ops",
		APIBaseURL: server.URL,
		Token:      "test-token",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("sync create: %v", err)
	}
	if report.CreatedCount != len(DefaultPlans()["v2-ops"].Issues) {
		t.Fatalf("unexpected created count: %+v", report)
	}
	if len(posted) == 0 {
		t.Fatal("expected POST payloads")
	}
	labels, ok := posted[0]["labels"].([]any)
	if !ok || len(labels) != len(DefaultPlans()["v2-ops"].Labels) {
		t.Fatalf("expected labels in POST payload, got %+v", posted[0])
	}
}

func TestSyncRejectsUnknownPlan(t *testing.T) {
	_, err := Sync(context.Background(), SyncOptions{
		Owner:    "OpenAGIs",
		Repo:     "BigClaw",
		PlanName: "unknown",
		DryRun:   true,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown plan") {
		t.Fatalf("expected unknown plan error, got %v", err)
	}
}

func TestSyncDryRunWithoutTokenPreviewsWholePlanWithoutNetwork(t *testing.T) {
	report, err := Sync(context.Background(), SyncOptions{
		Owner:    "OpenAGIs",
		Repo:     "BigClaw",
		PlanName: "v2-ops",
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("sync dry-run without token: %v", err)
	}
	if report.Existing != 0 || report.Skipped != 0 {
		t.Fatalf("expected local-only preview counts, got %+v", report)
	}
	if report.CreatedCount != len(DefaultPlans()["v2-ops"].Issues) {
		t.Fatalf("unexpected created count: %+v", report)
	}
}
