package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO101RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO101ResidualSrcBigClawSweepGStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacements := migration.LegacyReportingOpsModuleReplacements()
	if len(replacements) != 4 {
		t.Fatalf("expected 4 retired module replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"src/bigclaw/observability.py": {
			replacementKind: "go-observability-runtime",
			goReplacements: []string{
				"bigclaw-go/internal/observability/recorder.go",
				"bigclaw-go/internal/observability/task_run.go",
				"bigclaw-go/internal/observability/audit.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/observability/recorder_test.go",
				"bigclaw-go/internal/observability/task_run_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go observability runtime",
		},
		"src/bigclaw/reports.py": {
			replacementKind: "go-reporting-surface",
			goReplacements: []string{
				"bigclaw-go/internal/reporting/reporting.go",
				"bigclaw-go/internal/reportstudio/reportstudio.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/reporting/reporting_test.go",
				"bigclaw-go/internal/reportstudio/reportstudio_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go reporting builders",
		},
		"src/bigclaw/evaluation.py": {
			replacementKind: "go-evaluation-benchmark",
			goReplacements: []string{
				"bigclaw-go/internal/evaluation/evaluation.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/evaluation/evaluation_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go benchmark",
		},
		"src/bigclaw/operations.py": {
			replacementKind: "go-operations-control-plane",
			goReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
				"bigclaw-go/internal/contract/execution.go",
				"bigclaw-go/internal/control/controller.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/product/dashboard_run_contract_test.go",
				"bigclaw-go/internal/contract/execution_test.go",
				"bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md",
			},
			statusNeedle: "Go dashboard contract",
		},
	}

	for _, replacement := range replacements {
		want, ok := expected[replacement.RetiredPythonModule]
		if !ok {
			t.Fatalf("unexpected retired module in sweep-g registry: %+v", replacement)
		}
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(replacement.RetiredPythonModule))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python module to stay absent: %s", replacement.RetiredPythonModule)
		}
		if replacement.ReplacementKind != want.replacementKind {
			t.Fatalf("replacement kind for %s = %q, want %q", replacement.RetiredPythonModule, replacement.ReplacementKind, want.replacementKind)
		}
		assertExactStringSlice(t, replacement.GoReplacements, want.goReplacements, replacement.RetiredPythonModule+" go replacements")
		assertExactStringSlice(t, replacement.EvidencePaths, want.evidencePaths, replacement.RetiredPythonModule+" evidence paths")
		if !strings.Contains(replacement.Status, want.statusNeedle) {
			t.Fatalf("replacement status for %s missing %q: %q", replacement.RetiredPythonModule, want.statusNeedle, replacement.Status)
		}
	}
}

func TestBIGGO101GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyReportingOpsModuleReplacements() {
		for _, relativePath := range replacement.GoReplacements {
			if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
				t.Fatalf("expected Go replacement path to exist for %s: %s (%v)", replacement.RetiredPythonModule, relativePath, err)
			}
		}
		for _, relativePath := range replacement.EvidencePaths {
			if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
				t.Fatalf("expected evidence path to exist for %s: %s (%v)", replacement.RetiredPythonModule, relativePath, err)
			}
		}
	}
}

func TestBIGGO101LaneReportCapturesReplacementEvidence(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-101-residual-src-bigclaw-python-sweep-g.md")

	for _, needle := range []string{
		"BIG-GO-101",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `src/bigclaw` sweep-G physical Python file count before lane changes: `0`",
		"Focused `src/bigclaw` sweep-G physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused sweep-G ledger: `[]`",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`src/bigclaw/observability.py`",
		"`src/bigclaw/reports.py`",
		"`src/bigclaw/evaluation.py`",
		"`src/bigclaw/operations.py`",
		"`bigclaw-go/internal/migration/legacy_reporting_ops_modules.go`",
		"`bigclaw-go/internal/observability/recorder.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/reportstudio/reportstudio.go`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/contract/execution.go`",
		"`bigclaw-go/internal/control/controller.go`",
		"`bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO101",
		"`cd bigclaw-go && go test -count=1 ./internal/migration`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
