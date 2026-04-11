package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO221RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO221SrcBigclawTranche17PathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPythonPaths := []string{
		"src/bigclaw/__init__.py",
		"src/bigclaw/__main__.py",
		"src/bigclaw/audit_events.py",
		"src/bigclaw/collaboration.py",
		"src/bigclaw/console_ia.py",
		"src/bigclaw/design_system.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/runtime.py",
	}

	for _, relativePath := range retiredPythonPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired tranche-17 Python path to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO221GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/observability/audit_spec.go",
		"bigclaw-go/internal/collaboration/thread.go",
		"bigclaw-go/internal/consoleia/consoleia.go",
		"bigclaw-go/internal/designsystem/designsystem.go",
		"bigclaw-go/internal/evaluation/evaluation.go",
		"bigclaw-go/internal/observability/task_run.go",
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO221LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-221-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-221",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"Explicit assigned Python asset list:",
		"`src/bigclaw/__init__.py`",
		"`src/bigclaw/__main__.py`",
		"`src/bigclaw/audit_events.py`",
		"`src/bigclaw/collaboration.py`",
		"`src/bigclaw/console_ia.py`",
		"`src/bigclaw/design_system.py`",
		"`src/bigclaw/evaluation.py`",
		"`src/bigclaw/run_detail.py`",
		"`src/bigclaw/runtime.py`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/internal/observability/audit.go`",
		"`bigclaw-go/internal/observability/audit_spec.go`",
		"`bigclaw-go/internal/collaboration/thread.go`",
		"`bigclaw-go/internal/consoleia/consoleia.go`",
		"`bigclaw-go/internal/designsystem/designsystem.go`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/internal/observability/task_run.go`",
		"`bigclaw-go/internal/worker/runtime.go`",
		"`bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`for path in src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/audit_events.py src/bigclaw/collaboration.py src/bigclaw/console_ia.py src/bigclaw/design_system.py src/bigclaw/evaluation.py src/bigclaw/run_detail.py src/bigclaw/runtime.py; do test ! -e \"$path\" && printf 'absent %s\\n' \"$path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO221(RepositoryHasNoPythonFiles|SrcBigclawTranche17PathsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche17$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
