package regression

import (
	"strings"
	"testing"
)

func TestRuntimeSchedulerOrchestrationMigrationPlanDocs(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/runtime-scheduler-orchestration-migration-plan.md",
			substrings: []string{
				"BIG-GO-906",
				"tests/test_runtime.py",
				"tests/test_scheduler.py",
				"tests/test_orchestration.py",
				"tests/test_workflow.py",
				"tests/test_runtime_matrix.py",
				"bigclaw-go/scripts/e2e/run_task_smoke.py",
				"bigclaw-go/scripts/e2e/export_validation_bundle.py",
				"cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906 && PYTHONPATH=src python3 -m pytest tests/test_runtime.py tests/test_scheduler.py tests/test_orchestration.py tests/test_workflow.py tests/test_runtime_matrix.py -q",
				"cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906/bigclaw-go && go test ./internal/worker ./internal/scheduler ./internal/workflow ./internal/orchestrator -count=1",
				"Branch name: `codex/BIG-GO-906-runtime-scheduler-orchestration-migration`",
				"Runtime closeout behavior can drift silently if Python tests are deleted before",
			},
		},
		{
			path: "docs/migration.md",
			substrings: []string{
				"docs/reports/runtime-scheduler-orchestration-migration-plan.md",
			},
		},
		{
			path: "docs/reports/migration-readiness-report.md",
			substrings: []string{
				"docs/reports/runtime-scheduler-orchestration-migration-plan.md",
				"first-batch test ports, script conversion order, validation commands, branch",
			},
		},
		{
			path: "docs/reports/migration-plan-review-notes.md",
			substrings: []string{
				"docs/reports/runtime-scheduler-orchestration-migration-plan.md",
				"legacy Python execution surfaces",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
