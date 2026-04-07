package migration

type LegacyTestContractReplacement struct {
	RetiredPythonTest string
	ReplacementKind   string
	GoReplacements    []string
	EvidencePaths     []string
	Status            string
}

func LegacyTestContractSweepBReplacements() []LegacyTestContractReplacement {
	return []LegacyTestContractReplacement{
		{
			RetiredPythonTest: "tests/test_control_center.py",
			ReplacementKind:   "go-control-plane-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/control/controller.go",
				"bigclaw-go/internal/api/server.go",
				"bigclaw-go/internal/api/v2.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/control/controller_test.go",
				"bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md",
			},
			Status: "retired Python control-center contract is owned by the Go control plane and v2 operations surface",
		},
		{
			RetiredPythonTest: "tests/test_operations.py",
			ReplacementKind:   "go-operations-contract-split",
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
			Status: "retired Python operations contract is replaced by Go-owned dashboard, execution, and control-center contract surfaces",
		},
		{
			RetiredPythonTest: "tests/test_ui_review.py",
			ReplacementKind:   "go-review-pack-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/uireview/uireview.go",
				"bigclaw-go/internal/uireview/builder.go",
				"bigclaw-go/internal/uireview/render.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/uireview/uireview_test.go",
				"docs/issue-plan.md",
				"reports/OPE-128-validation.md",
			},
			Status: "retired Python UI review contract is replaced by the Go-native review-pack builder, auditor, and renderer",
		},
	}
}
