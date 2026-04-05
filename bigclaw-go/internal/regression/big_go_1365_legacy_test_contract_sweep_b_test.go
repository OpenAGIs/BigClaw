package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO1365LegacyTestContractSweepBManifestMatchesDeferredLegacyTests(t *testing.T) {
	replacements := migration.LegacyTestContractSweepBReplacements()
	if len(replacements) != 3 {
		t.Fatalf("expected 3 legacy test replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"tests/test_control_center.py": {
			replacementKind: "go-control-plane-surface",
			goReplacements: []string{
				"bigclaw-go/internal/control/controller.go",
				"bigclaw-go/internal/api/server.go",
				"bigclaw-go/internal/api/v2.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/control/controller_test.go",
				"bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md",
			},
			statusNeedle: "Go control plane",
		},
		"tests/test_operations.py": {
			replacementKind: "go-operations-contract-split",
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
			statusNeedle: "Go-owned dashboard",
		},
		"tests/test_ui_review.py": {
			replacementKind: "go-review-pack-surface",
			goReplacements: []string{
				"bigclaw-go/internal/uireview/uireview.go",
				"bigclaw-go/internal/uireview/builder.go",
				"bigclaw-go/internal/uireview/render.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/uireview/uireview_test.go",
				"docs/issue-plan.md",
				"reports/OPE-128-validation.md",
			},
			statusNeedle: "Go-native review-pack",
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

func TestBIGGO1365LegacyTestContractSweepBReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyTestContractSweepBReplacements() {
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

func TestBIGGO1365LegacyTestContractSweepBLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md")

	for _, needle := range []string{
		"BIG-GO-1365",
		"Repository-wide Python file count: `0`.",
		"`reports/BIG-GO-948-validation.md`",
		"`tests/test_control_center.py`",
		"`tests/test_operations.py`",
		"`tests/test_ui_review.py`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`",
		"`bigclaw-go/internal/control/controller.go`",
		"`bigclaw-go/internal/contract/execution.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/uireview/uireview.go`",
		"`bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md`",
		"`reports/OPE-128-validation.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1365LegacyTestContractSweepB",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
