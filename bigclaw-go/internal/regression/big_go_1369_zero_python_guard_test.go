package regression

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1369RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1369OpsReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1369OpsWrappersStayGoNative(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	tests := []struct {
		wrapper string
		args    []string
		needle  string
	}{
		{wrapper: "scripts/ops/bigclawctl", args: []string{"--help"}, needle: "usage: bigclawctl <github-sync|workspace|automation|refill|local-issues|create-issues|dev-smoke|symphony|issue|panel> ..."},
		{wrapper: "scripts/ops/bigclaw-issue", args: []string{"--help"}, needle: "usage: bigclawctl issue [flags] [args...]"},
		{wrapper: "scripts/ops/bigclaw-panel", args: []string{"--help"}, needle: "usage: bigclawctl panel [flags] [args...]"},
		{wrapper: "scripts/ops/bigclaw-symphony", args: []string{"--help"}, needle: "usage: bigclawctl symphony [flags] [args...]"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(filepath.Base(tc.wrapper), func(t *testing.T) {
			output := runOpsWrapper(t, rootRepo, rootRepo, tc.wrapper, tc.args...)
			if !strings.Contains(string(output), tc.needle) {
				t.Fatalf("expected %s help output to contain %q, got %s", tc.wrapper, tc.needle, string(output))
			}
		})
	}
}

func TestBIGGO1369BigclawctlWrapperResolvesRelativeRepoFromInvocationDir(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	invocationDir := filepath.Join(rootRepo, "reports")

	output := runOpsWrapper(t, rootRepo, invocationDir, "scripts/ops/bigclawctl",
		"local-issues", "list",
		"--repo", "..",
		"--local-issues", "local-issues.json",
		"--json",
	)

	expectedLocalIssuesPath := filepath.Join(rootRepo, "local-issues.json")
	if !bytes.Contains(output, []byte(`"local_issues": "`+expectedLocalIssuesPath+`"`)) {
		t.Fatalf("expected resolved local issue path %q in output, got %s", expectedLocalIssuesPath, string(output))
	}
}

func TestBIGGO1369LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1369-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1369",
		"Repository-wide Python file count: `0`.",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`find . -name '*.py' | wc -l`",
		"`bash scripts/ops/bigclawctl local-issues list --repo .. --local-issues local-issues.json --json`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1369",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func runOpsWrapper(t *testing.T, rootRepo string, dir string, wrapper string, args ...string) []byte {
	t.Helper()

	commandArgs := append([]string{filepath.Join(rootRepo, wrapper)}, args...)
	cmd := exec.Command("bash", commandArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run %s %v from %s: %v (%s)", wrapper, args, dir, err, string(output))
	}
	return output
}
