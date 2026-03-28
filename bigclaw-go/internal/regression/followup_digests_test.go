package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFollowupDigestsCaptureLinksAndConstraints(t *testing.T) {
	repoRoot := repoRoot(t)
	for issueID, digest := range followupDigestExpectations() {
		body := readRepoFile(t, repoRoot, digest.path)
		if !strings.Contains(body, issueID) {
			t.Fatalf("%s missing issue id %q", digest.path, issueID)
		}
		if !strings.Contains(body, digest.title) {
			t.Fatalf("%s missing title %q", digest.path, digest.title)
		}
		for _, link := range digest.links {
			if !strings.Contains(body, "`"+link+"`") {
				t.Fatalf("%s missing link %q", digest.path, link)
			}
		}
		for _, phrase := range digest.phrases {
			if !strings.Contains(body, phrase) {
				t.Fatalf("%s missing phrase %q", digest.path, phrase)
			}
		}
	}
}

func TestFollowupIndexesReferenceNewDigests(t *testing.T) {
	repoRoot := repoRoot(t)
	for _, digest := range followupDigestExpectations() {
		trimmedDigest := strings.TrimPrefix(digest.path, "bigclaw-go/")
		for _, indexPath := range digest.indexes {
			body := readRepoFile(t, repoRoot, indexPath)
			if !strings.Contains(body, trimmedDigest) && !strings.Contains(body, digest.path) {
				t.Fatalf("%s missing digest reference %q or %q", indexPath, trimmedDigest, digest.path)
			}
		}
	}
}

type followupDigestExpectation struct {
	path    string
	title   string
	links   []string
	phrases []string
	indexes []string
}

