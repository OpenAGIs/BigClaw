package migration

func LegacyTestContractSweepXReplacements() []LegacyTestContractReplacement {
	return []LegacyTestContractReplacement{
		{
			RetiredPythonTest: "tests/test_audit_events.py",
			ReplacementKind:   "go-audit-event-spec-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/observability/audit_spec.go",
				"bigclaw-go/internal/observability/audit.go",
				"bigclaw-go/internal/observability/audit_spec_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/observability/audit_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
				"reports/OPE-134-validation.md",
			},
			Status: "retired Python audit-events coverage is replaced by the Go audit-event specification and observability validation surface",
		},
		{
			RetiredPythonTest: "tests/test_connectors.py",
			ReplacementKind:   "go-intake-connector-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/intake/connector.go",
				"bigclaw-go/internal/intake/types.go",
				"bigclaw-go/internal/api/v2.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/intake/connector_test.go",
				"docs/go-domain-intake-parity-matrix.md",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python connectors coverage is replaced by the Go intake connector and v2 intake API surface",
		},
		{
			RetiredPythonTest: "tests/test_console_ia.py",
			ReplacementKind:   "go-console-ia-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/consoleia/consoleia.go",
				"bigclaw-go/internal/product/console.go",
				"bigclaw-go/internal/designsystem/designsystem.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/consoleia/consoleia_test.go",
				"bigclaw-go/internal/product/console_test.go",
				"reports/OPE-127-validation.md",
			},
			Status: "retired Python console IA coverage is replaced by the Go console IA contract and product console surface",
		},
		{
			RetiredPythonTest: "tests/test_dashboard_run_contract.py",
			ReplacementKind:   "go-dashboard-run-contract-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/product/dashboard_run_contract.go",
				"bigclaw-go/internal/contract/execution.go",
				"bigclaw-go/internal/api/server.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/product/dashboard_run_contract_test.go",
				"bigclaw-go/internal/contract/execution_test.go",
				"reports/OPE-129-validation.md",
			},
			Status: "retired Python dashboard/run contract coverage is replaced by the Go dashboard/run contract and execution contract surfaces",
		},
	}
}
