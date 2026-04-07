package migration

type LegacyModuleReplacement struct {
	RetiredPythonModule string
	ReplacementKind     string
	GoReplacements      []string
	EvidencePaths       []string
	Status              string
}

func LegacyModelRuntimeModuleReplacements() []LegacyModuleReplacement {
	return []LegacyModuleReplacement{
		{
			RetiredPythonModule: "src/bigclaw/models.py",
			ReplacementKind:     "go-package-split",
			GoReplacements: []string{
				"bigclaw-go/internal/domain/task.go",
				"bigclaw-go/internal/domain/priority.go",
				"bigclaw-go/internal/risk/assessment.go",
				"bigclaw-go/internal/triage/record.go",
				"bigclaw-go/internal/billing/statement.go",
				"bigclaw-go/internal/workflow/model.go",
			},
			EvidencePaths: []string{
				"docs/go-domain-intake-parity-matrix.md",
				"bigclaw-go/internal/workflow/model_test.go",
			},
			Status: "retired Python model contract replaced by split Go domain, risk, triage, billing, and workflow owners",
		},
		{
			RetiredPythonModule: "src/bigclaw/runtime.py",
			ReplacementKind:     "go-runtime-mainline",
			GoReplacements: []string{
				"bigclaw-go/internal/worker/runtime.go",
				"bigclaw-go/internal/worker/runtime_runonce.go",
				"bigclaw-go/internal/worker/runtime_test.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/docs/reports/worker-lifecycle-validation-report.md",
			},
			Status: "retired Python runtime module replaced by the Go worker runtime mainline and lifecycle evidence",
		},
	}
}
