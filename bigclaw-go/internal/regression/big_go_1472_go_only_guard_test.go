package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1472RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1472RepositoryHasNoPythonBootstrapDependencyFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	bannedFiles := []string{
		".python-version",
		"conftest.py",
		"Pipfile",
		"poetry.lock",
		"pyproject.toml",
		"pytest.ini",
		"tox.ini",
	}
	bannedPrefixes := []string{
		"requirements",
	}

	var found []string
	err := filepath.WalkDir(rootRepo, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		base := d.Name()
		for _, banned := range bannedFiles {
			if base == banned {
				relative, relErr := filepath.Rel(rootRepo, path)
				if relErr != nil {
					return relErr
				}
				found = append(found, filepath.ToSlash(relative))
				return nil
			}
		}
		for _, prefix := range bannedPrefixes {
			if strings.HasPrefix(base, prefix) && strings.HasSuffix(base, ".txt") {
				relative, relErr := filepath.Rel(rootRepo, path)
				if relErr != nil {
					return relErr
				}
				found = append(found, filepath.ToSlash(relative))
				return nil
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo root: %v", err)
	}
	if len(found) != 0 {
		t.Fatalf("expected Python bootstrap/test dependency files to remain absent, found %v", found)
	}
}

func TestBIGGO1472BootstrapTemplateStaysGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	template := readRepoFile(t, rootRepo, "docs/symphony-repo-bootstrap-template.md")

	for _, forbidden := range []string{
		"workspace_bootstrap.py",
		"workspace_bootstrap_cli.py",
	} {
		if strings.Contains(template, forbidden) {
			t.Fatalf("bootstrap template should not reference retired Python asset %q", forbidden)
		}
	}

	for _, required := range []string{
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/internal/bootstrap/*`",
		"`conftest.py`",
		"`pytest.ini`",
		"should not add Python",
	} {
		if !strings.Contains(template, required) {
			t.Fatalf("bootstrap template missing Go-only guidance %q", required)
		}
	}
}

func TestBIGGO1472LaneReportCapturesGoOnlyGuardState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1472-go-only-guard.md")

	for _, needle := range []string{
		"BIG-GO-1472",
		"`0` physical",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/internal/bootstrap/*`",
		"`scripts/dev_bootstrap.sh`",
		"`find . -type f -name '*.py' | sort`",
		"`find . -maxdepth 3 \\( -name 'pytest.ini' -o -name 'conftest.py' -o -name 'pyproject.toml' -o -name 'requirements*.txt' -o -name 'tox.ini' -o -name '.python-version' \\) | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1472'`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
