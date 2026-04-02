package regression

import (
	"os/exec"
	"strings"
	"testing"
)

func TestTrackedPythonFloorLockedToPackageRoot(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	cmd := exec.Command("git", "ls-files", "*.py")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files failed: %v", err)
	}

	lines := strings.Fields(string(output))
	if len(lines) != 1 {
		t.Fatalf("expected exactly one tracked Python file, got %d: %v", len(lines), lines)
	}
	if lines[0] != "src/bigclaw/__init__.py" {
		t.Fatalf("unexpected tracked Python file: %v", lines)
	}
}
