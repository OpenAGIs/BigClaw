package legacyshim

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkspaceShellWrappersExposeGoHelpSurfaces(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "workspace bootstrap wrapper",
			path: filepath.Join(repoRoot, "scripts", "ops", "bigclaw_workspace_bootstrap"),
			want: "usage: bigclawctl workspace bootstrap [flags]",
		},
		{
			name: "workspace runtime wrapper",
			path: filepath.Join(repoRoot, "scripts", "ops", "symphony_workspace_bootstrap"),
			want: "usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]",
		},
		{
			name: "workspace validate wrapper",
			path: filepath.Join(repoRoot, "scripts", "ops", "symphony_workspace_validate"),
			want: "usage: bigclawctl workspace validate [flags]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("bash", tc.path, "--help")
			cmd.Dir = repoRoot
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run %s --help: %v\n%s", tc.path, err, string(output))
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("unexpected help output for %s:\n%s", tc.path, string(output))
			}
		})
	}
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}
