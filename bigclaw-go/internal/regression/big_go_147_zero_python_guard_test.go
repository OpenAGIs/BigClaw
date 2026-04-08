package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO147RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO147RetiredContractAndGovernanceTrancheStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/models.py",
		"src/bigclaw/connectors.py",
		"src/bigclaw/dsl.py",
		"src/bigclaw/risk.py",
		"src/bigclaw/governance.py",
		"src/bigclaw/execution_contract.py",
		"src/bigclaw/audit_events.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired contract/governance Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO147GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/domain/task.go",
		"bigclaw-go/internal/intake/connector.go",
		"bigclaw-go/internal/intake/types.go",
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/risk/assessment.go",
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/contract/execution.go",
		"bigclaw-go/internal/observability/audit_spec.go",
		"docs/go-domain-intake-parity-matrix.md",
		"docs/go-mainline-cutover-issue-pack.md",
		"docs/go-mainline-cutover-handoff.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO147LaneReportCapturesReplacementEvidence(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-147-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-147",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused retired contract/governance tranche physical Python file count before lane changes: `0`",
		"Focused retired contract/governance tranche physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused tranche ledger: `[]`",
		"`src/bigclaw/models.py`",
		"`src/bigclaw/connectors.py`",
		"`src/bigclaw/dsl.py`",
		"`src/bigclaw/risk.py`",
		"`src/bigclaw/governance.py`",
		"`src/bigclaw/execution_contract.py`",
		"`src/bigclaw/audit_events.py`",
		"`bigclaw-go/internal/domain/task.go`",
		"`bigclaw-go/internal/intake/connector.go`",
		"`bigclaw-go/internal/intake/types.go`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/risk/assessment.go`",
		"`bigclaw-go/internal/governance/freeze.go`",
		"`bigclaw-go/internal/contract/execution.go`",
		"`bigclaw-go/internal/observability/audit_spec.go`",
		"`docs/go-domain-intake-parity-matrix.md`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw -type f \\( -name 'models.py' -o -name 'connectors.py' -o -name 'dsl.py' -o -name 'risk.py' -o -name 'governance.py' -o -name 'execution_contract.py' -o -name 'audit_events.py' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO147(RepositoryHasNoPythonFiles|RetiredContractAndGovernanceTrancheStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
