package migration

func LegacyCollaborationIntakeModuleReplacements() []LegacyModuleReplacement {
	return []LegacyModuleReplacement{
		{
			RetiredPythonModule: "src/bigclaw/collaboration.py",
			ReplacementKind:     "go-collaboration-thread",
			GoReplacements: []string{
				"bigclaw-go/internal/collaboration/thread.go",
				"bigclaw-go/internal/flow/flow.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/collaboration/thread_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python collaboration surface is replaced by the Go collaboration thread and flow coordination surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/connectors.py",
			ReplacementKind:     "go-intake-connectors",
			GoReplacements: []string{
				"bigclaw-go/internal/intake/connector.go",
				"bigclaw-go/internal/intake/types.go",
				"bigclaw-go/internal/prd/intake.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/intake/connector_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python connectors surface is replaced by the Go intake connector registry and canonical intake types",
		},
		{
			RetiredPythonModule: "src/bigclaw/dsl.py",
			ReplacementKind:     "go-workflow-definition",
			GoReplacements: []string{
				"bigclaw-go/internal/workflow/definition.go",
				"bigclaw-go/internal/prd/intake.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/workflow/definition_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python DSL surface is replaced by the Go workflow-definition renderer and intake PRD vocabulary",
		},
		{
			RetiredPythonModule: "src/bigclaw/planning.py",
			ReplacementKind:     "go-planning-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/planning/planning.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/planning/planning_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python planning surface is replaced by the Go planning release-candidate and evidence synthesis surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/pilot.py",
			ReplacementKind:     "go-pilot-rollout",
			GoReplacements: []string{
				"bigclaw-go/internal/pilot/report.go",
				"bigclaw-go/internal/pilot/rollout.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/pilot/report_test.go",
				"bigclaw-go/internal/pilot/rollout_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python pilot surface is replaced by the Go pilot report and rollout readiness surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/ui_review.py",
			ReplacementKind:     "go-ui-review-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/uireview/uireview.go",
				"bigclaw-go/internal/uireview/builder.go",
				"bigclaw-go/internal/uireview/render.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/uireview/uireview_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python UI-review surface is replaced by the Go review-pack builder, renderer, and audit surface",
		},
	}
}
