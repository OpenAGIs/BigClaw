package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO163LegacyTestContractSweepXManifestMatchesDeferredLegacyTests(t *testing.T) {
	replacements := migration.LegacyTestContractSweepXReplacements()
	if len(replacements) != 4 {
		t.Fatalf("expected 4 legacy test replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"tests/test_audit_events.py": {
			replacementKind: "go-audit-event-spec-surface",
			goReplacements: []string{
				"bigclaw-go/internal/observability/audit_spec.go",
				"bigclaw-go/internal/observability/audit.go",
				"bigclaw-go/internal/observability/audit_spec_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/observability/audit_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
				"reports/OPE-134-validation.md",
			},
			statusNeedle: "Go audit-event",
		},
		"tests/test_connectors.py": {
			replacementKind: "go-intake-connector-surface",
			goReplacements: []string{
				"bigclaw-go/internal/intake/connector.go",
				"bigclaw-go/internal/intake/types.go",
				"bigclaw-go/internal/api/v2.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/intake/connector_test.go",
				"docs/go-domain-intake-parity-matrix.md",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go intake connector",
		},
		"tests/test_console_ia.py": {
			replacementKind: "go-console-ia-surface",
			goReplacements: []string{
				"bigclaw-go/internal/consoleia/consoleia.go",
				"bigclaw-go/internal/product/console.go",
				"bigclaw-go/internal/designsystem/designsystem.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/consoleia/consoleia_test.go",
				"bigclaw-go/internal/product/console_test.go",
				"reports/OPE-127-validation.md",
			},
			statusNeedle: "Go console IA",
		},
		"tests/test_dashboard_run_contract.py": {
			replacementKind: "go-dashboard-run-contract-surface",
			goReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
				"bigclaw-go/internal/contract/execution.go",
				"bigclaw-go/internal/api/server.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/product/dashboard_run_contract_test.go",
				"bigclaw-go/internal/contract/execution_test.go",
				"reports/OPE-129-validation.md",
			},
			statusNeedle: "Go dashboard/run contract",
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

func TestBIGGO163LegacyTestContractSweepXReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyTestContractSweepXReplacements() {
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

func TestBIGGO163LegacyTestContractSweepXLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-163-legacy-test-contract-sweep-x.md")

	for _, needle := range []string{
		"BIG-GO-163",
		"Repository-wide Python file count: `0`.",
		"`tests/test_audit_events.py`",
		"`tests/test_connectors.py`",
		"`tests/test_console_ia.py`",
		"`tests/test_dashboard_run_contract.py`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_x.go`",
		"`bigclaw-go/internal/observability/audit_spec.go`",
		"`docs/go-domain-intake-parity-matrix.md`",
		"`bigclaw-go/internal/consoleia/consoleia.go`",
		"`reports/OPE-127-validation.md`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`reports/OPE-129-validation.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO163LegacyTestContractSweepX",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
