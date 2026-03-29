package executionparity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQueueToSchedulerExecutionRecordsFullChain(t *testing.T) {
	t.Parallel()

	queue := &Queue{}
	ledger := &Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	reportPath := filepath.Join(t.TempDir(), "reports", "run-1.md")

	queue.Enqueue(Task{
		ID:            "BIG-502",
		Source:        "linear",
		Title:         "Record execution",
		Description:   "full chain",
		Priority:      0,
		RiskLevel:     RiskMedium,
		RequiredTools: []string{"browser"},
	})

	task := queue.Dequeue()
	if task == nil {
		t.Fatal("expected dequeued task")
	}

	record, err := (Scheduler{}).Execute(*task, "run-1", ledger, reportPath)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}

	if record.Decision.Medium != "browser" || !record.Decision.Approved {
		t.Fatalf("unexpected decision: %+v", record.Decision)
	}
	if record.Run.Status != "approved" {
		t.Fatalf("run status = %q, want approved", record.Run.Status)
	}
	if len(entries) != 1 {
		t.Fatalf("ledger entries = %d, want 1", len(entries))
	}
	if entries[0].Traces[0].Span != "scheduler.decide" {
		t.Fatalf("unexpected trace: %+v", entries[0].Traces)
	}
	if len(entries[0].Artifacts) != 2 || entries[0].Artifacts[0].Kind != "page" || entries[0].Artifacts[1].Kind != "report" {
		t.Fatalf("unexpected artifacts: %+v", entries[0].Artifacts)
	}
	if entries[0].Audits[0].Details["reason"] != "browser automation task" {
		t.Fatalf("unexpected audit: %+v", entries[0].Audits)
	}
	reportBody := readFile(t, reportPath)
	if !strings.Contains(reportBody, "Status: approved") {
		t.Fatalf("unexpected report body: %s", reportBody)
	}
	htmlBody := readFile(t, strings.TrimSuffix(reportPath, ".md")+".html")
	if !strings.Contains(htmlBody, "Status: approved") {
		t.Fatalf("unexpected html body: %s", htmlBody)
	}
}

func TestHighRiskExecutionRecordsPendingApproval(t *testing.T) {
	t.Parallel()

	ledger := &Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	task := Task{
		ID:          "BIG-502-risk",
		Source:      "jira",
		Title:       "Prod change",
		Description: "manual review",
		RiskLevel:   RiskHigh,
	}

	record, err := (Scheduler{}).Execute(task, "run-2", ledger, "")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}

	if record.Decision.Approved {
		t.Fatalf("expected pending approval decision, got %+v", record.Decision)
	}
	if record.Run.Status != "needs-approval" {
		t.Fatalf("run status = %q, want needs-approval", record.Run.Status)
	}
	if len(entries) != 1 || entries[0].Audits[0].Outcome != "pending" {
		t.Fatalf("unexpected ledger entries: %+v", entries)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
