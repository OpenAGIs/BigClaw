package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO23LegacyTestContractSweepCManifestMatchesDeferredLegacyTests(t *testing.T) {
	replacements := migration.LegacyTestContractSweepCReplacements()
	if len(replacements) != 2 {
		t.Fatalf("expected 2 legacy test replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"tests/test_issue_archive.py": {
			replacementKind: "go-issue-priority-archive-surface",
			goReplacements: []string{
				"bigclaw-go/internal/issuearchive/archive.go",
				"bigclaw-go/internal/issuearchive/archive_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/reports/big-go-1596-go-only-sweep-refill.md",
				"reports/BIG-GO-948-validation.md",
			},
			statusNeedle: "Go issue-priority archive",
		},
		"tests/test_pilot.py": {
			replacementKind: "go-pilot-readiness-surface",
			goReplacements: []string{
				"bigclaw-go/internal/pilot/report.go",
				"bigclaw-go/internal/pilot/report_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/reports/big-go-1597-python-asset-sweep.md",
				"reports/BIG-GO-948-validation.md",
			},
			statusNeedle: "Go pilot readiness",
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

func TestBIGGO23LegacyTestContractSweepCReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyTestContractSweepCReplacements() {
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

func TestBIGGO23LegacyTestContractSweepCLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-23-legacy-test-contract-sweep-c.md")

	for _, needle := range []string{
		"BIG-GO-23",
		"Repository-wide Python file count: `0`.",
		"`reports/BIG-GO-948-validation.md`",
		"`tests/test_issue_archive.py`",
		"`tests/test_pilot.py`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_c.go`",
		"`bigclaw-go/internal/issuearchive/archive.go`",
		"`bigclaw-go/internal/issuearchive/archive_test.go`",
		"`bigclaw-go/internal/pilot/report.go`",
		"`bigclaw-go/internal/pilot/report_test.go`",
		"`bigclaw-go/docs/reports/big-go-1596-go-only-sweep-refill.md`",
		"`bigclaw-go/docs/reports/big-go-1597-python-asset-sweep.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO23LegacyTestContractSweepC",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
