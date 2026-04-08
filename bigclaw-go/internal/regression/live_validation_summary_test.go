package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveValidationSummaryStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	summaryPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-summary.json")

	var summary struct {
		RunID           string   `json:"run_id"`
		Status          string   `json:"status"`
		BundlePath      string   `json:"bundle_path"`
		SummaryPath     string   `json:"summary_path"`
		CloseoutCommand []string `json:"closeout_commands"`
		Local           struct {
			Enabled            bool   `json:"enabled"`
			BundleReportPath   string `json:"bundle_report_path"`
			CanonicalReportPath string `json:"canonical_report_path"`
			Status             string `json:"status"`
			TaskID             string `json:"task_id"`
			AuditLogPath       string `json:"audit_log_path"`
			ServiceLogPath     string `json:"service_log_path"`
		} `json:"local"`
		Kubernetes struct {
			Enabled             bool   `json:"enabled"`
			BundleReportPath    string `json:"bundle_report_path"`
			CanonicalReportPath string `json:"canonical_report_path"`
			Status              string `json:"status"`
			TaskID              string `json:"task_id"`
			AuditLogPath        string `json:"audit_log_path"`
			ServiceLogPath      string `json:"service_log_path"`
		} `json:"kubernetes"`
		Ray struct {
			Enabled             bool   `json:"enabled"`
			BundleReportPath    string `json:"bundle_report_path"`
			CanonicalReportPath string `json:"canonical_report_path"`
			Status              string `json:"status"`
			TaskID              string `json:"task_id"`
			AuditLogPath        string `json:"audit_log_path"`
			ServiceLogPath      string `json:"service_log_path"`
		} `json:"ray"`
		Broker struct {
			Enabled              bool   `json:"enabled"`
			BundleSummaryPath    string `json:"bundle_summary_path"`
			CanonicalSummaryPath string `json:"canonical_summary_path"`
			ValidationPackPath   string `json:"validation_pack_path"`
			ConfigurationState   string `json:"configuration_state"`
			Status               string `json:"status"`
			Reason               string `json:"reason"`
		} `json:"broker"`
		SharedQueueCompanion struct {
			Available            bool              `json:"available"`
			CanonicalReportPath  string            `json:"canonical_report_path"`
			CanonicalSummaryPath string            `json:"canonical_summary_path"`
			BundleReportPath     string            `json:"bundle_report_path"`
			BundleSummaryPath    string            `json:"bundle_summary_path"`
			Status               string            `json:"status"`
			Count                int               `json:"count"`
			CrossNodeCompletions int               `json:"cross_node_completions"`
			DuplicateStarted     int               `json:"duplicate_started_tasks"`
			DuplicateCompleted   int               `json:"duplicate_completed_tasks"`
			MissingCompleted     int               `json:"missing_completed_tasks"`
			SubmittedByNode      map[string]int    `json:"submitted_by_node"`
			CompletedByNode      map[string]int    `json:"completed_by_node"`
			Nodes                []string          `json:"nodes"`
		} `json:"shared_queue_companion"`
	}

	readJSONFile(t, summaryPath, &summary)

	if summary.RunID != "20260316T140138Z" || summary.Status != "succeeded" {
		t.Fatalf("unexpected live validation summary metadata: %+v", summary)
	}
	if summary.BundlePath != "docs/reports/live-validation-runs/20260316T140138Z" || summary.SummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/summary.json" {
		t.Fatalf("unexpected bundle paths: bundle=%s summary=%s", summary.BundlePath, summary.SummaryPath)
	}
	if len(summary.CloseoutCommand) != 3 ||
		summary.CloseoutCommand[0] != "cd bigclaw-go && ./scripts/e2e/run_all.sh" ||
		summary.CloseoutCommand[1] != "cd bigclaw-go && go test ./..." ||
		summary.CloseoutCommand[2] != "git push origin <branch> && git log -1 --stat" {
		t.Fatalf("unexpected closeout commands: %+v", summary.CloseoutCommand)
	}

	if !summary.Local.Enabled || summary.Local.CanonicalReportPath != "docs/reports/sqlite-smoke-report.json" || summary.Local.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json" || summary.Local.Status != "succeeded" || summary.Local.TaskID == "" {
		t.Fatalf("unexpected local lane summary: %+v", summary.Local)
	}
	if !summary.Kubernetes.Enabled || summary.Kubernetes.CanonicalReportPath != "docs/reports/kubernetes-live-smoke-report.json" || summary.Kubernetes.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json" || summary.Kubernetes.Status != "succeeded" || summary.Kubernetes.TaskID == "" {
		t.Fatalf("unexpected kubernetes lane summary: %+v", summary.Kubernetes)
	}
	if !summary.Ray.Enabled || summary.Ray.CanonicalReportPath != "docs/reports/ray-live-smoke-report.json" || summary.Ray.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json" || summary.Ray.Status != "succeeded" || summary.Ray.TaskID == "" {
		t.Fatalf("unexpected ray lane summary: %+v", summary.Ray)
	}

	if summary.Broker.Enabled || summary.Broker.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json" || summary.Broker.CanonicalSummaryPath != "docs/reports/broker-validation-summary.json" || summary.Broker.ValidationPackPath != "docs/reports/broker-failover-fault-injection-validation-pack.md" || summary.Broker.ConfigurationState != "not_configured" || summary.Broker.Status != "skipped" || summary.Broker.Reason != "not_configured" {
		t.Fatalf("unexpected broker summary: %+v", summary.Broker)
	}

	if !summary.SharedQueueCompanion.Available || summary.SharedQueueCompanion.CanonicalReportPath != "docs/reports/multi-node-shared-queue-report.json" || summary.SharedQueueCompanion.CanonicalSummaryPath != "docs/reports/shared-queue-companion-summary.json" || summary.SharedQueueCompanion.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json" || summary.SharedQueueCompanion.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json" || summary.SharedQueueCompanion.Status != "succeeded" {
		t.Fatalf("unexpected shared-queue companion summary paths: %+v", summary.SharedQueueCompanion)
	}
	if summary.SharedQueueCompanion.Count != 200 || summary.SharedQueueCompanion.CrossNodeCompletions != 99 || summary.SharedQueueCompanion.DuplicateStarted != 0 || summary.SharedQueueCompanion.DuplicateCompleted != 0 || summary.SharedQueueCompanion.MissingCompleted != 0 {
		t.Fatalf("unexpected shared-queue companion counts: %+v", summary.SharedQueueCompanion)
	}
	if len(summary.SharedQueueCompanion.Nodes) != 2 || summary.SharedQueueCompanion.Nodes[0] != "node-a" || summary.SharedQueueCompanion.Nodes[1] != "node-b" {
		t.Fatalf("unexpected shared-queue companion nodes: %+v", summary.SharedQueueCompanion.Nodes)
	}
	if summary.SharedQueueCompanion.SubmittedByNode["node-a"] != 100 || summary.SharedQueueCompanion.SubmittedByNode["node-b"] != 100 || summary.SharedQueueCompanion.CompletedByNode["node-a"] != 73 || summary.SharedQueueCompanion.CompletedByNode["node-b"] != 127 {
		t.Fatalf("unexpected shared-queue per-node counts: submitted=%+v completed=%+v", summary.SharedQueueCompanion.SubmittedByNode, summary.SharedQueueCompanion.CompletedByNode)
	}

	summaryContents := readRepoFile(t, repoRoot, "docs/reports/live-validation-summary.json")
	if strings.Contains(summaryContents, "python -c \"print('hello from ray')\"") {
		t.Fatal("live-validation-summary.json should not advertise the retired inline-Python ray smoke default")
	}
	if !strings.Contains(summaryContents, "sh -c 'echo hello from ray'") {
		t.Fatal("live-validation-summary.json should retain the shell-native ray smoke entrypoint")
	}

	canonicalRayReport := readRepoFile(t, repoRoot, "docs/reports/ray-live-smoke-report.json")
	if strings.Contains(canonicalRayReport, "python -c \"print('hello from ray')\"") {
		t.Fatal("ray-live-smoke-report.json should not advertise the retired inline-Python ray smoke default")
	}
	if !strings.Contains(canonicalRayReport, "sh -c 'echo hello from ray'") {
		t.Fatal("ray-live-smoke-report.json should retain the shell-native ray smoke entrypoint")
	}

	bundledRayReport := readRepoFile(t, repoRoot, "docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json")
	if strings.Contains(bundledRayReport, "python -c \"print('hello from ray')\"") {
		t.Fatal("bundled ray-live-smoke-report.json should not advertise the retired inline-Python ray smoke default")
	}
	if !strings.Contains(bundledRayReport, "sh -c 'echo hello from ray'") {
		t.Fatal("bundled ray-live-smoke-report.json should retain the shell-native ray smoke entrypoint")
	}
}
