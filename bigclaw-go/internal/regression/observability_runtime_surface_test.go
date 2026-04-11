package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type observabilityTelemetryEvidencePack struct {
	Ticket         string `json:"ticket"`
	Track          string `json:"track"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		ObservabilityReport string   `json:"observability_report"`
		FollowUpDigest      string   `json:"follow_up_digest"`
		ReviewReadiness     string   `json:"review_readiness"`
		IssueCoverage       string   `json:"issue_coverage"`
		CodePaths           []string `json:"code_paths"`
		ValidationTests     []string `json:"validation_tests"`
	} `json:"evidence_inputs"`
	ReviewerChecks []struct {
		Name     string `json:"name"`
		Status   string `json:"status"`
		Evidence string `json:"evidence"`
	} `json:"reviewer_checks"`
	CurrentCeiling []string `json:"current_ceiling"`
}

func TestObservabilityRuntimeSurfaceInvariants(t *testing.T) {
	root := repoRoot(t)

	observabilityReport := readRepoFile(t, root, "docs/reports/go-control-plane-observability-report.md")
	tracingDigest := readRepoFile(t, root, "docs/reports/tracing-backend-follow-up-digest.md")
	telemetryDigest := readRepoFile(t, root, "docs/reports/telemetry-pipeline-controls-follow-up-digest.md")

	for _, needle := range []string{
		"GET /v2/reports/distributed/export",
		"docs/reports/tracing-backend-follow-up-digest.md",
		"docs/reports/telemetry-pipeline-controls-follow-up-digest.md",
		"docs/reports/telemetry-sampling-cardinality-evidence-pack.json",
	} {
		if !strings.Contains(observabilityReport, needle) {
			t.Fatalf("observability report missing %q", needle)
		}
	}

	for _, needle := range []string{
		"docs/reports/go-control-plane-observability-report.md",
		"docs/reports/review-readiness.md",
		"docs/reports/issue-coverage.md",
		"GET /v2/reports/distributed/export",
		"internal/api/distributed.go",
		"internal/api/server.go",
		"no external tracing backend",
		"no cross-process span propagation beyond in-memory `trace_id` grouping",
	} {
		if !strings.Contains(tracingDigest, needle) {
			t.Fatalf("tracing follow-up digest missing %q", needle)
		}
	}

	for _, needle := range []string{
		"docs/reports/go-control-plane-observability-report.md",
		"docs/reports/telemetry-sampling-cardinality-evidence-pack.json",
		"internal/api/metrics.go",
		"internal/observability/recorder.go",
		"internal/worker/runtime.go",
		"no full OpenTelemetry-native metrics / tracing pipeline",
		"no configurable sampling or high-cardinality controls",
	} {
		if !strings.Contains(telemetryDigest, needle) {
			t.Fatalf("telemetry follow-up digest missing %q", needle)
		}
	}
}

func TestObservabilityTelemetryEvidencePackReviewerLinks(t *testing.T) {
	root := repoRoot(t)

	var pack observabilityTelemetryEvidencePack
	readJSONFile(t, filepath.Join(root, "docs", "reports", "telemetry-sampling-cardinality-evidence-pack.json"), &pack)

	if pack.Ticket != "OPE-265" || pack.Track != "BIG-PAR-076" || pack.Status != "repo-evidence-pack" {
		t.Fatalf("unexpected telemetry evidence pack identity: %+v", pack)
	}
	if len(pack.ReviewerChecks) != 4 {
		t.Fatalf("unexpected reviewer check count: %+v", pack.ReviewerChecks)
	}
	for _, ceiling := range pack.CurrentCeiling {
		if strings.TrimSpace(ceiling) == "" {
			t.Fatalf("expected non-empty telemetry ceiling entries, got %+v", pack.CurrentCeiling)
		}
	}
	if len(pack.CurrentCeiling) != 3 {
		t.Fatalf("unexpected telemetry current ceiling payload: %+v", pack.CurrentCeiling)
	}

	requiredEvidence := map[string]string{
		"metrics_surface_avoids_task_and_trace_labels":      "bigclaw-go/internal/api/metrics.go",
		"trace_and_task_identifiers_stay_in_debug_payloads": "bigclaw-go/internal/api/server.go",
		"sampling_policy_is_not_configurable_yet":           "bigclaw-go/docs/reports/telemetry-pipeline-controls-follow-up-digest.md",
		"review_docs_call_out_non_production_posture":       "bigclaw-go/docs/reports/review-readiness.md",
	}
	for _, check := range pack.ReviewerChecks {
		if check.Status != "pass" {
			t.Fatalf("expected reviewer check %q to pass, got %+v", check.Name, check)
		}
		wantEvidence, ok := requiredEvidence[check.Name]
		if !ok {
			t.Fatalf("unexpected reviewer check: %+v", check)
		}
		if check.Evidence != wantEvidence {
			t.Fatalf("reviewer check %q evidence = %q, want %q", check.Name, check.Evidence, wantEvidence)
		}
		if _, err := os.Stat(resolveRepoPath(root, check.Evidence)); err != nil {
			t.Fatalf("expected reviewer check evidence %q to exist: %v", check.Evidence, err)
		}
	}

	for _, candidate := range []string{
		pack.EvidenceInputs.ObservabilityReport,
		pack.EvidenceInputs.FollowUpDigest,
		pack.EvidenceInputs.ReviewReadiness,
		pack.EvidenceInputs.IssueCoverage,
	} {
		body := readRepoFile(t, root, strings.TrimPrefix(candidate, "bigclaw-go/"))
		if !strings.Contains(body, "telemetry-sampling-cardinality-evidence-pack.json") {
			t.Fatalf("expected %q to reference telemetry evidence pack", candidate)
		}
	}
}
