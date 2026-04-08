package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO173LegacyTestContractSweepZManifestMatchesDeferredLegacyTests(t *testing.T) {
	replacements := migration.LegacyTestContractSweepZReplacements()
	if len(replacements) != 4 {
		t.Fatalf("expected 4 legacy test replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"tests/test_repo_collaboration.py": {
			replacementKind: "go-repo-collaboration-thread-surface",
			goReplacements: []string{
				"bigclaw-go/internal/collaboration/thread.go",
				"bigclaw-go/internal/collaboration/thread_test.go",
				"bigclaw-go/internal/repo/board.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/repo/repo_surfaces_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
				"docs/go-mainline-cutover-handoff.md",
			},
			statusNeedle: "Go-native collaboration",
		},
		"tests/test_repo_gateway.py": {
			replacementKind: "go-repo-gateway-normalization-surface",
			goReplacements: []string{
				"bigclaw-go/internal/repo/gateway.go",
				"bigclaw-go/internal/repo/repo_surfaces_test.go",
				"bigclaw-go/internal/repo/commits.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"docs/go-mainline-cutover-handoff.md",
			},
			statusNeedle: "Go repo gateway",
		},
		"tests/test_repo_governance.py": {
			replacementKind: "go-repo-governance-contract-surface",
			goReplacements: []string{
				"bigclaw-go/internal/repo/governance.go",
				"bigclaw-go/internal/repo/governance_test.go",
				"bigclaw-go/internal/repo/plane.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"docs/go-mainline-cutover-handoff.md",
			},
			statusNeedle: "Go permission contract",
		},
		"tests/test_repo_registry.py": {
			replacementKind: "go-repo-registry-routing-surface",
			goReplacements: []string{
				"bigclaw-go/internal/repo/registry.go",
				"bigclaw-go/internal/repo/repo_surfaces_test.go",
				"bigclaw-go/internal/repo/links.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"docs/go-mainline-cutover-handoff.md",
			},
			statusNeedle: "Go repo registry",
		},
	}

	for _, replacement := range replacements {
		want, ok := expected[replacement.RetiredPythonTest]
		if !ok {
			t.Fatalf("unexpected retired legacy test in replacement registry: %+v", replacement)
		}
		if replacement.ReplacementKind != want.replacementKind {
			t.Fatalf("replacement kind for %s = %q, want %q", replacement.RetiredPythonTest, replacement.ReplacementKind, want.replacementKind)
		}
		assertExactStringSlice(t, replacement.GoReplacements, want.goReplacements, replacement.RetiredPythonTest+" go replacements")
		assertExactStringSlice(t, replacement.EvidencePaths, want.evidencePaths, replacement.RetiredPythonTest+" evidence paths")
		if !strings.Contains(replacement.Status, want.statusNeedle) {
			t.Fatalf("replacement status for %s missing %q: %q", replacement.RetiredPythonTest, want.statusNeedle, replacement.Status)
		}
	}
}

func TestBIGGO173LegacyTestContractSweepZReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyTestContractSweepZReplacements() {
		if _, err := os.Stat(filepath.Join(rootRepo, replacement.RetiredPythonTest)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test to stay absent: %s", replacement.RetiredPythonTest)
		}
		for _, relativePath := range replacement.GoReplacements {
			if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
				t.Fatalf("expected Go replacement path to exist for %s: %s (%v)", replacement.RetiredPythonTest, relativePath, err)
			}
		}
		for _, relativePath := range replacement.EvidencePaths {
			if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
				t.Fatalf("expected evidence path to exist for %s: %s (%v)", replacement.RetiredPythonTest, relativePath, err)
			}
		}
	}
}

func TestBIGGO173LegacyTestContractSweepZLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-173-legacy-test-contract-sweep-z.md")

	for _, needle := range []string{
		"BIG-GO-173",
		"Repository-wide Python file count: `0`.",
		"`tests/test_repo_collaboration.py`",
		"`tests/test_repo_gateway.py`",
		"`tests/test_repo_governance.py`",
		"`tests/test_repo_registry.py`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_z.go`",
		"`bigclaw-go/internal/collaboration/thread.go`",
		"`bigclaw-go/internal/repo/gateway.go`",
		"`bigclaw-go/internal/repo/governance.go`",
		"`bigclaw-go/internal/repo/registry.go`",
		"`bigclaw-go/internal/repo/repo_surfaces_test.go`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO173LegacyTestContractSweepZ",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
