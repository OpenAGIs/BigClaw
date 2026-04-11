package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSharedQueueReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "multi-node-shared-queue-report.json")
	bundleReportPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-runs", "20260316T140138Z", "multi-node-shared-queue-report.json")
	summaryPath := filepath.Join(repoRoot, "docs", "reports", "shared-queue-companion-summary.json")

	type node struct {
		Name       string `json:"name"`
		BaseURL    string `json:"base_url"`
		AuditPath  string `json:"audit_path"`
		ServiceLog string `json:"service_log"`
	}
	type report struct {
		GeneratedAt          string         `json:"generated_at"`
		RootStateDir         string         `json:"root_state_dir"`
		QueuePath            string         `json:"queue_path"`
		Count                int            `json:"count"`
		SubmittedByNode      map[string]int `json:"submitted_by_node"`
		CompletedByNode      map[string]int `json:"completed_by_node"`
		CrossNodeCompletions int            `json:"cross_node_completions"`
		DuplicateStarted     []string       `json:"duplicate_started_tasks"`
		DuplicateCompleted   []string       `json:"duplicate_completed_tasks"`
		MissingCompleted     []string       `json:"missing_completed_tasks"`
		AllOK                bool           `json:"all_ok"`
		Nodes                []node         `json:"nodes"`
	}

	var canonical report
	readJSONFile(t, reportPath, &canonical)

	if canonical.GeneratedAt != "2026-03-16T15:59:45Z" {
		t.Fatalf("unexpected canonical shared-queue generated_at: %s", canonical.GeneratedAt)
	}
	if canonical.Count != 200 || canonical.CrossNodeCompletions != 99 || !canonical.AllOK {
		t.Fatalf("unexpected canonical shared-queue rollup: %+v", canonical)
	}
	if len(canonical.DuplicateStarted) != 0 || len(canonical.DuplicateCompleted) != 0 || len(canonical.MissingCompleted) != 0 {
		t.Fatalf("unexpected canonical duplicate or missing tasks: %+v", canonical)
	}
	if canonical.SubmittedByNode["node-a"] != 100 || canonical.SubmittedByNode["node-b"] != 100 || canonical.CompletedByNode["node-a"] != 73 || canonical.CompletedByNode["node-b"] != 127 {
		t.Fatalf("unexpected canonical per-node totals: submitted=%+v completed=%+v", canonical.SubmittedByNode, canonical.CompletedByNode)
	}
	if !strings.HasSuffix(canonical.QueuePath, "/shared-queue.db") || !strings.Contains(canonical.RootStateDir, "bigclawd-multinode-") {
		t.Fatalf("unexpected canonical queue layout: root=%s queue=%s", canonical.RootStateDir, canonical.QueuePath)
	}
	if len(canonical.Nodes) != 2 || canonical.Nodes[0].Name != "node-a" || canonical.Nodes[1].Name != "node-b" {
		t.Fatalf("unexpected canonical shared-queue nodes: %+v", canonical.Nodes)
	}
	for _, node := range canonical.Nodes {
		if !strings.HasPrefix(node.BaseURL, "http://127.0.0.1:") {
			t.Fatalf("unexpected canonical node base URL: %+v", node)
		}
		if !strings.HasSuffix(node.AuditPath, node.Name+"-audit.jsonl") {
			t.Fatalf("unexpected canonical node audit path: %+v", node)
		}
		if !strings.Contains(node.ServiceLog, node.Name+"-") || !strings.HasSuffix(node.ServiceLog, ".log") {
			t.Fatalf("unexpected canonical node service log path: %+v", node)
		}
	}

	var bundle report
	readJSONFile(t, bundleReportPath, &bundle)

	if bundle.GeneratedAt != "2026-03-13T09:45:19Z" {
		t.Fatalf("unexpected bundled shared-queue generated_at: %s", bundle.GeneratedAt)
	}
	if bundle.Count != canonical.Count || bundle.CrossNodeCompletions != canonical.CrossNodeCompletions || bundle.AllOK != canonical.AllOK {
		t.Fatalf("unexpected bundled shared-queue rollup: bundled=%+v canonical=%+v", bundle, canonical)
	}
	if len(bundle.DuplicateStarted) != len(canonical.DuplicateStarted) || len(bundle.DuplicateCompleted) != len(canonical.DuplicateCompleted) || len(bundle.MissingCompleted) != len(canonical.MissingCompleted) {
		t.Fatalf("unexpected bundled duplicate or missing tasks: bundled=%+v canonical=%+v", bundle, canonical)
	}
	if bundle.SubmittedByNode["node-a"] != canonical.SubmittedByNode["node-a"] || bundle.SubmittedByNode["node-b"] != canonical.SubmittedByNode["node-b"] || bundle.CompletedByNode["node-a"] != canonical.CompletedByNode["node-a"] || bundle.CompletedByNode["node-b"] != canonical.CompletedByNode["node-b"] {
		t.Fatalf("bundled per-node totals drifted: bundled submitted=%+v completed=%+v canonical submitted=%+v completed=%+v", bundle.SubmittedByNode, bundle.CompletedByNode, canonical.SubmittedByNode, canonical.CompletedByNode)
	}
	if len(bundle.Nodes) != len(canonical.Nodes) || bundle.Nodes[0].Name != canonical.Nodes[0].Name || bundle.Nodes[1].Name != canonical.Nodes[1].Name {
		t.Fatalf("bundled node identities drifted: bundled=%+v canonical=%+v", bundle.Nodes, canonical.Nodes)
	}
	for _, node := range bundle.Nodes {
		if !strings.HasPrefix(node.BaseURL, "http://127.0.0.1:") {
			t.Fatalf("unexpected bundled node base URL: %+v", node)
		}
		if !strings.HasSuffix(node.AuditPath, node.Name+"-audit.jsonl") {
			t.Fatalf("unexpected bundled node audit path: %+v", node)
		}
		if !strings.Contains(node.ServiceLog, node.Name+"-") || !strings.HasSuffix(node.ServiceLog, ".log") {
			t.Fatalf("unexpected bundled node service log path: %+v", node)
		}
	}

	var summary struct {
		Count                int            `json:"count"`
		CrossNodeCompletions int            `json:"cross_node_completions"`
		DuplicateStarted     int            `json:"duplicate_started_tasks"`
		DuplicateCompleted   int            `json:"duplicate_completed_tasks"`
		MissingCompleted     int            `json:"missing_completed_tasks"`
		SubmittedByNode      map[string]int `json:"submitted_by_node"`
		CompletedByNode      map[string]int `json:"completed_by_node"`
		Nodes                []string       `json:"nodes"`
	}
	readJSONFile(t, summaryPath, &summary)

	if summary.Count != canonical.Count || summary.CrossNodeCompletions != canonical.CrossNodeCompletions || summary.DuplicateStarted != len(canonical.DuplicateStarted) || summary.DuplicateCompleted != len(canonical.DuplicateCompleted) || summary.MissingCompleted != len(canonical.MissingCompleted) {
		t.Fatalf("shared-queue summary drifted from canonical report: summary=%+v canonical=%+v", summary, canonical)
	}
	if summary.SubmittedByNode["node-a"] != canonical.SubmittedByNode["node-a"] || summary.SubmittedByNode["node-b"] != canonical.SubmittedByNode["node-b"] || summary.CompletedByNode["node-a"] != canonical.CompletedByNode["node-a"] || summary.CompletedByNode["node-b"] != canonical.CompletedByNode["node-b"] {
		t.Fatalf("shared-queue summary per-node totals drifted: summary submitted=%+v completed=%+v canonical submitted=%+v canonical completed=%+v", summary.SubmittedByNode, summary.CompletedByNode, canonical.SubmittedByNode, canonical.CompletedByNode)
	}
	if len(summary.Nodes) != 2 || summary.Nodes[0] != canonical.Nodes[0].Name || summary.Nodes[1] != canonical.Nodes[1].Name {
		t.Fatalf("shared-queue summary nodes drifted: summary=%+v canonical=%+v", summary.Nodes, canonical.Nodes)
	}
}
