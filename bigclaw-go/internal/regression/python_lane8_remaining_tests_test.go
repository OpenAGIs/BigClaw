package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLane8PythonReplacementTrancheRemoved(t *testing.T) {
	repoRoot := repoRoot(t)
	for _, path := range []string{
		"tests/test_control_center.py",
		"tests/test_live_shadow_bundle.py",
		"tests/test_orchestration.py",
		"tests/test_queue.py",
		"tests/test_repo_links.py",
		"tests/test_repo_collaboration.py",
		"tests/test_repo_rollout.py",
		"tests/test_models.py",
		"tests/test_planning.py",
		"tests/test_operations.py",
		"tests/test_observability.py",
		"tests/test_evaluation.py",
		"tests/test_risk.py",
		"tests/test_runtime_matrix.py",
		"tests/test_scheduler.py",
	} {
		_, err := os.Stat(filepath.Join(repoRoot, path))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s to stay removed, stat err=%v", path, err)
		}
	}
}

func TestLane8CrossProcessCoordinationSurfaceStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "cross-process-coordination-capability-surface.json")

	var report struct {
		Status                 string            `json:"status"`
		RuntimeReadinessLevels map[string]string `json:"runtime_readiness_levels"`
		Summary                struct {
			SharedQueueCrossNodeCompletions int `json:"shared_queue_cross_node_completions"`
			TakeoverPassingScenarios        int `json:"takeover_passing_scenarios"`
			TakeoverStaleWriteRejections    int `json:"takeover_stale_write_rejections"`
			SharedQueueDuplicateCompleted   int `json:"shared_queue_duplicate_completed_tasks"`
		} `json:"summary"`
		CurrentCeiling []string `json:"current_ceiling"`
		Capabilities   []struct {
			Capability                string `json:"capability"`
			CurrentState              string `json:"current_state"`
			RuntimeReadiness          string `json:"runtime_readiness"`
			DeterministicLocalHarness bool   `json:"deterministic_local_harness"`
		} `json:"capabilities"`
	}
	readJSONFile(t, reportPath, &report)

	if report.Status != "local-capability-surface" {
		t.Fatalf("unexpected coordination surface status: %+v", report)
	}
	for _, readiness := range []string{"contract_only", "harness_proven", "live_proven", "supporting_surface"} {
		if report.RuntimeReadinessLevels[readiness] == "" {
			t.Fatalf("missing runtime_readiness_levels[%q]", readiness)
		}
	}
	if report.Summary.SharedQueueCrossNodeCompletions != 99 ||
		report.Summary.TakeoverPassingScenarios != 3 ||
		report.Summary.TakeoverStaleWriteRejections != 2 ||
		report.Summary.SharedQueueDuplicateCompleted != 0 {
		t.Fatalf("unexpected coordination summary: %+v", report.Summary)
	}
	if !matchesNote(report.CurrentCeiling, "no partitioned topic model") {
		t.Fatalf("expected partitioned topic ceiling, got %+v", report.CurrentCeiling)
	}

	byCapability := map[string]struct {
		CurrentState         string
		RuntimeReadiness     string
		DeterministicHarness bool
	}{}
	for _, capability := range report.Capabilities {
		byCapability[capability.Capability] = struct {
			CurrentState         string
			RuntimeReadiness     string
			DeterministicHarness bool
		}{
			CurrentState:         capability.CurrentState,
			RuntimeReadiness:     capability.RuntimeReadiness,
			DeterministicHarness: capability.DeterministicLocalHarness,
		}
	}
	if item := byCapability["partitioned_topic_routing"]; item.CurrentState != "not_available" || item.RuntimeReadiness != "contract_only" {
		t.Fatalf("unexpected partitioned topic capability: %+v", item)
	}
	if item := byCapability["broker_backed_subscriber_ownership"]; item.CurrentState != "not_available" || item.RuntimeReadiness != "contract_only" {
		t.Fatalf("unexpected broker-backed ownership capability: %+v", item)
	}
	if item := byCapability["shared_queue_task_coordination"]; item.RuntimeReadiness != "live_proven" {
		t.Fatalf("unexpected shared queue capability: %+v", item)
	}
	if item := byCapability["subscriber_takeover_semantics"]; !item.DeterministicHarness || item.RuntimeReadiness != "live_proven" {
		t.Fatalf("unexpected takeover capability: %+v", item)
	}
}

func TestLane8ValidationBundleContinuationScorecardStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-scorecard.json")

	var report struct {
		Status  string `json:"status"`
		Summary struct {
			RecentBundleCount                           int    `json:"recent_bundle_count"`
			LatestRunID                                 string `json:"latest_run_id"`
			LatestAllExecutorTracksSucceeded            bool   `json:"latest_all_executor_tracks_succeeded"`
			RecentBundleChainHasNoFailures              bool   `json:"recent_bundle_chain_has_no_failures"`
			AllExecutorTracksHaveRepeatedRecentCoverage bool   `json:"all_executor_tracks_have_repeated_recent_coverage"`
		} `json:"summary"`
		SharedQueueCompanion struct {
			CrossNodeCompletions int    `json:"cross_node_completions"`
			DuplicateCompleted   int    `json:"duplicate_completed_tasks"`
			Mode                 string `json:"mode"`
			SummaryPath          string `json:"summary_path"`
		} `json:"shared_queue_companion"`
		ExecutorLanes []struct {
			Lane                   string `json:"lane"`
			LatestStatus           string `json:"latest_status"`
			EnabledRuns            int    `json:"enabled_runs"`
			ConsecutiveSuccesses   int    `json:"consecutive_successes"`
			AllRecentRunsSucceeded bool   `json:"all_recent_runs_succeeded"`
		} `json:"executor_lanes"`
		ContinuationChecks []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
			Detail string `json:"detail"`
		} `json:"continuation_checks"`
	}
	readJSONFile(t, reportPath, &report)

	if report.Status != "local-continuation-scorecard" {
		t.Fatalf("unexpected continuation scorecard status: %+v", report)
	}
	if report.Summary.RecentBundleCount != 3 ||
		report.Summary.LatestRunID != "20260316T140138Z" ||
		!report.Summary.LatestAllExecutorTracksSucceeded ||
		!report.Summary.RecentBundleChainHasNoFailures ||
		!report.Summary.AllExecutorTracksHaveRepeatedRecentCoverage {
		t.Fatalf("unexpected continuation scorecard summary: %+v", report.Summary)
	}
	if report.SharedQueueCompanion.CrossNodeCompletions != 99 ||
		report.SharedQueueCompanion.DuplicateCompleted != 0 ||
		report.SharedQueueCompanion.Mode != "bundle-companion-summary" ||
		report.SharedQueueCompanion.SummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected continuation shared-queue companion: %+v", report.SharedQueueCompanion)
	}

	lanes := map[string]struct {
		LatestStatus         string
		EnabledRuns          int
		ConsecutiveSuccesses int
		AllRecentSucceeded   bool
	}{}
	for _, lane := range report.ExecutorLanes {
		lanes[lane.Lane] = struct {
			LatestStatus         string
			EnabledRuns          int
			ConsecutiveSuccesses int
			AllRecentSucceeded   bool
		}{
			LatestStatus:         lane.LatestStatus,
			EnabledRuns:          lane.EnabledRuns,
			ConsecutiveSuccesses: lane.ConsecutiveSuccesses,
			AllRecentSucceeded:   lane.AllRecentRunsSucceeded,
		}
	}
	if len(lanes) != 3 ||
		lanes["local"].ConsecutiveSuccesses != 3 ||
		lanes["kubernetes"].ConsecutiveSuccesses != 3 ||
		lanes["ray"].ConsecutiveSuccesses != 2 ||
		lanes["ray"].EnabledRuns != 2 {
		t.Fatalf("unexpected continuation lanes: %+v", lanes)
	}
	for lane, item := range lanes {
		if item.LatestStatus != "succeeded" || !item.AllRecentSucceeded {
			t.Fatalf("lane %s not fully successful: %+v", lane, item)
		}
	}

	var repeatedCoverage, workflowBoundary bool
	for _, check := range report.ContinuationChecks {
		switch check.Name {
		case "all_executor_tracks_have_repeated_recent_coverage":
			repeatedCoverage = check.Passed && strings.Contains(check.Detail, "'ray': 2")
		case "continuation_surface_is_workflow_triggered":
			workflowBoundary = check.Passed && strings.Contains(check.Detail, "workflow execution")
		}
	}
	if !repeatedCoverage || !workflowBoundary {
		t.Fatalf("unexpected continuation checks: %+v", report.ContinuationChecks)
	}
}

