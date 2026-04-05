package migration

type LegacyContractTestReplacement struct {
	RetiredPythonTest  string
	ReplacementKind    string
	NativeReplacements []string
	EvidencePaths      []string
	Status             string
}

func LegacyContractTestReplacementsSweepA() []LegacyContractTestReplacement {
	return []LegacyContractTestReplacement{
		{
			RetiredPythonTest: "tests/test_execution_contract.py",
			ReplacementKind:   "go-contract-owner",
			NativeReplacements: []string{
				"bigclaw-go/internal/contract/execution.go",
				"bigclaw-go/internal/contract/execution_test.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"reports/OPE-131-validation.md",
			},
			Status: "retired Python execution contract test replaced by the Go execution contract owner and regression coverage",
		},
		{
			RetiredPythonTest: "tests/test_dashboard_run_contract.py",
			ReplacementKind:   "go-contract-owner",
			NativeReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
				"bigclaw-go/internal/product/dashboard_run_contract_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/api/expansion_test.go",
				"reports/OPE-129-validation.md",
			},
			Status: "retired Python dashboard/run contract test replaced by the Go product contract package and API expansion coverage",
		},
		{
			RetiredPythonTest: "tests/test_cross_process_coordination_surface.py",
			ReplacementKind:   "go-api-surface",
			NativeReplacements: []string{
				"bigclaw-go/internal/api/coordination_surface.go",
				"bigclaw-go/internal/regression/coordination_contract_surface_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/e2e-validation.md",
				"bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json",
			},
			Status: "retired Python coordination surface test replaced by the Go API surface loader and regression assertions over the checked-in contract payload",
		},
		{
			RetiredPythonTest: "tests/test_followup_digests.py",
			ReplacementKind:   "repo-native-report-guard",
			NativeReplacements: []string{
				"bigclaw-go/docs/reports/parallel-follow-up-index.md",
				"bigclaw-go/internal/regression/followup_index_docs_test.go",
			},
			EvidencePaths: []string{
				"docs/parallel-refill-queue.md",
				"reports/OPE-270-271-validation.md",
			},
			Status: "retired Python follow-up digest test replaced by the repo-native parallel follow-up index and Go regression coverage",
		},
		{
			RetiredPythonTest: "tests/test_parallel_refill.py",
			ReplacementKind:   "go-queue-and-native-docs",
			NativeReplacements: []string{
				"bigclaw-go/internal/refill/queue_test.go",
				"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
			},
			EvidencePaths: []string{
				"docs/parallel-refill-queue.json",
				"reports/OPE-270-271-validation.md",
			},
			Status: "retired Python parallel refill test replaced by the Go refill queue coverage and repo-native validation matrix checks",
		},
	}
}
