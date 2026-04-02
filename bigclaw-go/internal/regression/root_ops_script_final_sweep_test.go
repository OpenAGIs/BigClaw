package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootAndOpsScriptPythonSweepStaysEmpty(t *testing.T) {
	repoRoot := repoRoot(t)

	rootMatches, err := filepath.Glob(filepath.Join(repoRoot, "scripts", "*.py"))
	if err != nil {
		t.Fatalf("glob root scripts: %v", err)
	}
	if len(rootMatches) != 0 {
		t.Fatalf("expected no Python files under scripts/*.py, got %v", rootMatches)
	}

	opsDir := filepath.Join(repoRoot, "scripts", "ops")
	if info, err := os.Stat(opsDir); err == nil {
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory when present", opsDir)
		}
		opsMatches, err := filepath.Glob(filepath.Join(opsDir, "*.py"))
		if err != nil {
			t.Fatalf("glob ops scripts: %v", err)
		}
		if len(opsMatches) != 0 {
			t.Fatalf("expected no Python files under scripts/ops/*.py, got %v", opsMatches)
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat scripts/ops: %v", err)
	}

	report := readRepoFile(t, repoRoot, "docs/reports/root-ops-script-final-sweep.md")
	for _, needle := range []string{
		"BIG-GO-982",
		"`scripts/*.py`",
		"`scripts/ops/*.py`",
		"Repository-wide Python file count before this sweep:",
		"Repository-wide Python file count after this sweep:",
		"Net change from this issue:",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("final sweep report missing %q", needle)
		}
	}
}
