package migration

func LegacyReportingOpsModuleReplacements() []LegacyModuleReplacement {
	return []LegacyModuleReplacement{
		{
			RetiredPythonModule: "src/bigclaw/observability.py",
			ReplacementKind:     "go-observability-runtime",
			GoReplacements: []string{
				"bigclaw-go/internal/observability/recorder.go",
				"bigclaw-go/internal/observability/task_run.go",
				"bigclaw-go/internal/observability/audit.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/observability/recorder_test.go",
				"bigclaw-go/internal/observability/task_run_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python observability surface is replaced by the Go observability runtime, task-run recorder, and audit evidence owners",
		},
		{
			RetiredPythonModule: "src/bigclaw/reports.py",
			ReplacementKind:     "go-reporting-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/reporting/reporting.go",
				"bigclaw-go/internal/reportstudio/reportstudio.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/reporting/reporting_test.go",
				"bigclaw-go/internal/reportstudio/reportstudio_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python reports surface is replaced by the Go reporting builders and report studio publishing surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/evaluation.py",
			ReplacementKind:     "go-evaluation-benchmark",
			GoReplacements: []string{
				"bigclaw-go/internal/evaluation/evaluation.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/evaluation/evaluation_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python evaluation module is replaced by the Go benchmark, replay, and scorecard evaluation surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/operations.py",
			ReplacementKind:     "go-operations-control-plane",
			GoReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
				"bigclaw-go/internal/contract/execution.go",
				"bigclaw-go/internal/control/controller.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/product/dashboard_run_contract_test.go",
				"bigclaw-go/internal/contract/execution_test.go",
				"bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md",
			},
			Status: "retired Python operations surface is replaced by the Go dashboard contract, execution contract, and control-plane handlers",
		},
	}
}
