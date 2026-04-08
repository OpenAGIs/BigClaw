package migration

func LegacyPolicyGovernanceModuleReplacements() []LegacyModuleReplacement {
	return []LegacyModuleReplacement{
		{
			RetiredPythonModule: "src/bigclaw/risk.py",
			ReplacementKind:     "go-risk-policy-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/risk/risk.go",
				"bigclaw-go/internal/risk/assessment.go",
				"bigclaw-go/internal/policy/policy.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/risk/risk_test.go",
				"bigclaw-go/internal/risk/assessment_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python risk surface is replaced by the Go risk scorer, assessment model, and policy lane resolution surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/governance.py",
			ReplacementKind:     "go-governance-freeze",
			GoReplacements: []string{
				"bigclaw-go/internal/governance/freeze.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/governance/freeze_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python governance surface is replaced by the Go scope-freeze backlog board and governance audit surface",
		},
		{
			RetiredPythonModule: "src/bigclaw/execution_contract.py",
			ReplacementKind:     "go-execution-contract",
			GoReplacements: []string{
				"bigclaw-go/internal/contract/execution.go",
				"bigclaw-go/internal/api/policy_runtime.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/contract/execution_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python execution-contract surface is replaced by the Go execution contract, permission matrix, and policy runtime handlers",
		},
		{
			RetiredPythonModule: "src/bigclaw/audit_events.py",
			ReplacementKind:     "go-audit-spec-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/observability/audit.go",
				"bigclaw-go/internal/observability/audit_spec.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/observability/audit_test.go",
				"bigclaw-go/internal/observability/audit_spec_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python audit-events surface is replaced by the Go audit event registry and observability audit surface",
		},
	}
}
