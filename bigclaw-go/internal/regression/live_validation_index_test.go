package regression

import (
	"path/filepath"
	"testing"
)

func TestLiveValidationIndexStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	indexPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-index.json")

	var index struct {
		Latest struct {
			RunID            string   `json:"run_id"`
			GeneratedAt      string   `json:"generated_at"`
			Status           string   `json:"status"`
			BundlePath       string   `json:"bundle_path"`
			SummaryPath      string   `json:"summary_path"`
			CloseoutCommands []string `json:"closeout_commands"`
			Local            struct {
				Enabled             bool   `json:"enabled"`
				BundleReportPath    string `json:"bundle_report_path"`
				CanonicalReportPath string `json:"canonical_report_path"`
				Status              string `json:"status"`
				TaskID              string `json:"task_id"`
				AuditLogPath        string `json:"audit_log_path"`
				ServiceLogPath      string `json:"service_log_path"`
				FailureRootCause    struct {
					LocationKind string `json:"location_kind"`
				} `json:"failure_root_cause"`
				ValidationMatrix struct {
					Lane                  string `json:"lane"`
					Executor              string `json:"executor"`
					RootCauseLocationKind string `json:"root_cause_location_kind"`
				} `json:"validation_matrix"`
			} `json:"local"`
			Kubernetes struct {
				Enabled             bool   `json:"enabled"`
				BundleReportPath    string `json:"bundle_report_path"`
				CanonicalReportPath string `json:"canonical_report_path"`
				Status              string `json:"status"`
				TaskID              string `json:"task_id"`
				AuditLogPath        string `json:"audit_log_path"`
				ServiceLogPath      string `json:"service_log_path"`
				FailureRootCause    struct {
					LocationKind string `json:"location_kind"`
				} `json:"failure_root_cause"`
				ValidationMatrix struct {
					Lane                  string `json:"lane"`
					Executor              string `json:"executor"`
					RootCauseLocationKind string `json:"root_cause_location_kind"`
				} `json:"validation_matrix"`
			} `json:"kubernetes"`
			Ray struct {
				Enabled             bool   `json:"enabled"`
				BundleReportPath    string `json:"bundle_report_path"`
				CanonicalReportPath string `json:"canonical_report_path"`
				Status              string `json:"status"`
				TaskID              string `json:"task_id"`
				AuditLogPath        string `json:"audit_log_path"`
				ServiceLogPath      string `json:"service_log_path"`
				FailureRootCause    struct {
					LocationKind string `json:"location_kind"`
				} `json:"failure_root_cause"`
				ValidationMatrix struct {
					Lane                  string `json:"lane"`
					Executor              string `json:"executor"`
					RootCauseLocationKind string `json:"root_cause_location_kind"`
				} `json:"validation_matrix"`
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
				Available            bool   `json:"available"`
				CanonicalReportPath  string `json:"canonical_report_path"`
				CanonicalSummaryPath string `json:"canonical_summary_path"`
				BundleReportPath     string `json:"bundle_report_path"`
				BundleSummaryPath    string `json:"bundle_summary_path"`
				Status               string `json:"status"`
				Count                int    `json:"count"`
				CrossNodeCompletions int    `json:"cross_node_completions"`
			} `json:"shared_queue_companion"`
		} `json:"latest"`
		ContinuationGate struct {
			Path           string   `json:"path"`
			Status         string   `json:"status"`
			Recommendation string   `json:"recommendation"`
			FailingChecks  []string `json:"failing_checks"`
			NextActions    []string `json:"next_actions"`
			Enforcement    struct {
				Mode     string `json:"mode"`
				Outcome  string `json:"outcome"`
				ExitCode int    `json:"exit_code"`
			} `json:"enforcement"`
			Summary struct {
				LatestRunID                           string  `json:"latest_run_id"`
				LatestBundleAgeHours                  float64 `json:"latest_bundle_age_hours"`
				RecentBundleCount                     int     `json:"recent_bundle_count"`
				LatestAllExecutorTracksSucceeded      bool    `json:"latest_all_executor_tracks_succeeded"`
				RecentBundleChainHasNoFailures        bool    `json:"recent_bundle_chain_has_no_failures"`
				AllExecutorTracksHaveRepeatedCoverage bool    `json:"all_executor_tracks_have_repeated_recent_coverage"`
				Recommendation                        string  `json:"recommendation"`
				EnforcementMode                       string  `json:"enforcement_mode"`
				WorkflowOutcome                       string  `json:"workflow_outcome"`
				WorkflowExitCode                      int     `json:"workflow_exit_code"`
				PassingCheckCount                     int     `json:"passing_check_count"`
				FailingCheckCount                     int     `json:"failing_check_count"`
			} `json:"summary"`
			ReviewerPath struct {
				IndexPath   string `json:"index_path"`
				DigestPath  string `json:"digest_path"`
				DigestIssue struct {
					ID   string `json:"id"`
					Slug string `json:"slug"`
				} `json:"digest_issue"`
			} `json:"reviewer_path"`
		} `json:"continuation_gate"`
		RecentRuns []struct {
			RunID       string `json:"run_id"`
			GeneratedAt string `json:"generated_at"`
			Status      string `json:"status"`
			BundlePath  string `json:"bundle_path"`
			SummaryPath string `json:"summary_path"`
		} `json:"recent_runs"`
	}
	readJSONFile(t, indexPath, &index)

	if index.Latest.RunID != "20260316T140138Z" || index.Latest.Status != "succeeded" {
		t.Fatalf("unexpected live validation latest metadata: %+v", index.Latest)
	}
	if index.Latest.GeneratedAt != "2026-04-03T07:57:34.810349Z" || index.Latest.BundlePath != "docs/reports/live-validation-runs/20260316T140138Z" || index.Latest.SummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/summary.json" {
		t.Fatalf("unexpected live validation latest paths: %+v", index.Latest)
	}
	if len(index.Latest.CloseoutCommands) != 3 ||
		index.Latest.CloseoutCommands[0] != "cd bigclaw-go && ./scripts/e2e/run_all.sh" ||
		index.Latest.CloseoutCommands[1] != "cd bigclaw-go && go test ./..." ||
		index.Latest.CloseoutCommands[2] != "git push origin <branch> && git log -1 --stat" {
		t.Fatalf("unexpected live validation closeout commands: %+v", index.Latest.CloseoutCommands)
	}

	if !index.Latest.Local.Enabled || index.Latest.Local.CanonicalReportPath != "docs/reports/sqlite-smoke-report.json" || index.Latest.Local.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json" || index.Latest.Local.Status != "succeeded" || index.Latest.Local.TaskID == "" {
		t.Fatalf("unexpected local lane summary: %+v", index.Latest.Local)
	}
	if index.Latest.Local.ValidationMatrix.Lane != "local" || index.Latest.Local.ValidationMatrix.Executor != "local" || index.Latest.Local.ValidationMatrix.RootCauseLocationKind != "stderr_log" || index.Latest.Local.FailureRootCause.LocationKind != "stderr_log" {
		t.Fatalf("unexpected local matrix/root cause: %+v", index.Latest.Local)
	}
	if !index.Latest.Kubernetes.Enabled || index.Latest.Kubernetes.CanonicalReportPath != "docs/reports/kubernetes-live-smoke-report.json" || index.Latest.Kubernetes.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json" || index.Latest.Kubernetes.Status != "succeeded" || index.Latest.Kubernetes.TaskID == "" {
		t.Fatalf("unexpected kubernetes lane summary: %+v", index.Latest.Kubernetes)
	}
	if index.Latest.Kubernetes.ValidationMatrix.Lane != "k8s" || index.Latest.Kubernetes.ValidationMatrix.Executor != "kubernetes" || index.Latest.Kubernetes.ValidationMatrix.RootCauseLocationKind != "stderr_log" || index.Latest.Kubernetes.FailureRootCause.LocationKind != "stderr_log" {
		t.Fatalf("unexpected kubernetes matrix/root cause: %+v", index.Latest.Kubernetes)
	}
	if !index.Latest.Ray.Enabled || index.Latest.Ray.CanonicalReportPath != "docs/reports/ray-live-smoke-report.json" || index.Latest.Ray.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json" || index.Latest.Ray.Status != "succeeded" || index.Latest.Ray.TaskID == "" {
		t.Fatalf("unexpected ray lane summary: %+v", index.Latest.Ray)
	}
	if index.Latest.Ray.ValidationMatrix.Lane != "ray" || index.Latest.Ray.ValidationMatrix.Executor != "ray" || index.Latest.Ray.ValidationMatrix.RootCauseLocationKind != "stderr_log" || index.Latest.Ray.FailureRootCause.LocationKind != "stderr_log" {
		t.Fatalf("unexpected ray matrix/root cause: %+v", index.Latest.Ray)
	}
	if index.Latest.Broker.Enabled || index.Latest.Broker.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json" || index.Latest.Broker.CanonicalSummaryPath != "docs/reports/broker-validation-summary.json" || index.Latest.Broker.ValidationPackPath != "docs/reports/broker-failover-fault-injection-validation-pack.md" || index.Latest.Broker.ConfigurationState != "not_configured" || index.Latest.Broker.Status != "skipped" || index.Latest.Broker.Reason != "not_configured" {
		t.Fatalf("unexpected broker lane summary: %+v", index.Latest.Broker)
	}
	if !index.Latest.SharedQueueCompanion.Available || index.Latest.SharedQueueCompanion.CanonicalReportPath != "docs/reports/multi-node-shared-queue-report.json" || index.Latest.SharedQueueCompanion.CanonicalSummaryPath != "docs/reports/shared-queue-companion-summary.json" || index.Latest.SharedQueueCompanion.BundleReportPath != "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json" || index.Latest.SharedQueueCompanion.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json" || index.Latest.SharedQueueCompanion.Status != "succeeded" || index.Latest.SharedQueueCompanion.Count != 200 || index.Latest.SharedQueueCompanion.CrossNodeCompletions != 99 {
		t.Fatalf("unexpected shared-queue companion summary: %+v", index.Latest.SharedQueueCompanion)
	}

	if index.ContinuationGate.Path != "docs/reports/validation-bundle-continuation-policy-gate.json" || index.ContinuationGate.Status != "policy-go" || index.ContinuationGate.Recommendation != "go" {
		t.Fatalf("unexpected continuation gate metadata: %+v", index.ContinuationGate)
	}
	if len(index.ContinuationGate.FailingChecks) != 0 || len(index.ContinuationGate.NextActions) != 1 || index.ContinuationGate.NextActions[0] != "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions" {
		t.Fatalf("unexpected continuation gate actions: %+v", index.ContinuationGate)
	}
	if index.ContinuationGate.Enforcement.Mode != "hold" || index.ContinuationGate.Enforcement.Outcome != "pass" || index.ContinuationGate.Enforcement.ExitCode != 0 {
		t.Fatalf("unexpected continuation gate enforcement: %+v", index.ContinuationGate.Enforcement)
	}
	if index.ContinuationGate.Summary.LatestRunID != "20260316T140138Z" || index.ContinuationGate.Summary.RecentBundleCount != 3 || !index.ContinuationGate.Summary.LatestAllExecutorTracksSucceeded || !index.ContinuationGate.Summary.RecentBundleChainHasNoFailures || !index.ContinuationGate.Summary.AllExecutorTracksHaveRepeatedCoverage || index.ContinuationGate.Summary.Recommendation != "go" || index.ContinuationGate.Summary.EnforcementMode != "hold" || index.ContinuationGate.Summary.WorkflowOutcome != "pass" || index.ContinuationGate.Summary.WorkflowExitCode != 0 || index.ContinuationGate.Summary.PassingCheckCount != 6 || index.ContinuationGate.Summary.FailingCheckCount != 0 {
		t.Fatalf("unexpected continuation gate summary: %+v", index.ContinuationGate.Summary)
	}
	if index.ContinuationGate.Summary.LatestBundleAgeHours <= 0 || index.ContinuationGate.Summary.LatestBundleAgeHours >= 1 {
		t.Fatalf("unexpected continuation gate bundle age: %+v", index.ContinuationGate.Summary)
	}
	if index.ContinuationGate.ReviewerPath.IndexPath != "docs/reports/live-validation-index.md" || index.ContinuationGate.ReviewerPath.DigestPath != "docs/reports/validation-bundle-continuation-digest.md" || index.ContinuationGate.ReviewerPath.DigestIssue.ID != "OPE-271" || index.ContinuationGate.ReviewerPath.DigestIssue.Slug != "BIG-PAR-082" {
		t.Fatalf("unexpected continuation gate reviewer path: %+v", index.ContinuationGate.ReviewerPath)
	}

	if len(index.RecentRuns) != 3 {
		t.Fatalf("unexpected recent live validation run count: %+v", index.RecentRuns)
	}
	if index.RecentRuns[0].RunID != "20260316T140138Z" || index.RecentRuns[0].GeneratedAt != "2026-04-03T07:57:34.810349Z" || index.RecentRuns[0].Status != "succeeded" || index.RecentRuns[0].BundlePath != "docs/reports/live-validation-runs/20260316T140138Z" || index.RecentRuns[0].SummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/summary.json" {
		t.Fatalf("unexpected first recent run: %+v", index.RecentRuns[0])
	}
	if index.RecentRuns[1].RunID != "20260314T164647Z" || index.RecentRuns[1].GeneratedAt != "2026-03-14T16:46:57.671520+00:00" || index.RecentRuns[1].Status != "succeeded" || index.RecentRuns[1].BundlePath != "docs/reports/live-validation-runs/20260314T164647Z" || index.RecentRuns[1].SummaryPath != "docs/reports/live-validation-runs/20260314T164647Z/summary.json" {
		t.Fatalf("unexpected second recent run: %+v", index.RecentRuns[1])
	}
	if index.RecentRuns[2].RunID != "20260314T163430Z" || index.RecentRuns[2].GeneratedAt != "2026-03-14T16:34:42.080370+00:00" || index.RecentRuns[2].Status != "succeeded" || index.RecentRuns[2].BundlePath != "docs/reports/live-validation-runs/20260314T163430Z" || index.RecentRuns[2].SummaryPath != "docs/reports/live-validation-runs/20260314T163430Z/summary.json" {
		t.Fatalf("unexpected third recent run: %+v", index.RecentRuns[2])
	}
}
