package regression

import (
	"strings"
	"testing"
)

func TestLiveShadowRuntimeDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/migration-shadow.md",
			substrings: []string{
				"go run ./cmd/bigclawctl automation migration shadow-compare",
				"go run ./cmd/bigclawctl automation migration shadow-matrix",
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
				"GET /debug/status",
				"live_shadow_mirror_scorecard",
				"GET /v2/control-center",
				"distributed_diagnostics.live_shadow_mirror_scorecard",
			},
		},
		{
			path: "docs/reports/live-shadow-index.md",
			substrings: []string{
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
				"go test ./internal/regression -run TestRollbackDocsStayAligned",
			},
		},
		{
			path: "docs/reports/live-shadow-index.json",
			substrings: []string{
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
				"go test ./internal/regression -run TestRollbackDocsStayAligned",
			},
		},
		{
			path: "docs/reports/live-shadow-summary.json",
			substrings: []string{
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
				"go test ./internal/regression -run TestRollbackDocsStayAligned",
			},
		},
		{
			path: "docs/reports/live-shadow-runs/20260313T085655Z/README.md",
			substrings: []string{
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
				"go test ./internal/regression -run TestRollbackDocsStayAligned",
			},
		},
		{
			path: "docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
			substrings: []string{
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
				"go test ./internal/regression -run TestRollbackDocsStayAligned",
			},
		},
		{
			path: "docs/reports/migration-readiness-report.md",
			substrings: []string{
				"OPE-266` / `BIG-PAR-092",
				"GET /debug/status` live shadow mirror payload",
				"GET /v2/control-center` distributed diagnostics live shadow mirror payload",
				"`live_shadow_mirror_scorecard`",
				"`distributed_diagnostics.live_shadow_mirror_scorecard`",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"OPE-266` / `BIG-PAR-092",
				"GET /debug/status",
				"GET /v2/control-center",
				"distributed_diagnostics.live_shadow_mirror_scorecard",
			},
		},
		{
			path: "docs/reports/live-shadow-runs/20260313T085655Z/README.md",
			substrings: []string{
				"OPE-266` / `BIG-PAR-092",
				"live-shadow-comparison-follow-up-digest.md",
				"OPE-254` / `BIG-PAR-088",
				"rollback-safeguard-follow-up-digest.md",
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
		if tc.path == "docs/migration-shadow.md" ||
			tc.path == "docs/reports/live-shadow-index.md" ||
			tc.path == "docs/reports/live-shadow-index.json" ||
			tc.path == "docs/reports/live-shadow-summary.json" ||
			tc.path == "docs/reports/live-shadow-runs/20260313T085655Z/README.md" ||
			tc.path == "docs/reports/live-shadow-runs/20260313T085655Z/summary.json" {
			for _, needle := range []string{
				"python3 scripts/migration/shadow_compare.py",
				"python3 scripts/migration/shadow_matrix.py",
				"python3 scripts/migration/live_shadow_scorecard.py",
				"python3 scripts/migration/export_live_shadow_bundle",
			} {
				if strings.Contains(contents, needle) {
					t.Fatalf("%s should not reference retired Python migration helper %q", tc.path, needle)
				}
			}
		}
	}
}
