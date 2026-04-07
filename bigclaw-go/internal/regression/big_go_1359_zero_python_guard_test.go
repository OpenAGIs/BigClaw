package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1359RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1359PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1359RaySmokeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/docs/e2e-validation.md",
		"bigclaw-go/cmd/bigclawctl/main.go",
	}
	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected native replacement path to exist: %s (%v)", relativePath, err)
		}
	}

	raySmoke := readRepoFile(t, rootRepo, "bigclaw-go/scripts/e2e/ray_smoke.sh")
	for _, needle := range []string{
		"BIGCLAW_RAY_SMOKE_ENTRYPOINT",
		"sh -c 'echo hello from ray'",
	} {
		if !strings.Contains(raySmoke, needle) {
			t.Fatalf("ray smoke script missing shell-native replacement evidence %q", needle)
		}
	}
	if strings.Contains(raySmoke, "python -c") {
		t.Fatal("ray smoke script should not fall back to inline Python")
	}

	validationDoc := readRepoFile(t, rootRepo, "bigclaw-go/docs/e2e-validation.md")
	for _, needle := range []string{
		"## Ray smoke test",
		"export BIGCLAW_RAY_SMOKE_ENTRYPOINT=\"sh -c 'echo 123'\"",
	} {
		if !strings.Contains(validationDoc, needle) {
			t.Fatalf("validation doc missing shell-native Ray smoke guidance %q", needle)
		}
	}
	if strings.Contains(validationDoc, "- `python3`") {
		t.Fatal("validation doc should not require python3 for the active smoke path")
	}
}

func TestBIGGO1359LaneReportCapturesNativeReplacement(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1359-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1359",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh` now defaults `BIGCLAW_RAY_SMOKE_ENTRYPOINT` to `sh -c 'echo hello from ray'`",
		"`bigclaw-go/docs/e2e-validation.md` no longer lists `python3` as a prerequisite for the active smoke path",
		"`bigclaw-go/docs/e2e-validation.md` now documents `export BIGCLAW_RAY_SMOKE_ENTRYPOINT=\"sh -c 'echo 123'\"` as the override example",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1359",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
