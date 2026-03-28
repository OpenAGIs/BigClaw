package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/refill"
	"bigclaw-go/internal/testharness"
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

func captureRunStdout(t *testing.T, args []string) ([]byte, int) {
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
	code := run(args)
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	return output, code
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

func TestHelpCommandsCoverLegacyShimEntrypoints(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "dev-smoke",
			args: []string{"dev-smoke", "--help"},
			want: "usage: bigclawctl dev-smoke [flags]",
		},
		{
			name: "create-issues",
			args: []string{"create-issues", "--help"},
			want: "usage: bigclawctl create-issues [flags]",
		},
		{
			name: "github-sync",
			args: []string{"github-sync", "--help"},
			want: "usage: bigclawctl github-sync <install|status|sync> [flags]",
		},
		{
			name: "workspace",
			args: []string{"workspace", "--help"},
			want: "usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]",
		},
		{
			name: "legacy-python",
			args: []string{"legacy-python", "--help"},
			want: "usage: bigclawctl legacy-python <compile-check|pytest> [flags]",
		},
		{
			name: "pytest-harness",
			args: []string{"pytest-harness", "--help"},
			want: "usage: bigclawctl pytest-harness [flags]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, code := captureRunStdout(t, tc.args)
			if code != 0 {
				t.Fatalf("expected exit 0, got %d (stdout=%s)", code, string(output))
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected %q in output, got %s", tc.want, string(output))
			}
		})
	}
}

func TestRunPytestHarnessJSONOutput(t *testing.T) {
	projectRoot := testharness.ProjectRoot(t)

	output, err := captureStdout(t, func() error {
		return runPytestHarness([]string{"--project-root", projectRoot, "--json"})
	})
	if err != nil {
		t.Fatalf("run pytest-harness: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode output: %v (%s)", err, string(output))
	}
	if payload["status"] != "ok" {
		t.Fatalf("expected ok status, got %+v", payload)
	}
	if payload["project_root"] != "." {
		t.Fatalf("unexpected project_root: %+v", payload)
	}
	if payload["pyproject_path"] != "pyproject.toml" {
		t.Fatalf("unexpected pyproject_path: %+v", payload)
	}
	if payload["pyproject_exists"] != true {
		t.Fatalf("expected pyproject_exists=true, got %+v", payload)
	}
	if payload["pyproject_declares_pytest"] != false || payload["pyproject_has_pytest_config"] != false {
		t.Fatalf("expected pyproject pytest flags to be false after baseline cleanup, got %+v", payload)
	}
	commandRefs, ok := payload["pytest_command_ref_files"].([]any)
	if !ok || len(commandRefs) != 0 {
		t.Fatalf("expected 0 pytest command ref files, got %+v", payload["pytest_command_ref_files"])
	}
	if payload["conftest_exists"] != false {
		t.Fatalf("expected conftest_exists=false, got %+v", payload)
	}
	if payload["conftest_path"] != "tests/conftest.py" {
		t.Fatalf("unexpected conftest_path: %+v", payload)
	}
	if payload["inventory_summary"] != "tests=24 bigclaw_imports=24 pytest_imports=0 pytest_command_refs=0" {
		t.Fatalf("unexpected inventory summary: %+v", payload)
	}
	deleteStatus, ok := payload["conftest_delete_status"].(map[string]any)
	if !ok {
		t.Fatalf("expected conftest_delete_status object, got %+v", payload["conftest_delete_status"])
	}
	if deleteStatus["can_delete"] != true {
		t.Fatalf("expected can_delete=true, got %+v", deleteStatus)
	}
	if deleteStatus["legacy_test_modules"] != float64(24) || deleteStatus["bigclaw_import_modules"] != float64(24) || deleteStatus["pytest_import_modules"] != float64(0) {
		t.Fatalf("unexpected delete status counts: %+v", deleteStatus)
	}
	if deleteStatus["summary"] != "conftest_delete_ready=true blockers=none" {
		t.Fatalf("unexpected delete status summary: %+v", deleteStatus)
	}
	legacyDeleteStatus, ok := payload["legacy_pytest_delete_status"].(map[string]any)
	if !ok {
		t.Fatalf("expected legacy_pytest_delete_status object, got %+v", payload["legacy_pytest_delete_status"])
	}
	if legacyDeleteStatus["can_delete"] != false {
		t.Fatalf("expected legacy_pytest_delete_status.can_delete=false, got %+v", legacyDeleteStatus)
	}
	if legacyDeleteStatus["legacy_test_modules"] != float64(24) || legacyDeleteStatus["bigclaw_import_modules"] != float64(24) || legacyDeleteStatus["pytest_import_modules"] != float64(0) || legacyDeleteStatus["pytest_command_refs"] != float64(0) {
		t.Fatalf("unexpected legacy delete status counts: %+v", legacyDeleteStatus)
	}
	if legacyDeleteStatus["summary"] != "legacy_pytest_delete_ready=false blockers=24 legacy pytest modules remain under tests/; 24 legacy pytest modules still import bigclaw from src/" {
		t.Fatalf("unexpected legacy delete status summary: %+v", legacyDeleteStatus)
	}
	if payload["conftest_uses_pytest_plugins"] != false {
		t.Fatalf("expected conftest_uses_pytest_plugins=false, got %+v", payload)
	}
}

func TestRunPytestHarnessWritesReportFile(t *testing.T) {
	projectRoot := testharness.ProjectRoot(t)
	reportPath := filepath.Join(t.TempDir(), "pytest-harness-status.json")

	if err := runPytestHarness([]string{"--project-root", projectRoot, "--report-path", reportPath}); err != nil {
		t.Fatalf("run pytest-harness with report path: %v", err)
	}

	body, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report file: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode report file: %v (%s)", err, string(body))
	}
	if payload["inventory_summary"] != "tests=24 bigclaw_imports=24 pytest_imports=0 pytest_command_refs=0" {
		t.Fatalf("unexpected inventory summary in report file: %+v", payload)
	}
	deleteStatus, ok := payload["conftest_delete_status"].(map[string]any)
	if !ok {
		t.Fatalf("expected conftest_delete_status object, got %+v", payload["conftest_delete_status"])
	}
	if deleteStatus["summary"] != "conftest_delete_ready=true blockers=none" {
		t.Fatalf("unexpected delete status summary in report file: %+v", deleteStatus)
	}
	legacyDeleteStatus, ok := payload["legacy_pytest_delete_status"].(map[string]any)
	if !ok {
		t.Fatalf("expected legacy_pytest_delete_status object, got %+v", payload["legacy_pytest_delete_status"])
	}
	if legacyDeleteStatus["summary"] != "legacy_pytest_delete_ready=false blockers=24 legacy pytest modules remain under tests/; 24 legacy pytest modules still import bigclaw from src/" {
		t.Fatalf("unexpected legacy delete status summary in report file: %+v", legacyDeleteStatus)
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

	testharness.Chdir(t, repoRoot)

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
	testharness.PrependPathEnv(t, binDir)

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
