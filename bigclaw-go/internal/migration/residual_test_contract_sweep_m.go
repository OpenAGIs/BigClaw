package migration

func ResidualTestContractSweepMReplacements() []LegacyTestContractReplacement {
	return []LegacyTestContractReplacement{
		{
			RetiredPythonTest: "tests/test_design_system.py",
			ReplacementKind:   "go-design-system-contract",
			GoReplacements: []string{
				"bigclaw-go/internal/designsystem/designsystem.go",
				"bigclaw-go/internal/product/console.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/designsystem/designsystem_test.go",
				"bigclaw-go/internal/product/console_test.go",
				"docs/issue-plan.md",
			},
			Status: "retired Python design-system contract is replaced by the Go-native component-library audit and product console design-system surfaces",
		},
		{
			RetiredPythonTest: "tests/test_dsl.py",
			ReplacementKind:   "go-workflow-definition-contract",
			GoReplacements: []string{
				"bigclaw-go/internal/workflow/definition.go",
				"bigclaw-go/internal/workflow/closeout.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/workflow/definition_test.go",
				"docs/go-domain-intake-parity-matrix.md",
				"docs/issue-plan.md",
			},
			Status: "retired Python DSL contract is replaced by the Go workflow definition parser and closeout orchestration surfaces",
		},
		{
			RetiredPythonTest: "tests/test_evaluation.py",
			ReplacementKind:   "go-evaluation-replay-contract",
			GoReplacements: []string{
				"bigclaw-go/internal/evaluation/evaluation.go",
				"bigclaw-go/internal/planning/planning.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/evaluation/evaluation_test.go",
				"bigclaw-go/internal/planning/planning_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python evaluation contract is replaced by the Go evaluation runner, replay reporting, and planning-owned benchmark evidence surfaces",
		},
	}
}
