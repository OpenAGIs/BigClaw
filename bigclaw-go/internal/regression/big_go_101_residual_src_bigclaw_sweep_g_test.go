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

	replacements := append([]migration.LegacyModuleReplacement{}, migration.LegacyReportingOpsModuleReplacements()...)
	replacements = append(replacements, migration.LegacyPolicyGovernanceModuleReplacements()...)
	replacements = append(replacements, migration.LegacyOperatorProductModuleReplacements()...)
	replacements = append(replacements, migration.LegacyBootstrapSyncModuleReplacements()...)
	if len(replacements) != 21 {
		t.Fatalf("expected 21 retired module replacements, got %d", len(replacements))
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
		"src/bigclaw/risk.py": {
			replacementKind: "go-risk-policy-surface",
			goReplacements: []string{
				"bigclaw-go/internal/risk/risk.go",
				"bigclaw-go/internal/risk/assessment.go",
				"bigclaw-go/internal/policy/policy.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/risk/risk_test.go",
				"bigclaw-go/internal/risk/assessment_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go risk scorer",
		},
		"src/bigclaw/governance.py": {
			replacementKind: "go-governance-freeze",
			goReplacements: []string{
				"bigclaw-go/internal/governance/freeze.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/governance/freeze_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go scope-freeze backlog board",
		},
		"src/bigclaw/execution_contract.py": {
			replacementKind: "go-execution-contract",
			goReplacements: []string{
				"bigclaw-go/internal/contract/execution.go",
				"bigclaw-go/internal/api/policy_runtime.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/contract/execution_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go execution contract",
		},
		"src/bigclaw/audit_events.py": {
			replacementKind: "go-audit-spec-surface",
			goReplacements: []string{
				"bigclaw-go/internal/observability/audit.go",
				"bigclaw-go/internal/observability/audit_spec.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/observability/audit_test.go",
				"bigclaw-go/internal/observability/audit_spec_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go audit event registry",
		},
		"src/bigclaw/issue_archive.py": {
			replacementKind: "go-issue-archive-surface",
			goReplacements: []string{
				"bigclaw-go/internal/issuearchive/archive.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/issuearchive/archive_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go issue priority archive",
		},
		"src/bigclaw/run_detail.py": {
			replacementKind: "go-run-detail-surface",
			goReplacements: []string{
				"bigclaw-go/internal/observability/task_run.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/observability/task_run_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go task-run detail",
		},
		"src/bigclaw/dashboard_run_contract.py": {
			replacementKind: "go-dashboard-contract",
			goReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/product/dashboard_run_contract_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go dashboard and run-detail contract",
		},
		"src/bigclaw/saved_views.py": {
			replacementKind: "go-saved-views-catalog",
			goReplacements: []string{
				"bigclaw-go/internal/product/saved_views.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/product/saved_views_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go saved-view catalog",
		},
		"src/bigclaw/console_ia.py": {
			replacementKind: "go-console-ia-surface",
			goReplacements: []string{
				"bigclaw-go/internal/consoleia/consoleia.go",
				"bigclaw-go/internal/product/console.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/consoleia/consoleia_test.go",
				"bigclaw-go/internal/product/console_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go console interaction architecture",
		},
		"src/bigclaw/design_system.py": {
			replacementKind: "go-design-system-surface",
			goReplacements: []string{
				"bigclaw-go/internal/designsystem/designsystem.go",
				"bigclaw-go/internal/product/console.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/designsystem/designsystem_test.go",
				"bigclaw-go/internal/product/console_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go design token library",
		},
		"src/bigclaw/github_sync.py": {
			replacementKind: "go-github-sync",
			goReplacements: []string{
				"bigclaw-go/internal/githubsync/sync.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/githubsync/sync_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go GitHub sync install",
		},
		"src/bigclaw/workspace_bootstrap.py": {
			replacementKind: "go-workspace-bootstrap",
			goReplacements: []string{
				"bigclaw-go/internal/bootstrap/bootstrap.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/bootstrap/bootstrap_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go bootstrap engine",
		},
		"src/bigclaw/workspace_bootstrap_cli.py": {
			replacementKind: "go-bootstrap-cli",
			goReplacements: []string{
				"bigclaw-go/internal/bootstrap/bootstrap.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/bootstrap/bootstrap_test.go",
				"bigclaw-go/cmd/bigclawctl/main_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go bootstrap engine",
		},
		"src/bigclaw/workspace_bootstrap_validation.py": {
			replacementKind: "go-bootstrap-validation",
			goReplacements: []string{
				"bigclaw-go/internal/bootstrap/bootstrap.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/bootstrap/bootstrap_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go bootstrap validation",
		},
		"src/bigclaw/parallel_refill.py": {
			replacementKind: "go-refill-queue",
			goReplacements: []string{
				"bigclaw-go/internal/refill/queue.go",
				"bigclaw-go/internal/refill/local_store.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/refill/queue_test.go",
				"bigclaw-go/internal/refill/local_store_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go refill queue",
		},
		"src/bigclaw/service.py": {
			replacementKind: "go-mainline-service",
			goReplacements: []string{
				"bigclaw-go/cmd/bigclawd/main.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			evidencePaths: []string{
				"bigclaw-go/cmd/bigclawd/main_test.go",
				"bigclaw-go/cmd/bigclawctl/main_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go bigclawd daemon",
		},
		"src/bigclaw/__main__.py": {
			replacementKind: "go-mainline-entrypoint",
			goReplacements: []string{
				"bigclaw-go/cmd/bigclawd/main.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			evidencePaths: []string{
				"bigclaw-go/cmd/bigclawd/main_test.go",
				"bigclaw-go/cmd/bigclawctl/main_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go daemon and operator CLIs",
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

	replacements := append([]migration.LegacyModuleReplacement{}, migration.LegacyReportingOpsModuleReplacements()...)
	replacements = append(replacements, migration.LegacyPolicyGovernanceModuleReplacements()...)
	replacements = append(replacements, migration.LegacyOperatorProductModuleReplacements()...)
	replacements = append(replacements, migration.LegacyBootstrapSyncModuleReplacements()...)
	for _, replacement := range replacements {
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
		"`src/bigclaw/risk.py`",
		"`src/bigclaw/governance.py`",
		"`src/bigclaw/execution_contract.py`",
		"`src/bigclaw/audit_events.py`",
		"`src/bigclaw/issue_archive.py`",
		"`src/bigclaw/run_detail.py`",
		"`src/bigclaw/dashboard_run_contract.py`",
		"`src/bigclaw/saved_views.py`",
		"`src/bigclaw/console_ia.py`",
		"`src/bigclaw/design_system.py`",
		"`src/bigclaw/github_sync.py`",
		"`src/bigclaw/workspace_bootstrap.py`",
		"`src/bigclaw/workspace_bootstrap_cli.py`",
		"`src/bigclaw/workspace_bootstrap_validation.py`",
		"`src/bigclaw/parallel_refill.py`",
		"`src/bigclaw/service.py`",
		"`src/bigclaw/__main__.py`",
		"`bigclaw-go/internal/migration/legacy_reporting_ops_modules.go`",
		"`bigclaw-go/internal/migration/legacy_policy_governance_modules.go`",
		"`bigclaw-go/internal/migration/legacy_operator_product_modules.go`",
		"`bigclaw-go/internal/migration/legacy_bootstrap_sync_modules.go`",
		"`bigclaw-go/internal/observability/recorder.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/reportstudio/reportstudio.go`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/contract/execution.go`",
		"`bigclaw-go/internal/control/controller.go`",
		"`bigclaw-go/internal/risk/risk.go`",
		"`bigclaw-go/internal/risk/assessment.go`",
		"`bigclaw-go/internal/policy/policy.go`",
		"`bigclaw-go/internal/governance/freeze.go`",
		"`bigclaw-go/internal/api/policy_runtime.go`",
		"`bigclaw-go/internal/observability/audit.go`",
		"`bigclaw-go/internal/observability/audit_spec.go`",
		"`bigclaw-go/internal/issuearchive/archive.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/product/saved_views.go`",
		"`bigclaw-go/internal/consoleia/consoleia.go`",
		"`bigclaw-go/internal/designsystem/designsystem.go`",
		"`bigclaw-go/internal/product/console.go`",
		"`bigclaw-go/internal/githubsync/sync.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`bigclaw-go/internal/refill/local_store.go`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
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
