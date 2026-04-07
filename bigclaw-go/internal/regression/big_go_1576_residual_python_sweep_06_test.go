package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1576ResidualPythonCandidatesStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	candidates := []string{
		"src/bigclaw/console_ia.py",
		"src/bigclaw/issue_archive.py",
		"src/bigclaw/queue.py",
		"src/bigclaw/risk.py",
		"src/bigclaw/workspace_bootstrap.py",
		"tests/test_dashboard_run_contract.py",
		"tests/test_issue_archive.py",
		"tests/test_parallel_validation_bundle.py",
		"tests/test_repo_rollout.py",
		"tests/test_shadow_matrix_corpus.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
	}

	for _, relativePath := range candidates {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected BIG-GO-1576 Python candidate to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1576GoReplacementsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacements := []string{
		"bigclaw-go/internal/consoleia/consoleia.go",
		"bigclaw-go/internal/issuearchive/archive.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/risk/risk.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"bigclaw-go/internal/api/expansion_test.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go",
		"bigclaw-go/internal/product/clawhost_rollout_test.go",
		"bigclaw-go/internal/pilot/rollout_test.go",
		"bigclaw-go/internal/regression/production_corpus_surface_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"scripts/ops/bigclawctl",
		"bigclaw-go/docs/reports/big-go-1576-go-only-residual-python-sweep-06.md",
	}

	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected BIG-GO-1576 replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1576SweepReportCapturesLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1576-go-only-residual-python-sweep-06.md")

	required := []string{
		"BIG-GO-1576 Go-only residual Python sweep 06",
		"`src/bigclaw/console_ia.py`",
		"`src/bigclaw/issue_archive.py`",
		"`src/bigclaw/queue.py`",
		"`src/bigclaw/risk.py`",
		"`src/bigclaw/workspace_bootstrap.py`",
		"`tests/test_dashboard_run_contract.py`",
		"`tests/test_issue_archive.py`",
		"`tests/test_parallel_validation_bundle.py`",
		"`tests/test_repo_rollout.py`",
		"`tests/test_shadow_matrix_corpus.py`",
		"`scripts/ops/bigclaw_workspace_bootstrap.py`",
		"`bigclaw-go/scripts/e2e/export_validation_bundle.py`",
		"`bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`",
		"`bigclaw-go/internal/consoleia/consoleia.go`",
		"`bigclaw-go/internal/issuearchive/archive.go`",
		"`bigclaw-go/internal/queue/queue.go`",
		"`bigclaw-go/internal/risk/risk.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`",
		"No Python compatibility shim remains for this sweep set.",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"Historical reports and migration plans still mention some removed Python paths as prior-state",
	}
	for _, needle := range required {
		if !strings.Contains(report, needle) {
			t.Fatalf("BIG-GO-1576 sweep report missing substring %q", needle)
		}
	}
}
