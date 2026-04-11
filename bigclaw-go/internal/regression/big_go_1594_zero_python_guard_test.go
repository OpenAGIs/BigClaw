package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1594RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1594AssignedPythonAssetsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/collaboration.py",
		"src/bigclaw/github_sync.py",
		"src/bigclaw/pilot.py",
		"src/bigclaw/repo_triage.py",
		"src/bigclaw/validation_policy.py",
		"tests/test_cost_control.py",
		"tests/test_github_sync.py",
		"tests/test_orchestration.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected assigned Python asset to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1594GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/collaboration/thread.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/pilot/report.go",
		"bigclaw-go/internal/repo/triage.go",
		"bigclaw-go/internal/policy/validation.go",
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/githubsync/sync_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"scripts/ops/bigclawctl",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1594LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1594-go-only-sweep-refill.md")

	for _, needle := range []string{
		"BIG-GO-1594",
		"Repository-wide Python file count before lane changes: `0`.",
		"Repository-wide Python file count after lane changes: `0`.",
		"Explicit remaining Python asset list: none.",
		"`src/bigclaw/collaboration.py` -> `bigclaw-go/internal/collaboration/thread.go`",
		"`src/bigclaw/github_sync.py` -> `bigclaw-go/internal/githubsync/sync.go`",
		"`src/bigclaw/pilot.py` -> `bigclaw-go/internal/pilot/report.go`",
		"`src/bigclaw/repo_triage.py` -> `bigclaw-go/internal/repo/triage.go`",
		"`src/bigclaw/validation_policy.py` -> `bigclaw-go/internal/policy/validation.go`",
		"`tests/test_cost_control.py` -> `bigclaw-go/internal/costcontrol/controller_test.go`",
		"`tests/test_github_sync.py` -> `bigclaw-go/internal/githubsync/sync_test.go`",
		"`tests/test_orchestration.py` -> `bigclaw-go/internal/workflow/orchestration_test.go`",
		"`scripts/ops/bigclawctl`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`for path in src/bigclaw/collaboration.py src/bigclaw/github_sync.py src/bigclaw/pilot.py src/bigclaw/repo_triage.py src/bigclaw/validation_policy.py tests/test_cost_control.py tests/test_github_sync.py tests/test_orchestration.py; do test ! -e \"$path\" && printf 'absent %s\\n' \"$path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1594(RepositoryHasNoPythonFiles|AssignedPythonAssetsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
		"Residual risk: this checkout already started with zero physical Python files, so BIG-GO-1594 hardens that baseline rather than lowering the numeric file count further.",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
