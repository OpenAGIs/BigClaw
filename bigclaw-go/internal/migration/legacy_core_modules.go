package migration

func LegacyCoreModuleReplacements() []LegacyModuleReplacement {
	return []LegacyModuleReplacement{
		{
			RetiredPythonModule: "src/bigclaw/connectors.py",
			ReplacementKind:     "go-intake-contract",
			GoReplacements: []string{
				"bigclaw-go/internal/intake/types.go",
				"bigclaw-go/internal/intake/connector.go",
				"bigclaw-go/internal/intake/connector_test.go",
			},
			EvidencePaths: []string{
				"docs/go-domain-intake-parity-matrix.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche9_test.go",
			},
			Status: "retired Python connector layer replaced by the Go intake connector and source-issue contract surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/mapping.py",
			ReplacementKind:     "go-intake-mapping",
			GoReplacements: []string{
				"bigclaw-go/internal/intake/mapping.go",
				"bigclaw-go/internal/intake/mapping_test.go",
			},
			EvidencePaths: []string{
				"docs/go-domain-intake-parity-matrix.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche5_test.go",
			},
			Status: "retired Python mapping helpers replaced by the Go intake mapping surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/dsl.py",
			ReplacementKind:     "go-workflow-definition",
			GoReplacements: []string{
				"bigclaw-go/internal/workflow/definition.go",
				"bigclaw-go/internal/workflow/definition_test.go",
				"bigclaw-go/internal/workflow/engine.go",
			},
			EvidencePaths: []string{
				"docs/go-domain-intake-parity-matrix.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche12_test.go",
			},
			Status: "retired Python DSL module replaced by the Go workflow definition and execution surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/scheduler.py",
			ReplacementKind:     "go-scheduler-mainline",
			GoReplacements: []string{
				"bigclaw-go/internal/scheduler/scheduler.go",
				"bigclaw-go/internal/scheduler/scheduler_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python scheduler loop replaced by the Go scheduler mainline",
		},
		{
			RetiredPythonModule: "src/bigclaw/workflow.py",
			ReplacementKind:     "go-workflow-mainline",
			GoReplacements: []string{
				"bigclaw-go/internal/workflow/engine.go",
				"bigclaw-go/internal/workflow/model.go",
				"bigclaw-go/internal/workflow/closeout.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python workflow module replaced by the Go workflow engine, model, and closeout surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/queue.py",
			ReplacementKind:     "go-queue-mainline",
			GoReplacements: []string{
				"bigclaw-go/internal/queue/queue.go",
				"bigclaw-go/internal/queue/memory_queue.go",
				"bigclaw-go/internal/queue/sqlite_queue.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
				"bigclaw-go/docs/reports/queue-reliability-report.md",
			},
			Status: "retired Python queue module replaced by the Go queue implementations and reliability evidence",
		},
		{
			RetiredPythonModule: "src/bigclaw/planning.py",
			ReplacementKind:     "go-planning-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/planning/planning.go",
				"bigclaw-go/internal/planning/planning_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python planning helpers replaced by the Go planning surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/orchestration.py",
			ReplacementKind:     "go-orchestration-mainline",
			GoReplacements: []string{
				"bigclaw-go/internal/workflow/orchestration.go",
				"bigclaw-go/internal/orchestrator/loop.go",
				"bigclaw-go/internal/control/controller.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python orchestration module replaced by the Go orchestration loop and controller surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/observability.py",
			ReplacementKind:     "go-observability-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/observability/recorder.go",
				"bigclaw-go/internal/observability/task_run.go",
				"bigclaw-go/internal/observability/audit.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/docs/reports/go-control-plane-observability-report.md",
			},
			Status: "retired Python observability module replaced by the Go audit, recorder, and task-run evidence surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/reports.py",
			ReplacementKind:     "go-reporting-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/reporting/reporting.go",
				"bigclaw-go/internal/reportstudio/reportstudio.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go",
			},
			Status: "retired Python reporting module replaced by the Go reporting and report-studio surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/evaluation.py",
			ReplacementKind:     "go-evaluation-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/evaluation/evaluation.go",
				"bigclaw-go/internal/evaluation/evaluation_test.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go",
			},
			Status: "retired Python evaluation module replaced by the Go evaluation surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/operations.py",
			ReplacementKind:     "go-operations-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
				"bigclaw-go/internal/control/controller.go",
				"bigclaw-go/internal/api/server.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go",
			},
			Status: "retired Python operations module replaced by the Go dashboard, controller, and API server surface",
		},
	}
}
