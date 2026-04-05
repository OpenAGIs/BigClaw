package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO1361LegacyCoreModuleReplacementManifestMatchesRetiredModules(t *testing.T) {
	replacements := migration.LegacyCoreModuleReplacements()
	if len(replacements) != 12 {
		t.Fatalf("expected 12 legacy core module replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"src/bigclaw/connectors.py": {
			replacementKind: "go-intake-contract",
			goReplacements: []string{
				"bigclaw-go/internal/intake/types.go",
				"bigclaw-go/internal/intake/connector.go",
				"bigclaw-go/internal/intake/connector_test.go",
			},
			evidencePaths: []string{
				"docs/go-domain-intake-parity-matrix.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche9_test.go",
			},
			statusNeedle: "Go intake connector",
		},
		"src/bigclaw/mapping.py": {
			replacementKind: "go-intake-mapping",
			goReplacements: []string{
				"bigclaw-go/internal/intake/mapping.go",
				"bigclaw-go/internal/intake/mapping_test.go",
			},
			evidencePaths: []string{
				"docs/go-domain-intake-parity-matrix.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche5_test.go",
			},
			statusNeedle: "Go intake mapping",
		},
		"src/bigclaw/dsl.py": {
			replacementKind: "go-workflow-definition",
			goReplacements: []string{
				"bigclaw-go/internal/workflow/definition.go",
				"bigclaw-go/internal/workflow/definition_test.go",
				"bigclaw-go/internal/workflow/engine.go",
			},
			evidencePaths: []string{
				"docs/go-domain-intake-parity-matrix.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche12_test.go",
			},
			statusNeedle: "Go workflow definition",
		},
		"src/bigclaw/scheduler.py": {
			replacementKind: "go-scheduler-mainline",
			goReplacements: []string{
				"bigclaw-go/internal/scheduler/scheduler.go",
				"bigclaw-go/internal/scheduler/scheduler_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go scheduler mainline",
		},
		"src/bigclaw/workflow.py": {
			replacementKind: "go-workflow-mainline",
			goReplacements: []string{
				"bigclaw-go/internal/workflow/engine.go",
				"bigclaw-go/internal/workflow/model.go",
				"bigclaw-go/internal/workflow/closeout.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go workflow engine",
		},
		"src/bigclaw/queue.py": {
			replacementKind: "go-queue-mainline",
			goReplacements: []string{
				"bigclaw-go/internal/queue/queue.go",
				"bigclaw-go/internal/queue/memory_queue.go",
				"bigclaw-go/internal/queue/sqlite_queue.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
				"bigclaw-go/docs/reports/queue-reliability-report.md",
			},
			statusNeedle: "Go queue implementations",
		},
		"src/bigclaw/planning.py": {
			replacementKind: "go-planning-surface",
			goReplacements: []string{
				"bigclaw-go/internal/planning/planning.go",
				"bigclaw-go/internal/planning/planning_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go planning surface",
		},
		"src/bigclaw/orchestration.py": {
			replacementKind: "go-orchestration-mainline",
			goReplacements: []string{
				"bigclaw-go/internal/workflow/orchestration.go",
				"bigclaw-go/internal/orchestrator/loop.go",
				"bigclaw-go/internal/control/controller.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			statusNeedle: "Go orchestration loop",
		},
		"src/bigclaw/observability.py": {
			replacementKind: "go-observability-surface",
			goReplacements: []string{
				"bigclaw-go/internal/observability/recorder.go",
				"bigclaw-go/internal/observability/task_run.go",
				"bigclaw-go/internal/observability/audit.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/docs/reports/go-control-plane-observability-report.md",
			},
			statusNeedle: "Go audit, recorder",
		},
		"src/bigclaw/reports.py": {
			replacementKind: "go-reporting-surface",
			goReplacements: []string{
				"bigclaw-go/internal/reporting/reporting.go",
				"bigclaw-go/internal/reportstudio/reportstudio.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go",
			},
			statusNeedle: "Go reporting",
		},
		"src/bigclaw/evaluation.py": {
			replacementKind: "go-evaluation-surface",
			goReplacements: []string{
				"bigclaw-go/internal/evaluation/evaluation.go",
				"bigclaw-go/internal/evaluation/evaluation_test.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go",
			},
			statusNeedle: "Go evaluation surface",
		},
		"src/bigclaw/operations.py": {
			replacementKind: "go-operations-surface",
			goReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
				"bigclaw-go/internal/control/controller.go",
				"bigclaw-go/internal/api/server.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go",
			},
			statusNeedle: "Go dashboard, controller, and API server",
		},
	}

	for _, replacement := range replacements {
		want, ok := expected[replacement.RetiredPythonModule]
		if !ok {
			t.Fatalf("unexpected retired module in replacement registry: %+v", replacement)
		}
		if replacement.ReplacementKind != want.replacementKind {
			t.Fatalf("replacement kind for %s = %q, want %q", replacement.RetiredPythonModule, replacement.ReplacementKind, want.replacementKind)
		}
		assertExactStringSlice(t, replacement.GoReplacements, want.goReplacements, replacement.RetiredPythonModule+" go replacements")
		assertExactStringSlice(t, replacement.EvidencePaths, want.evidencePaths, replacement.RetiredPythonModule+" evidence paths")
		if !strings.Contains(replacement.Status, want.statusNeedle) {
			t.Fatalf("replacement status for %s missing %q: %q", replacement.RetiredPythonModule, want.statusNeedle, replacement.Status)
		}
	}
}

func TestBIGGO1361LegacyCoreModuleReplacementReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyCoreModuleReplacements() {
		for _, relativePath := range replacement.GoReplacements {
			if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
				t.Fatalf("expected Go replacement path to exist for %s: %s (%v)", replacement.RetiredPythonModule, relativePath, err)
			}
		}
		for _, relativePath := range replacement.EvidencePaths {
			if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
				t.Fatalf("expected evidence path to exist for %s: %s (%v)", replacement.RetiredPythonModule, relativePath, err)
			}
		}
	}
}

func TestBIGGO1361LegacyCoreModuleReplacementLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1361-legacy-core-module-replacement.md")

	for _, needle := range []string{
		"BIG-GO-1361",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw/connectors.py`",
		"`src/bigclaw/scheduler.py`",
		"`src/bigclaw/observability.py`",
		"`src/bigclaw/operations.py`",
		"`bigclaw-go/internal/migration/legacy_core_modules.go`",
		"`bigclaw-go/internal/intake/connector.go`",
		"`bigclaw-go/internal/scheduler/scheduler.go`",
		"`bigclaw-go/internal/workflow/orchestration.go`",
		"`bigclaw-go/internal/observability/recorder.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`docs/go-domain-intake-parity-matrix.md`",
		"`bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`",
		"`find . -name '*.py' | wc -l`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1361LegacyCoreModuleReplacement",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
