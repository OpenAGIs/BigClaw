package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBIGGO1RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO1GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "reports/BIG-GO-1-validation.md")

	for _, needle := range []string{
		"BIG-GO-1",
		"Repository-wide physical `.py` files: `0`",
		"`src/bigclaw/*.py`: `none`",
		"`tests/*.py`: `none`",
		"`scripts/*.py`: `none`",
		"`bigclaw-go/scripts/*.py`: `none`",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func regressionRepoRoot(t *testing.T) string {
	t.Helper()

	rootRepo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve regression repo root: %v", err)
	}
	return filepath.Clean(rootRepo)
}

func collectPythonFiles(t *testing.T, root string) []string {
	t.Helper()

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("stat %s: %v", root, err)
	}
	if !info.IsDir() {
		if strings.HasSuffix(info.Name(), ".py") {
			return []string{filepath.Clean(root)}
		}
		return nil
	}

	var pythonFiles []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".py") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			pythonFiles = append(pythonFiles, filepath.ToSlash(rel))
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(pythonFiles)
	return pythonFiles
}

func readRepoFile(t *testing.T, rootRepo, relativePath string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}
	return string(data)
}