func TestLane8LiveShadowScorecardStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "live-shadow-mirror-scorecard.json")

	var report struct {
		Ticket  string `json:"ticket"`
		Status  string `json:"status"`
		Summary struct {
			MatrixMismatched      int  `json:"matrix_mismatched"`
			CorpusCoveragePresent bool `json:"corpus_coverage_present"`
			StaleInputs           int  `json:"stale_inputs"`
		} `json:"summary"`
		Freshness []struct {
			Status string `json:"status"`
		} `json:"freshness"`
		ParityEntries []struct {
			Parity struct {
				Status string `json:"status"`
			} `json:"parity"`
		} `json:"parity_entries"`
	}
	readJSONFile(t, reportPath, &report)

	if report.Ticket != "BIG-PAR-092" || report.Status != "repo-native-live-shadow-scorecard" {
		t.Fatalf("unexpected live shadow scorecard identity: %+v", report)
	}
	if report.Summary.MatrixMismatched != 0 || !report.Summary.CorpusCoveragePresent || report.Summary.StaleInputs != 0 {
		t.Fatalf("unexpected live shadow scorecard summary: %+v", report.Summary)
	}
	if len(report.Freshness) != 2 {
		t.Fatalf("unexpected freshness entries: %+v", report.Freshness)
	}
	for _, item := range report.Freshness {
		if item.Status != "fresh" {
			t.Fatalf("expected all freshness entries to be fresh, got %+v", report.Freshness)
		}
	}
	for _, item := range report.ParityEntries {
		if item.Parity.Status != "parity-ok" {
			t.Fatalf("expected parity-ok entries, got %+v", report.ParityEntries)
		}
	}
}

func TestLane8ShadowMatrixCorpusCoverageStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "shadow-matrix-report.json")

	var report struct {
		CorpusCoverage struct {
			ManifestName              string `json:"manifest_name"`
			UncoveredCorpusSliceCount int    `json:"uncovered_corpus_slice_count"`
			UncoveredSlices           []struct {
				SliceID string `json:"slice_id"`
			} `json:"uncovered_slices"`
		} `json:"corpus_coverage"`
	}
	readJSONFile(t, reportPath, &report)

	if report.CorpusCoverage.ManifestName != "anonymized-production-corpus-v1" ||
		report.CorpusCoverage.UncoveredCorpusSliceCount != 1 ||
		len(report.CorpusCoverage.UncoveredSlices) != 1 ||
		report.CorpusCoverage.UncoveredSlices[0].SliceID != "browser-human-review" {
		t.Fatalf("unexpected shadow matrix corpus coverage: %+v", report.CorpusCoverage)
	}
}

func TestLane8SubscriberTakeoverHarnessStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "multi-subscriber-takeover-validation-report.json")

	var report struct {
		Status  string `json:"status"`
		Summary struct {
			ScenarioCount          int `json:"scenario_count"`
			PassingScenarios       int `json:"passing_scenarios"`
			FailingScenarios       int `json:"failing_scenarios"`
			StaleWriteRejections   int `json:"stale_write_rejections"`
			DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
		} `json:"summary"`
		Scenarios []struct {
			ID                     string `json:"id"`
			AllAssertionsPassed    bool   `json:"all_assertions_passed"`
			DuplicateDeliveryCount int    `json:"duplicate_delivery_count"`
			StaleWriteRejections   int    `json:"stale_write_rejections"`
			TakeoverSubscriber     string `json:"takeover_subscriber"`
			CheckpointAfter        struct {
				Owner string `json:"owner"`
			} `json:"checkpoint_after"`
			DuplicateEvents []string `json:"duplicate_events"`
		} `json:"scenarios"`
	}
	readJSONFile(t, reportPath, &report)

	if report.Status != "local-executable" ||
		report.Summary.ScenarioCount != 3 ||
		report.Summary.PassingScenarios != 3 ||
		report.Summary.FailingScenarios != 0 ||
		report.Summary.StaleWriteRejections != 2 ||
		report.Summary.DuplicateDeliveryCount != 4 {
		t.Fatalf("unexpected takeover report summary: %+v", report)
	}

	var staleWriter, splitBrain bool
	for _, scenario := range report.Scenarios {
		if !scenario.AllAssertionsPassed {
			t.Fatalf("scenario did not pass assertions: %+v", scenario)
		}
		switch scenario.ID {
		case "lease-expiry-stale-writer-rejected":
			staleWriter = scenario.StaleWriteRejections == 1 &&
				scenario.CheckpointAfter.Owner == scenario.TakeoverSubscriber &&
				containsLane8String(scenario.DuplicateEvents, "evt-81")
		case "split-brain-dual-replay-window":
			splitBrain = scenario.DuplicateDeliveryCount == 2 &&
				scenario.CheckpointAfter.Owner == scenario.TakeoverSubscriber
		}
	}
	if !staleWriter || !splitBrain {
		t.Fatalf("expected stale-writer and split-brain scenarios to stay aligned: %+v", report.Scenarios)
	}
}

