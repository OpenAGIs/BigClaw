package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO112ResidualTestContractSweepMManifestMatchesRetiredTests(t *testing.T) {
	replacements := migration.ResidualTestContractSweepMReplacements()
	if len(replacements) != 3 {
		t.Fatalf("expected 3 residual test replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"tests/test_design_system.py": {
			replacementKind: "go-design-system-contract",
			goReplacements: []string{
				"bigclaw-go/internal/designsystem/designsystem.go",
				"bigclaw-go/internal/product/console.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/designsystem/designsystem_test.go",
				"bigclaw-go/internal/product/console_test.go",
				"docs/issue-plan.md",
			},
			statusNeedle: "Go-native component-library audit",
		},
		"tests/test_dsl.py": {
			replacementKind: "go-workflow-definition-contract",
			goReplacements: []string{
				"bigclaw-go/internal/workflow/definition.go",
				"bigclaw-go/internal/workflow/closeout.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/workflow/definition_test.go",
				"docs/go-domain-intake-parity-matrix.md",
				"docs/issue-plan.md",
			},
			statusNeedle: "Go workflow definition parser",
		},
		"tests/test_evaluation.py": {
			replacementKind: "go-evaluation-replay-contract",
			goReplacements: []string{
				"bigclaw-go/internal/evaluation/evaluation.go",
				"bigclaw-go/internal/planning/planning.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/evaluation/evaluation_test.go",
				"bigclaw-go/internal/planning/planning_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go evaluation runner",
		},
	}

	for _, replacement := range replacements {
		want, ok := expected[replacement.RetiredPythonTest]
		if !ok {
			t.Fatalf("unexpected retired test in replacement registry: %+v", replacement)
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

func TestBIGGO112ResidualTestContractSweepMReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.ResidualTestContractSweepMReplacements() {
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

func TestBIGGO112ResidualTestContractSweepMLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-112-residual-test-contract-sweep-m.md")

	for _, needle := range []string{
		"BIG-GO-112",
		"Repository-wide Python file count: `0`.",
		"`tests/test_design_system.py`",
		"`tests/test_dsl.py`",
		"`tests/test_evaluation.py`",
		"`bigclaw-go/internal/migration/residual_test_contract_sweep_m.go`",
		"`bigclaw-go/internal/designsystem/designsystem.go`",
		"`bigclaw-go/internal/product/console.go`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/workflow/closeout.go`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/designsystem/designsystem_test.go`",
		"`bigclaw-go/internal/workflow/definition_test.go`",
		"`bigclaw-go/internal/evaluation/evaluation_test.go`",
		"`docs/go-domain-intake-parity-matrix.md`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO112ResidualTestContractSweepM",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
