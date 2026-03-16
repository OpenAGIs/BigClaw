package regression

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRollbackDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/rollback-safeguard-follow-up-digest.md",
			substrings: []string{
				"OPE-267` / `BIG-PAR-078",
				"## Tenant-Scoped Trigger Surface",
				"Pause the tenant rollout segment",
				"manual, evidence-backed operator action",
				"no tenant-scoped automated rollback trigger",
			},
		},
		{
			path: "docs/migration.md",
			substrings: []string{
				"tenant-scoped trigger surface",
				"rollback remains operator-driven",
			},
		},
		{
			path: "docs/reports/migration-plan-review-notes.md",
			substrings: []string{
				"tenant-scoped trigger surface",
				"OPE-267` / `BIG-PAR-078",
			},
		},
		{
			path: "docs/reports/migration-readiness-report.md",
			substrings: []string{
				"current trigger surface and manual rollback guardrails",
				"OPE-267` / `BIG-PAR-078",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"rollback safeguard trigger surface",
				"OPE-267` / `BIG-PAR-078",
			},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{
				"rollback safeguard trigger surfaces",
				"OPE-267` / `BIG-PAR-078",
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

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func readRepoFile(t *testing.T, root string, relative string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(contents)
}
