package regression

import (
	"strings"
	"testing"
)

func TestObservabilityFollowUpDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/tracing-backend-follow-up-digest.md",
			substrings: []string{
				"OPE-264` / `BIG-PAR-075",
				"GET /v2/reports/distributed/export",
				"no external tracing backend",
				"no cross-process span propagation beyond in-memory trace grouping",
			},
		},
		{
			path: "docs/reports/telemetry-pipeline-controls-follow-up-digest.md",
			substrings: []string{
				"OPE-265` / `BIG-PAR-076",
				"telemetry-sampling-cardinality-evidence-pack.json",
				"no full OpenTelemetry-native metrics / tracing pipeline",
				"no configurable sampling or high-cardinality controls",
			},
		},
		{
			path: "docs/reports/go-control-plane-observability-report.md",
			substrings: []string{
				"GET /v2/reports/distributed/export",
				"telemetry-sampling-cardinality-evidence-pack.json",
				"tracing-backend-follow-up-digest.md",
				"telemetry-pipeline-controls-follow-up-digest.md",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"OPE-264` / `BIG-PAR-075",
				"OPE-265` / `BIG-PAR-076",
				"GET /v2/reports/distributed/export",
				"telemetry-sampling-cardinality-evidence-pack.json",
			},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{
				"OPE-264` / `BIG-PAR-075",
				"OPE-265` / `BIG-PAR-076",
				"go-control-plane-observability-report.md",
				"telemetry-sampling-cardinality-evidence-pack.json",
			},
		},
		{
			path: "../docs/openclaw-parallel-gap-analysis.md",
			substrings: []string{
				"OPE-264` / `BIG-PAR-075",
				"OPE-265` / `BIG-PAR-076",
				"tracing-backend-follow-up-digest.md",
				"telemetry-pipeline-controls-follow-up-digest.md",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
