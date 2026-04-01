package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOpsWrapperScriptsDispatchExpectedArgs(t *testing.T) {
	repoRoot := repoRootFromCommandDir(t)

	t.Run("bigclaw-github-sync", func(t *testing.T) {
		lines := runOpsWrapperScript(t, repoRoot, "bigclaw-github-sync", []string{"status", "--json"}, nil)
		if !reflect.DeepEqual(lines, []string{"github-sync", "status", "--json"}) {
			t.Fatalf("unexpected wrapper args: got=%+v", lines)
		}
	})

	t.Run("bigclaw-refill-queue", func(t *testing.T) {
		lines := runOpsWrapperScript(t, repoRoot, "bigclaw-refill-queue", []string{"--apply"}, nil)
		if !reflect.DeepEqual(lines, []string{"refill", "--apply"}) {
			t.Fatalf("unexpected wrapper args: got=%+v", lines)
		}
	})

	t.Run("symphony-workspace-bootstrap", func(t *testing.T) {
		lines := runOpsWrapperScript(t, repoRoot, "symphony-workspace-bootstrap", []string{"bootstrap", "--json"}, nil)
		if !reflect.DeepEqual(lines, []string{"workspace", "bootstrap", "--json"}) {
			t.Fatalf("unexpected wrapper args: got=%+v", lines)
		}
	})

	t.Run("bigclaw-workspace-bootstrap injects defaults", func(t *testing.T) {
		lines := runOpsWrapperScript(t, repoRoot, "bigclaw-workspace-bootstrap", []string{"--workspace", "/tmp/ws"}, nil)
		want := []string{
			"workspace",
			"bootstrap",
			"--workspace",
			"/tmp/ws",
			"--repo-url",
			"git@github.com:OpenAGIs/BigClaw.git",
			"--cache-key",
			"openagis-bigclaw",
		}
		if !reflect.DeepEqual(lines, want) {
			t.Fatalf("unexpected wrapper args: got=%+v want=%+v", lines, want)
		}
	})

	t.Run("bigclaw-workspace-bootstrap preserves explicit values", func(t *testing.T) {
		env := []string{
			"BIGCLAW_BOOTSTRAP_REPO_URL=ssh://example/repo.git",
			"BIGCLAW_BOOTSTRAP_CACHE_KEY=env-cache",
		}
		lines := runOpsWrapperScript(t, repoRoot, "bigclaw-workspace-bootstrap", []string{
			"--workspace", "/tmp/ws",
			"--repo-url", "ssh://explicit/repo.git",
			"--cache-key=explicit-cache",
		}, env)
		want := []string{
			"workspace",
			"bootstrap",
			"--workspace",
			"/tmp/ws",
			"--repo-url",
			"ssh://explicit/repo.git",
			"--cache-key=explicit-cache",
		}
		if !reflect.DeepEqual(lines, want) {
			t.Fatalf("unexpected wrapper args: got=%+v want=%+v", lines, want)
		}
	})

	t.Run("symphony-workspace-validate translates legacy flags", func(t *testing.T) {
		lines := runOpsWrapperScript(t, repoRoot, "symphony-workspace-validate", []string{
			"--repo-url", "git@github.com:OpenAGIs/BigClaw.git",
			"--workspace-root", "/tmp/ws",
			"--issues", "BIG-1", "BIG-2",
			"--report-file", "/tmp/report.json",
			"--no-cleanup",
			"--json",
		}, nil)
		want := []string{
			"workspace",
			"validate",
			"--repo-url",
			"git@github.com:OpenAGIs/BigClaw.git",
			"--workspace-root",
			"/tmp/ws",
			"--issues",
			"BIG-1,BIG-2",
			"--report",
			"/tmp/report.json",
			"--cleanup=false",
			"--json",
		}
		if !reflect.DeepEqual(lines, want) {
			t.Fatalf("unexpected wrapper args: got=%+v want=%+v", lines, want)
		}
	})
}

func runOpsWrapperScript(t *testing.T, repoRoot string, name string, args []string, extraEnv []string) []string {
	t.Helper()

	tempRepo := t.TempDir()
	opsDir := filepath.Join(tempRepo, "scripts", "ops")
	if err := os.MkdirAll(opsDir, 0o755); err != nil {
		t.Fatalf("mkdir ops dir: %v", err)
	}

	wrapperSource := filepath.Join(repoRoot, "scripts", "ops", name)
	wrapperBody, err := os.ReadFile(wrapperSource)
	if err != nil {
		t.Fatalf("read wrapper source: %v", err)
	}
	wrapperPath := filepath.Join(opsDir, name)
	if err := os.WriteFile(wrapperPath, wrapperBody, 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	stubPath := filepath.Join(opsDir, "bigclawctl")
	stubBody := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$@\"\n"
	if err := os.WriteFile(stubPath, []byte(stubBody), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	cmd := exec.Command("bash", append([]string{wrapperPath}, args...)...)
	cmd.Env = append(os.Environ(), extraEnv...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run wrapper %s: %v (%s)", name, err, string(output))
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func repoRootFromCommandDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}
