package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1585RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1585E2EBucketStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	e2eDir := filepath.Join(rootRepo, "bigclaw-go", "scripts", "e2e")

	info, err := os.Stat(e2eDir)
	if err != nil {
		t.Fatalf("expected e2e bucket to exist: %s (%v)", e2eDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected e2e bucket path to be a directory: %s", e2eDir)
	}

	pythonFiles := collectPythonFiles(t, e2eDir)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected e2e bucket to remain Python-free: %v", pythonFiles)
	}
}

func TestBIGGO1585ActiveE2EReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"bigclaw-go/scripts/e2e/broker_bootstrap_summary.go",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected active E2E replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1585LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1585-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1585",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/scripts/e2e`: `0` Python files",
		"Directory present on disk: `yes`.",
		"`bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh`",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/scripts/e2e -type f -name '*.py' 2>/dev/null | sort`",
		"`test -d bigclaw-go/scripts/e2e`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1585",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
