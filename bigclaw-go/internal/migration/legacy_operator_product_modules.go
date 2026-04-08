package migration

func LegacyOperatorProductModuleReplacements() []LegacyModuleReplacement {
	return []LegacyModuleReplacement{
		{
			RetiredPythonModule: "src/bigclaw/issue_archive.py",
			ReplacementKind:     "go-issue-archive-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/issuearchive/archive.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/issuearchive/archive_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python issue-archive surface is replaced by the Go issue priority archive and audit surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/run_detail.py",
			ReplacementKind:     "go-run-detail-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/observability/task_run.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/observability/task_run_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python run-detail surface is replaced by the Go task-run detail and replay evidence surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/dashboard_run_contract.py",
			ReplacementKind:     "go-dashboard-contract",
			GoReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/product/dashboard_run_contract_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python dashboard-run contract is replaced by the Go dashboard and run-detail contract surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/saved_views.py",
			ReplacementKind:     "go-saved-views-catalog",
			GoReplacements: []string{
				"bigclaw-go/internal/product/saved_views.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/product/saved_views_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python saved-views surface is replaced by the Go saved-view catalog and digest subscription surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/console_ia.py",
			ReplacementKind:     "go-console-ia-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/consoleia/consoleia.go",
				"bigclaw-go/internal/product/console.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/consoleia/consoleia_test.go",
				"bigclaw-go/internal/product/console_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python console-ia surface is replaced by the Go console interaction architecture and operator console surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/design_system.py",
			ReplacementKind:     "go-design-system-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/designsystem/designsystem.go",
				"bigclaw-go/internal/product/console.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/designsystem/designsystem_test.go",
				"bigclaw-go/internal/product/console_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python design-system surface is replaced by the Go design token library and operator console surface contracts",
		},
	}
}
