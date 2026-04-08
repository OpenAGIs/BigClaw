package migration

func LegacyTestContractSweepDReplacements() []LegacyTestContractReplacement {
	return []LegacyTestContractReplacement{
		{
			RetiredPythonTest: "tests/test_design_system.py",
			ReplacementKind:   "go-design-system-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/designsystem/designsystem.go",
				"bigclaw-go/internal/designsystem/designsystem_test.go",
				"bigclaw-go/internal/api/expansion_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/product/console_test.go",
				"reports/OPE-92-validation.md",
			},
			Status: "retired Python design-system coverage is replaced by Go-owned designsystem and expansion contract surfaces",
		},
		{
			RetiredPythonTest: "tests/test_dsl.py",
			ReplacementKind:   "go-workflow-definition-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/workflow/definition.go",
				"bigclaw-go/internal/workflow/definition_test.go",
				"bigclaw-go/internal/api/expansion.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/api/expansion_test.go",
				"bigclaw-go/cmd/bigclawctl/migration_commands.go",
			},
			Status: "retired Python DSL coverage is replaced by the Go workflow-definition parser and render API surface",
		},
		{
			RetiredPythonTest: "tests/test_evaluation.py",
			ReplacementKind:   "go-evaluation-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/evaluation/evaluation.go",
				"bigclaw-go/internal/evaluation/evaluation_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/planning/planning.go",
				"bigclaw-go/internal/planning/planning_test.go",
			},
			Status: "retired Python evaluation coverage is replaced by the Go evaluation and planning evidence surface",
		},
		{
			RetiredPythonTest: "tests/test_parallel_validation_bundle.py",
			ReplacementKind:   "go-validation-bundle-continuation-surface",
			GoReplacements: []string{
				"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
				"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
				"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
				"bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json",
				"bigclaw-go/docs/reports/shared-queue-companion-summary.json",
			},
			Status: "retired Python validation-bundle coverage is replaced by the Go automation bundle command and checked-in continuation scorecard fixtures",
		},
	}
}