func followupDigestExpectations() map[string]followupDigestExpectation {
	return map[string]followupDigestExpectation{
		"OPE-264": {
			path:  filepath.ToSlash(filepath.Join("docs", "reports", "tracing-backend-follow-up-digest.md")),
			title: "BIG-PAR-075",
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
				filepath.ToSlash(filepath.Join("docs", "reports", "go-control-plane-observability-report.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "review-readiness.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "issue-coverage.md")),
				filepath.ToSlash(filepath.Join("..", "docs", "openclaw-parallel-gap-analysis.md")),
			},
		},
		"OPE-265": {
			path:  filepath.ToSlash(filepath.Join("docs", "reports", "telemetry-pipeline-controls-follow-up-digest.md")),
			title: "BIG-PAR-076",
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
				filepath.ToSlash(filepath.Join("docs", "reports", "go-control-plane-observability-report.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "review-readiness.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "issue-coverage.md")),
				filepath.ToSlash(filepath.Join("..", "docs", "openclaw-parallel-gap-analysis.md")),
			},
		},
		"OPE-266": {
			path:  filepath.ToSlash(filepath.Join("docs", "reports", "live-shadow-comparison-follow-up-digest.md")),
			title: "BIG-PAR-092",
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
				filepath.ToSlash(filepath.Join("docs", "reports", "migration-readiness-report.md")),
				filepath.ToSlash(filepath.Join("docs", "migration-shadow.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "migration-plan-review-notes.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "review-readiness.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "issue-coverage.md")),
				filepath.ToSlash(filepath.Join("..", "docs", "openclaw-parallel-gap-analysis.md")),
			},
		},
		"OPE-254": {
			path:  filepath.ToSlash(filepath.Join("docs", "reports", "rollback-safeguard-follow-up-digest.md")),
			title: "BIG-PAR-088",
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
				filepath.ToSlash(filepath.Join("docs", "reports", "migration-readiness-report.md")),
				filepath.ToSlash(filepath.Join("docs", "migration.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "migration-plan-review-notes.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "review-readiness.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "issue-coverage.md")),
				filepath.ToSlash(filepath.Join("..", "docs", "openclaw-parallel-gap-analysis.md")),
			},
		},
		"OPE-268": {
			path:  filepath.ToSlash(filepath.Join("docs", "reports", "production-corpus-migration-coverage-digest.md")),
			title: "BIG-PAR-079",
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
				filepath.ToSlash(filepath.Join("docs", "reports", "migration-readiness-report.md")),
				filepath.ToSlash(filepath.Join("docs", "migration-shadow.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "review-readiness.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "issue-coverage.md")),
				filepath.ToSlash(filepath.Join("..", "docs", "openclaw-parallel-gap-analysis.md")),
			},
		},
		"OPE-269": {
			path:  filepath.ToSlash(filepath.Join("docs", "reports", "subscriber-takeover-executability-follow-up-digest.md")),
			title: "BIG-PAR-080",
			links: []string{
				"docs/reports/multi-subscriber-takeover-validation-report.md",
				"docs/reports/multi-subscriber-takeover-validation-report.json",
				"scripts/e2e/subscriber_takeover_fault_matrix.py",
				"docs/reports/event-bus-reliability-report.md",
				"docs/reports/issue-coverage.md",
				"docs/reports/review-readiness.md",
				"docs/openclaw-parallel-gap-analysis.md",
			},
			phrases: []string{
				"live two-node shared-queue proof",
				"live schema parity exists but shared durable ownership does not",
			},
			indexes: []string{
				filepath.ToSlash(filepath.Join("docs", "reports", "multi-subscriber-takeover-validation-report.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "event-bus-reliability-report.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "review-readiness.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "issue-coverage.md")),
				filepath.ToSlash(filepath.Join("..", "docs", "openclaw-parallel-gap-analysis.md")),
				filepath.ToSlash(filepath.Join("docs", "e2e-validation.md")),
			},
		},
		"OPE-261": {
			path:  filepath.ToSlash(filepath.Join("docs", "reports", "cross-process-coordination-boundary-digest.md")),
			title: "BIG-PAR-085",
			links: []string{
				"docs/reports/event-bus-reliability-report.md",
				"docs/reports/multi-node-coordination-report.md",
				"docs/reports/cross-process-coordination-capability-surface.json",
				"scripts/e2e/cross_process_coordination_surface.py",
				"docs/reports/review-readiness.md",
				"docs/reports/issue-coverage.md",
				"docs/openclaw-parallel-gap-analysis.md",
			},
			phrases: []string{
				"no partitioned topic model",
				"no broker-backed cross-process subscriber coordination",
			},
			indexes: []string{
				filepath.ToSlash(filepath.Join("docs", "reports", "event-bus-reliability-report.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "multi-node-coordination-report.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "review-readiness.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "issue-coverage.md")),
				filepath.ToSlash(filepath.Join("..", "docs", "openclaw-parallel-gap-analysis.md")),
			},
		},
		"OPE-271": {
			path:  filepath.ToSlash(filepath.Join("docs", "reports", "validation-bundle-continuation-digest.md")),
			title: "BIG-PAR-082",
			links: []string{
				"docs/reports/live-validation-index.md",
				"docs/reports/live-validation-summary.json",
				"docs/reports/shared-queue-companion-summary.json",
				"docs/reports/validation-bundle-continuation-scorecard.json",
				"scripts/e2e/validation_bundle_continuation_scorecard.py",
				"docs/reports/validation-bundle-continuation-policy-gate.json",
				"scripts/e2e/validation_bundle_continuation_policy_gate.py",
				"docs/reports/multi-node-coordination-report.md",
				"docs/reports/review-readiness.md",
				"docs/openclaw-parallel-gap-analysis.md",
			},
			phrases: []string{
				"rolling continuation scorecard",
				"continuation across future validation bundles remains workflow-triggered",
			},
			indexes: []string{
				filepath.ToSlash(filepath.Join("docs", "reports", "live-validation-index.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "multi-node-coordination-report.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "review-readiness.md")),
				filepath.ToSlash(filepath.Join("docs", "reports", "issue-coverage.md")),
				filepath.ToSlash(filepath.Join("..", "docs", "openclaw-parallel-gap-analysis.md")),
			},
		},
	}
}
