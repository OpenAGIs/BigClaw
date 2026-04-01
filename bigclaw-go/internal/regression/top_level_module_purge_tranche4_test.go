package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche4(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/governance.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/reports.py",
		"tests/test_observability.py",
		"tests/test_reports.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/observability/audit_spec.go",
		"bigclaw-go/internal/observability/audit_spec_test.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/observability/recorder_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestTopLevelModulePurgePythonCountDrops(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	count := 0
	if err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".venv" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".py") {
			count++
		}
		return nil
	}); err != nil {
		t.Fatalf("walk repo root: %v", err)
	}

	const prePurgePythonFileCount = 66
	const expectedPostPurgePythonFileCount = 61
	if count >= prePurgePythonFileCount {
		t.Fatalf("expected Python file count to drop below %d, got %d", prePurgePythonFileCount, count)
	}
	if count != expectedPostPurgePythonFileCount {
		t.Fatalf("expected Python file count to land at %d after tranche 4 purge, got %d", expectedPostPurgePythonFileCount, count)
	}
}
