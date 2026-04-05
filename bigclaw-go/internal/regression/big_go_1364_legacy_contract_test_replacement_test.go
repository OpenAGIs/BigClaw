package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO1364LegacyContractTestReplacementManifestMatchesRetiredTests(t *testing.T) {
	replacements := migration.LegacyContractTestReplacementsSweepA()
	if len(replacements) != 5 {
		t.Fatalf("expected 5 legacy contract test replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind    string
		nativeReplacements []string
		evidencePaths      []string
		statusNeedle       string
	}{
		"tests/test_execution_contract.py": {
			replacementKind: "go-contract-owner",
			nativeReplacements: []string{
				"bigclaw-go/internal/contract/execution.go",
				"bigclaw-go/internal/contract/execution_test.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"reports/OPE-131-validation.md",
			},
			statusNeedle: "Go execution contract owner",
		},
		"tests/test_dashboard_run_contract.py": {
			replacementKind: "go-contract-owner",
			nativeReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
				"bigclaw-go/internal/product/dashboard_run_contract_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/api/expansion_test.go",
				"reports/OPE-129-validation.md",
			},
			statusNeedle: "Go product contract package",
		},
		"tests/test_cross_process_coordination_surface.py": {
			replacementKind: "go-api-surface",
			nativeReplacements: []string{
				"bigclaw-go/internal/api/coordination_surface.go",
				"bigclaw-go/internal/regression/coordination_contract_surface_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/e2e-validation.md",
				"bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json",
			},
			statusNeedle: "Go API surface loader",
		},
		"tests/test_followup_digests.py": {
			replacementKind: "repo-native-report-guard",
			nativeReplacements: []string{
				"bigclaw-go/docs/reports/parallel-follow-up-index.md",
				"bigclaw-go/internal/regression/followup_index_docs_test.go",
			},
			evidencePaths: []string{
				"docs/parallel-refill-queue.md",
				"reports/OPE-270-271-validation.md",
			},
			statusNeedle: "repo-native parallel follow-up index",
		},
		"tests/test_parallel_refill.py": {
			replacementKind: "go-queue-and-native-docs",
			nativeReplacements: []string{
				"bigclaw-go/internal/refill/queue_test.go",
				"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
			},
			evidencePaths: []string{
				"docs/parallel-refill-queue.json",
				"reports/OPE-270-271-validation.md",
			},
			statusNeedle: "Go refill queue coverage",
		},
	}

	for _, replacement := range replacements {
		want, ok := expected[replacement.RetiredPythonTest]
		if !ok {
			t.Fatalf("unexpected retired Python test in replacement registry: %+v", replacement)
		}
		if replacement.ReplacementKind != want.replacementKind {
			t.Fatalf("replacement kind for %s = %q, want %q", replacement.RetiredPythonTest, replacement.ReplacementKind, want.replacementKind)
		}
		assertExactStringSlice(t, replacement.NativeReplacements, want.nativeReplacements, replacement.RetiredPythonTest+" native replacements")
		assertExactStringSlice(t, replacement.EvidencePaths, want.evidencePaths, replacement.RetiredPythonTest+" evidence paths")
		if !strings.Contains(replacement.Status, want.statusNeedle) {
			t.Fatalf("replacement status for %s missing %q: %q", replacement.RetiredPythonTest, want.statusNeedle, replacement.Status)
		}
	}
}

func TestBIGGO1364LegacyContractTestReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyContractTestReplacementsSweepA() {
		if _, err := os.Stat(filepath.Join(rootRepo, replacement.RetiredPythonTest)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test to remain absent: %s", replacement.RetiredPythonTest)
		}
		for _, relativePath := range replacement.NativeReplacements {
			if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
				t.Fatalf("expected native replacement path to exist for %s: %s (%v)", replacement.RetiredPythonTest, relativePath, err)
			}
		}
		for _, relativePath := range replacement.EvidencePaths {
			if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
				t.Fatalf("expected evidence path to exist for %s: %s (%v)", replacement.RetiredPythonTest, relativePath, err)
			}
		}
	}
}

func TestBIGGO1364LegacyContractTestReplacementLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1364-legacy-contract-test-replacement.md")

	for _, needle := range []string{
		"BIG-GO-1364",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/internal/migration/legacy_contract_tests.go`",
		"`tests/test_execution_contract.py`",
		"`tests/test_dashboard_run_contract.py`",
		"`tests/test_cross_process_coordination_surface.py`",
		"`tests/test_followup_digests.py`",
		"`tests/test_parallel_refill.py`",
		"`bigclaw-go/internal/contract/execution.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/api/coordination_surface.go`",
		"`bigclaw-go/docs/reports/parallel-follow-up-index.md`",
		"`bigclaw-go/internal/refill/queue_test.go`",
		"`find . -name '*.py' | wc -l`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1364LegacyContractTestReplacement",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
