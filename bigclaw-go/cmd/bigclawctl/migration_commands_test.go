package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/refill"
)

func captureStdout(t *testing.T, fn func() error) ([]byte, error) {
	t.Helper()
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()
	runErr := fn()
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	return output, runErr
}

func TestRunDevSmokeJSONOutput(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return runDevSmoke([]string{"--json"})
	})
	if err != nil {
		t.Fatalf("run dev-smoke: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode output: %v (%s)", err, string(output))
	}
	if payload["status"] != "ok" {
		t.Fatalf("expected ok status, got %+v", payload)
	}
	if payload["accepted"] != true {
		t.Fatalf("expected accepted=true, got %+v", payload)
	}
}

func TestRunCreateIssuesCreatesOnlyMissing(t *testing.T) {
	requestCount := 0
	createdPayloads := []map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"title": "[EPIC] BIG-EPIC-1 任务接入与连接器"},
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues"):
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			createdPayloads = append(createdPayloads, payload)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": len(createdPayloads),
				"title":  payload["title"],
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	output, err := captureStdout(t, func() error {
		return runCreateIssues([]string{
			"--plan", "v2-ops",
			"--owner", "OpenAGIs",
			"--repo-name", "BigClaw",
			"--api-base", server.URL,
			"--token", "test-token",
			"--json",
		})
	})
	if err != nil {
		t.Fatalf("run create-issues: %v", err)
	}
	if requestCount < 2 {
		t.Fatalf("expected list + create requests, got %d", requestCount)
	}
	if len(createdPayloads) != len(createIssuePlans["v2-ops"]) {
		t.Fatalf("expected %d created issues, got %d", len(createIssuePlans["v2-ops"]), len(createdPayloads))
	}
	if labels, ok := createdPayloads[0]["labels"].([]any); !ok || len(labels) != 3 {
		t.Fatalf("expected v2 labels in create payload, got %+v", createdPayloads[0])
	}
	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode output: %v (%s)", err, string(output))
	}
	if payload["created_count"] != float64(len(createIssuePlans["v2-ops"])) {
		t.Fatalf("unexpected created_count payload: %+v", payload)
	}
}

func TestRunIssueRoutesStateShortcutToLocalIssues(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "workflow.md"), []byte("# workflow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	storePath := filepath.Join(repoRoot, "local-issues.json")
	store, err := refill.LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	if _, err := store.CreateIssue(refill.LocalIssueCreateParams{
		Identifier: "BIG-GO-902",
		Title:      "script migration",
		State:      "Todo",
	}); err != nil {
		t.Fatalf("create issue: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	output, err := captureStdout(t, func() error {
		return runIssue([]string{"state", "BIG-GO-902", "In Progress", "--json"})
	})
	if err != nil {
		t.Fatalf("run issue state: %v", err)
	}
	if !bytes.Contains(output, []byte(`"state": "In Progress"`)) {
		t.Fatalf("expected updated state in output, got %s", string(output))
	}
}

func TestRunPanelUsesSymphonyFromPATH(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "workflow.md"), []byte("# workflow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "symphony.log")
	scriptPath := filepath.Join(binDir, "symphony")
	script := "#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" > \"" + logPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := runPanel([]string{"--repo", repoRoot, "status"}); err != nil {
		t.Fatalf("run panel: %v", err)
	}
	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(logBody), "panel --workflow "+filepath.Join(repoRoot, "workflow.md")+" status") {
		t.Fatalf("unexpected symphony invocation: %s", string(logBody))
	}
}

func TestTranslateCompatExecArgsAddsRepoAndResolvesRelativeOverride(t *testing.T) {
	repoRoot := t.TempDir()
	invocationDir := filepath.Join(repoRoot, "nested")
	overrideRepo := filepath.Join(invocationDir, "alt-repo")
	if err := os.MkdirAll(overrideRepo, 0o755); err != nil {
		t.Fatal(err)
	}

	translated, err := translateCompatExecArgs([]string{"github-sync", "status", "--repo", "alt-repo"}, repoRoot, invocationDir)
	if err != nil {
		t.Fatalf("translate compat args: %v", err)
	}
	expected := []string{"github-sync", "status", "--repo", overrideRepo}
	if !reflect.DeepEqual(translated, expected) {
		t.Fatalf("unexpected translated args: got %v want %v", translated, expected)
	}

	translated, err = translateCompatExecArgs([]string{"refill", "--json"}, repoRoot, invocationDir)
	if err != nil {
		t.Fatalf("translate compat args without repo: %v", err)
	}
	expected = []string{"refill", "--json", "--repo", repoRoot}
	if !reflect.DeepEqual(translated, expected) {
		t.Fatalf("unexpected translated args without repo: got %v want %v", translated, expected)
	}
}

func TestRunDevBootstrapSkipsLegacyPythonByDefault(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "bigclaw-go"), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var commands [][]string
	err := runDevBootstrap(devBootstrapOptions{
		RepoRoot: repoRoot,
		Stdout:   &stdout,
		RunCommand: func(cmd *exec.Cmd) error {
			commands = append(commands, append([]string{filepath.Base(cmd.Path)}, cmd.Args[1:]...))
			return nil
		},
	})
	if err != nil {
		t.Fatalf("run dev bootstrap: %v", err)
	}
	expected := [][]string{{"go", "test", "./..."}}
	if !reflect.DeepEqual(commands, expected) {
		t.Fatalf("unexpected command sequence: got %v want %v", commands, expected)
	}
	if !strings.Contains(stdout.String(), "BigClaw Go development environment is ready.") {
		t.Fatalf("expected Go-ready message, got %s", stdout.String())
	}
}

func TestRunDevBootstrapIncludesLegacyPythonSteps(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "bigclaw-go"), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var commands [][]string
	err := runDevBootstrap(devBootstrapOptions{
		RepoRoot:     repoRoot,
		LegacyPython: true,
		Stdout:       &stdout,
		RunCommand: func(cmd *exec.Cmd) error {
			commands = append(commands, append([]string{filepath.Base(cmd.Path)}, cmd.Args[1:]...))
			return nil
		},
	})
	if err != nil {
		t.Fatalf("run dev bootstrap with legacy python: %v", err)
	}
	expected := [][]string{
		{"go", "test", "./..."},
		{"python3", "-m", "venv", filepath.Join(repoRoot, ".venv")},
		{"python", "-m", "pip", "install", "-U", "pip"},
		{"python", "-m", "pip", "install", "-e", repoRoot + "[dev]"},
		{"python", "-m", "pytest"},
	}
	if !reflect.DeepEqual(commands, expected) {
		t.Fatalf("unexpected legacy bootstrap sequence: got %v want %v", commands, expected)
	}
	if !strings.Contains(stdout.String(), "legacy Python migration environments are ready") {
		t.Fatalf("expected legacy-ready message, got %s", stdout.String())
	}
}