func TestLane8FollowupDigestsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		issueID string
		title   string
		path    string
		links   []string
		phrases []string
		indexes []string
	}{
		{
			issueID: "OPE-264",
			title:   "BIG-PAR-075",
			path:    "docs/reports/tracing-backend-follow-up-digest.md",
			links: []string{
				"docs/reports/go-control-plane-observability-report.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"internal/observability/recorder.go",
				"internal/api/server.go",
			},
			phrases: []string{
				"no external tracing backend",
				"no cross-process span propagation beyond in-memory trace grouping",
			},
			indexes: []string{
				"docs/reports/go-control-plane-observability-report.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"../docs/openclaw-parallel-gap-analysis.md",
			},
		},
		{
			issueID: "OPE-265",
			title:   "BIG-PAR-076",
			path:    "docs/reports/telemetry-pipeline-controls-follow-up-digest.md",
			links: []string{
				"docs/reports/go-control-plane-observability-report.md",
				"docs/reports/review-readiness.md",
				"internal/api/server.go",
				"internal/observability/recorder.go",
				"internal/worker/runtime.go",
			},
			phrases: []string{
				"no full OpenTelemetry-native metrics / tracing pipeline",
				"no configurable sampling or high-cardinality controls",
			},
			indexes: []string{
				"docs/reports/go-control-plane-observability-report.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"../docs/openclaw-parallel-gap-analysis.md",
			},
		},
		{
			issueID: "OPE-266",
			title:   "BIG-PAR-092",
			path:    "docs/reports/live-shadow-comparison-follow-up-digest.md",
			links: []string{
				"docs/reports/migration-readiness-report.md",
				"docs/migration-shadow.md",
				"docs/reports/shadow-compare-report.json",
				"docs/reports/shadow-matrix-report.json",
				"docs/reports/live-shadow-mirror-scorecard.json",
				"docs/reports/migration-plan-review-notes.md",
			},
			phrases: []string{
				"repo-native live shadow mirror scorecard",
				"no live legacy-vs-Go production traffic comparison",
			},
			indexes: []string{
				"docs/reports/migration-readiness-report.md",
				"docs/migration-shadow.md",
				"docs/reports/migration-plan-review-notes.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"../docs/openclaw-parallel-gap-analysis.md",
			},
		},
		{
			issueID: "OPE-254",
			title:   "BIG-PAR-088",
			path:    "docs/reports/rollback-safeguard-follow-up-digest.md",
			links: []string{
				"docs/reports/migration-readiness-report.md",
				"docs/migration.md",
				"docs/reports/migration-plan-review-notes.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
			},
			phrases: []string{
				"rollback remains operator-driven",
				"no tenant-scoped automated rollback trigger",
			},
			indexes: []string{
				"docs/reports/migration-readiness-report.md",
				"docs/migration.md",
				"docs/reports/migration-plan-review-notes.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"../docs/openclaw-parallel-gap-analysis.md",
			},
		},
		{
			issueID: "OPE-268",
			title:   "BIG-PAR-079",
			path:    "docs/reports/production-corpus-migration-coverage-digest.md",
			links: []string{
				"docs/reports/migration-readiness-report.md",
				"docs/reports/shadow-matrix-report.json",
				"docs/reports/shadow-compare-report.json",
				"docs/migration-shadow.md",
				"docs/reports/issue-coverage.md",
				"examples/shadow-corpus-manifest.json",
			},
			phrases: []string{
				"fixture-backed evidence only",
				"no real production issue/task corpus coverage",
			},
			indexes: []string{
				"docs/reports/migration-readiness-report.md",
				"docs/migration-shadow.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"../docs/openclaw-parallel-gap-analysis.md",
			},
		},
		{
			issueID: "OPE-269",
			title:   "BIG-PAR-080",
			path:    "docs/reports/subscriber-takeover-executability-follow-up-digest.md",
			links: []string{
				"docs/reports/multi-subscriber-takeover-validation-report.md",
				"docs/reports/multi-subscriber-takeover-validation-report.json",
				"go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...",
				"docs/reports/event-bus-reliability-report.md",
				"docs/reports/issue-coverage.md",
				"docs/reports/review-readiness.md",
			},
			phrases: []string{
				"live two-node shared-queue proof",
				"live schema parity exists but shared durable ownership does not",
			},
			indexes: []string{
				"docs/reports/multi-subscriber-takeover-validation-report.md",
				"docs/reports/event-bus-reliability-report.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"../docs/openclaw-parallel-gap-analysis.md",
				"docs/e2e-validation.md",
			},
		},
		{
			issueID: "OPE-261",
			title:   "BIG-PAR-085",
			path:    "docs/reports/cross-process-coordination-boundary-digest.md",
			links: []string{
				"docs/reports/event-bus-reliability-report.md",
				"docs/reports/multi-node-coordination-report.md",
				"docs/reports/cross-process-coordination-capability-surface.json",
				"go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface ...",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
			},
			phrases: []string{
				"no partitioned topic model",
				"no broker-backed cross-process subscriber coordination",
			},
			indexes: []string{
				"docs/reports/event-bus-reliability-report.md",
				"docs/reports/multi-node-coordination-report.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"../docs/openclaw-parallel-gap-analysis.md",
			},
		},
		{
			issueID: "OPE-271",
			title:   "BIG-PAR-082",
			path:    "docs/reports/validation-bundle-continuation-digest.md",
			links: []string{
				"docs/reports/live-validation-index.md",
				"docs/reports/live-validation-summary.json",
				"docs/reports/shared-queue-companion-summary.json",
				"docs/reports/validation-bundle-continuation-scorecard.json",
				"go run ./cmd/bigclawctl automation e2e continuation-scorecard ...",
				"docs/reports/validation-bundle-continuation-policy-gate.json",
				"go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...",
				"docs/reports/multi-node-coordination-report.md",
				"docs/reports/review-readiness.md",
			},
			phrases: []string{
				"rolling continuation scorecard",
				"continuation across future validation bundles remains workflow-triggered",
			},
			indexes: []string{
				"docs/reports/live-validation-index.md",
				"docs/reports/multi-node-coordination-report.md",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"../docs/openclaw-parallel-gap-analysis.md",
			},
		},
	}

	for _, tc := range cases {
		text := readRepoFile(t, repoRoot, tc.path)
		if !strings.Contains(text, tc.issueID) || !strings.Contains(text, tc.title) {
			t.Fatalf("%s missing issue metadata %s / %s", tc.path, tc.issueID, tc.title)
		}
		for _, link := range tc.links {
			if !strings.Contains(text, "`"+link+"`") {
				t.Fatalf("%s missing link %q", tc.path, link)
			}
		}
		for _, phrase := range tc.phrases {
			if !strings.Contains(text, phrase) {
				t.Fatalf("%s missing phrase %q", tc.path, phrase)
			}
		}
		digestPath := tc.path
		for _, index := range tc.indexes {
			indexText := readRepoFile(t, repoRoot, index)
			if !strings.Contains(indexText, digestPath) && !strings.Contains(indexText, "bigclaw-go/"+digestPath) {
				t.Fatalf("%s missing digest reference %q", index, digestPath)
			}
		}
	}
}

func containsLane8String(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
