package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1514ScriptsDirectoriesRemainPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, relativeDir := range []string{
		"scripts",
		"scripts/ops",
	} {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected %s to remain Python-free, found %v", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1514RetiredRefillWrapperStaysDeleted(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	deletedPaths := []string{
		"scripts/ops/bigclaw_refill_queue.py",
	}
	for _, relativePath := range deletedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired refill wrapper to stay deleted: %s", relativePath)
		}
	}

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/refill/queue.go",
	}
	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected refill replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1514RefillWrapperDeletionEvidenceIsRecorded(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1514-refill-wrapper-sweep.md")
	for _, needle := range []string{
		"BIG-GO-1514",
		"Before count: `0` physical `.py` files.",
		"After count: `0` physical `.py` files.",
		"`scripts`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`scripts/ops/bigclaw_refill_queue.py`",
		"`7f1d265e9deb6e3543bc41f23485d1e3c800c71d`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts scripts/ops -type f -name '*.py' -print | sort`",
		"go test -count=1 ./internal/regression -run 'TestBIGGO1514",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}

	migrationPlan := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")
	if !strings.Contains(migrationPlan, "retired `scripts/ops/bigclaw_refill_queue.py`; use `bigclawctl refill`") {
		t.Fatalf("docs/go-cli-script-migration-plan.md must name the retired refill wrapper path explicitly")
	}

	readme := readRepoFile(t, rootRepo, "README.md")
	if !strings.Contains(readme, "`scripts/ops/bigclaw_refill_queue.py` wrapper stays retired") {
		t.Fatalf("README.md must keep the deleted refill wrapper guidance explicit")
	}
}
