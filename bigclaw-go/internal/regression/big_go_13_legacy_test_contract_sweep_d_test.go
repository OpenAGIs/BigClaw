package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO13LegacyTestContractSweepDManifestMatchesDeferredLegacyTests(t *testing.T) {
	replacements := migration.LegacyTestContractSweepDReplacements()
	if len(replacements) != 4 {
		t.Fatalf("expected 4 legacy test replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"tests/test_design_system.py": {
			replacementKind: "go-design-system-surface",
			goReplacements: []string{
				"bigclaw-go/internal/designsystem/designsystem.go",
				"bigclaw-go/internal/designsystem/designsystem_test.go",
				"bigclaw-go/internal/api/expansion_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/product/console_test.go",
				"reports/OPE-92-validation.md",
			},
			statusNeedle: "Go-owned designsystem",
		},
		"tests/test_dsl.py": {
			replacementKind: "go-workflow-definition-surface",
			goReplacements: []string{
				"bigclaw-go/internal/workflow/definition.go",
				"bigclaw-go/internal/workflow/definition_test.go",
				"bigclaw-go/internal/api/expansion.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/api/expansion_test.go",
				"bigclaw-go/cmd/bigclawctl/migration_commands.go",
			},
			statusNeedle: "Go workflow-definition",
		},
		"tests/test_evaluation.py": {
			replacementKind: "go-evaluation-surface",
			goReplacements: []string{
				"bigclaw-go/internal/evaluation/evaluation.go",
				"bigclaw-go/internal/evaluation/evaluation_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/planning/planning.go",
				"bigclaw-go/internal/planning/planning_test.go",
			},
			statusNeedle: "Go evaluation",
		},
		"tests/test_parallel_validation_bundle.py": {
			replacementKind: "go-validation-bundle-continuation-surface",
			goReplacements: []string{
				"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
				"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
				"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
				"bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json",
				"bigclaw-go/docs/reports/shared-queue-companion-summary.json",
			},
			statusNeedle: "Go automation bundle command",
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

func TestBIGGO13LegacyTestContractSweepDReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyTestContractSweepDReplacements() {
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

func TestBIGGO13LegacyTestContractSweepDLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md")

	for _, needle := range []string{
		"BIG-GO-13",
		"`reports/BIG-GO-948-validation.md`",
		"`tests/test_design_system.py`",
		"`tests/test_dsl.py`",
		"`tests/test_evaluation.py`",
		"`tests/test_parallel_validation_bundle.py`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`",
		"`bigclaw-go/internal/designsystem/designsystem.go`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO13LegacyTestContractSweepD",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
