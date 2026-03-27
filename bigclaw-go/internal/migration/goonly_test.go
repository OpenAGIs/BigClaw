package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildReportClassifiesRemainingNonGoSurface(t *testing.T) {
	repoRoot := t.TempDir()
	files := map[string]string{
		"src/bigclaw/runtime.py":                     "print('runtime')\n",
		"tests/test_runtime.py":                      "def test_runtime():\n    pass\n",
		"scripts/ops/bigclaw_workspace_bootstrap.py": "print('ops')\n",
		"scripts/dev_bootstrap.sh":                   "#!/usr/bin/env bash\nexit 0\n",
		"bigclaw-go/scripts/e2e/run_all.sh":          "#!/usr/bin/env bash\nexit 0\n",
		"bigclaw-go/scripts/e2e/run_all_test.py":     "def test_run_all():\n    pass\n",
		"pyproject.toml":                             "[project]\nname='bigclaw'\n",
		"local-issues.json":                          "{\n  \"issues\": [\n    {\"identifier\": \"BIG-VNEXT-GO-102\", \"state\": \"In Progress\"},\n    {\"identifier\": \"BIG-VNEXT-GO-106\", \"state\": \"Done\"}\n  ]\n}\n",
	}
	for relative, body := range files {
		absolute := filepath.Join(repoRoot, relative)
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", relative, err)
		}
		if err := os.WriteFile(absolute, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", relative, err)
		}
	}

	report, err := BuildReport(repoRoot)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if report.Summary.NonGoFiles != 7 {
		t.Fatalf("expected 7 classified files, got %d", report.Summary.NonGoFiles)
	}
	if report.Summary.LegacyRuntimeFiles != 1 || report.Summary.LegacyTestFiles != 1 {
		t.Fatalf("unexpected legacy summary: %+v", report.Summary)
	}
	if report.Summary.PythonScriptFiles != 2 {
		t.Fatalf("expected 2 python script files, got %+v", report.Summary)
	}
	if report.Summary.ShellScriptFiles != 2 {
		t.Fatalf("expected 2 shell script files, got %+v", report.Summary)
	}
	if report.Summary.PythonToolchainFiles != 1 || report.Summary.ValidationHarnessFiles != 2 {
		t.Fatalf("unexpected toolchain or harness summary: %+v", report.Summary)
	}
	if report.Summary.ParallelSliceCount < 10 || report.Summary.FirstBatchSliceCount < 3 {
		t.Fatalf("expected parallel slice plan, got %+v", report.Summary)
	}
	if report.Summary.InProgressSliceCount == 0 || report.Summary.DoneSliceCount == 0 {
		t.Fatalf("expected tracker status overlay in summary, got %+v", report.Summary)
	}
	if got := findSliceStatus(report.Slices, "BIG-VNEXT-GO-102"); got != "in_progress" {
		t.Fatalf("expected BIG-VNEXT-GO-102 in_progress, got %q", got)
	}
	if got := findSliceStatus(report.Slices, "BIG-VNEXT-GO-106"); got != "done" {
		t.Fatalf("expected BIG-VNEXT-GO-106 done, got %q", got)
	}
}

func TestRenderMarkdownIncludesStrategyAndFirstBatch(t *testing.T) {
	report, err := BuildReport(t.TempDir())
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	body := RenderMarkdown(report)
	for _, needle := range []string{
		"# BigClaw Go-Only Migration Plan",
		"docs/reports/go-only-migration-inventory.json",
		"## Execution Snapshot",
		"Slice status counts:",
		"Tracker status:",
		"BIG-VNEXT-GO-101",
		"BIG-VNEXT-GO-110",
		"## Branch And PR Strategy",
		"Start `BIG-VNEXT-GO-102`, `BIG-VNEXT-GO-103`, and `BIG-VNEXT-GO-104` in parallel",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("markdown missing %q", needle)
		}
	}
}

func findSliceStatus(slices []Slice, identifier string) string {
	for _, slice := range slices {
		if slice.Identifier == identifier {
			return slice.Status
		}
	}
	return ""
}
