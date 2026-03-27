package crossprocesscoordination

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildReportSummarizesCheckedInInputs(t *testing.T) {
	repoRoot := t.TempDir()
	writeCrossProcessFixture(t, repoRoot, "bigclaw-go/docs/reports/multi-node-shared-queue-report.json", map[string]any{
		"count":                     200,
		"cross_node_completions":    99,
		"duplicate_completed_tasks": []string{},
		"duplicate_started_tasks":   []string{},
	})
	writeCrossProcessFixture(t, repoRoot, "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json", map[string]any{
		"summary": map[string]any{
			"scenario_count":           3,
			"passing_scenarios":        3,
			"duplicate_delivery_count": 4,
			"stale_write_rejections":   2,
		},
	})
	writeCrossProcessFixture(t, repoRoot, "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json", map[string]any{
		"summary": map[string]any{
			"scenario_count":         3,
			"passing_scenarios":      3,
			"stale_write_rejections": 3,
		},
	})

	report, err := BuildReport(BuildOptions{
		RepoRoot:    repoRoot,
		GeneratedAt: time.Date(2026, 3, 28, 2, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}

	if got := report["status"]; got != "local-capability-surface" {
		t.Fatalf("unexpected status: %v", got)
	}
	summary := report["summary"].(map[string]any)
	if summary["shared_queue_cross_node_completions"].(float64) != 99 || summary["takeover_passing_scenarios"].(float64) != 3 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	capabilities := report["capabilities"].([]any)
	if len(capabilities) != 7 {
		t.Fatalf("unexpected capability count: %d", len(capabilities))
	}
}

func TestWriteReportPrettyPrintsJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docs/reports/cross-process-coordination-capability-surface.json")
	if err := WriteReport(path, map[string]any{"status": "ok"}, true); err != nil {
		t.Fatalf("write report: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if string(body) != "{\n  \"status\": \"ok\"\n}\n" {
		t.Fatalf("unexpected report body: %q", string(body))
	}
}

func writeCrossProcessFixture(t *testing.T, repoRoot, relative string, payload map[string]any) {
	t.Helper()
	path := filepath.Join(repoRoot, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
