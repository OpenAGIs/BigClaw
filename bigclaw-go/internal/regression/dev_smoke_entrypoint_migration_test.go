package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootDevSmokeEntrypointStaysGoOnly(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"scripts/dev_smoke.py",
		"src/bigclaw/deprecation.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to stay absent: %s", relativePath)
		}
	}

	requiredFiles := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
	}
	for _, relativePath := range requiredFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go smoke entrypoint support file to exist: %s (%v)", relativePath, err)
		}
	}

	readme := readRepoFile(t, repoRoot, "README.md")
	requiredSnippets := []string{
		"## Go Dev Smoke Verify",
		"`scripts/dev_smoke.py` is removed; do not use a Python fallback for root smoke validation.",
		"bash scripts/ops/bigclawctl dev-smoke",
	}
	for _, needle := range requiredSnippets {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing required dev-smoke guidance %q", needle)
		}
	}

	if strings.Contains(readme, "python3 scripts/dev_smoke.py") {
		t.Fatalf("README.md should not advertise legacy Python dev smoke command")
	}
}
