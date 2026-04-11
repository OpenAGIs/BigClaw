package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO119RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO119PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO119HiddenAndAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	hiddenAndAuxiliaryDirs := []string{
		".github",
		".githooks",
		".symphony",
		"docs",
		"bigclaw-go/docs",
		"bigclaw-go/examples",
		"scripts/ops",
	}

	for _, relativeDir := range hiddenAndAuxiliaryDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected hidden or auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO119ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		".github/workflows/ci.yml",
		".githooks/post-commit",
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go replacement or native control path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO119LaneReportDocumentsPythonAssetSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-119-python-asset-sweep.md")

	requiredSubstrings := []string{
		"# BIG-GO-119 Python Asset Sweep",
		"Remaining physical Python asset inventory: `0` files.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"Hidden and lower-priority directories audited in this lane:",
		"`.github`: `0` Python files",
		"`.githooks`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`docs`: `0` Python files",
		"`bigclaw-go/docs`: `0` Python files",
		"`bigclaw-go/examples`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find .github .githooks .symphony docs bigclaw-go/docs bigclaw-go/examples scripts/ops -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO119(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|HiddenAndAuxiliaryDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(report, needle) {
			t.Fatalf("big-go-119 lane report missing substring %q", needle)
		}
	}
}
