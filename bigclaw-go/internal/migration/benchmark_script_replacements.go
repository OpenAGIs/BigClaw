package migration

type BenchmarkScriptReplacement struct {
	RetiredPythonPath string
	ReplacementKind   string
	ActivePaths       []string
	EvidencePaths     []string
	Status            string
}

func BenchmarkScriptReplacements() []BenchmarkScriptReplacement {
	return []BenchmarkScriptReplacement{
		{
			RetiredPythonPath: "bigclaw-go/scripts/benchmark/soak_local.py",
			ReplacementKind:   "go-cli-subcommand",
			ActivePaths: []string{
				"bigclaw-go/cmd/bigclawctl/automation_commands.go",
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
				"bigclaw-go/scripts/benchmark/run_suite.sh",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/benchmark-plan.md",
				"bigclaw-go/docs/reports/long-duration-soak-report.md",
			},
			Status: "retired benchmark soak helper replaced by the Go-native automation benchmark soak-local flow",
		},
		{
			RetiredPythonPath: "bigclaw-go/scripts/benchmark/run_matrix.py",
			ReplacementKind:   "go-cli-subcommand",
			ActivePaths: []string{
				"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
				"bigclaw-go/scripts/benchmark/run_suite.sh",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/benchmark-plan.md",
				"bigclaw-go/docs/reports/benchmark-matrix-report.json",
			},
			Status: "retired benchmark matrix helper replaced by the Go-native automation benchmark run-matrix flow",
		},
		{
			RetiredPythonPath: "bigclaw-go/scripts/benchmark/capacity_certification.py",
			ReplacementKind:   "go-cli-subcommand",
			ActivePaths: []string{
				"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/go-cli-script-migration.md",
				"bigclaw-go/docs/reports/capacity-certification-matrix.json",
				"bigclaw-go/docs/reports/capacity-certification-report.md",
			},
			Status: "retired capacity certification helper replaced by the Go-native automation benchmark capacity-certification flow",
		},
		{
			RetiredPythonPath: "bigclaw-go/scripts/benchmark/capacity_certification_test.py",
			ReplacementKind:   "go-test-coverage",
			ActivePaths: []string{
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
				"bigclaw-go/internal/regression/big_go_1160_script_migration_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/go-cli-script-migration.md",
			},
			Status: "retired Python-side benchmark test coverage replaced by Go command and regression tests",
		},
	}
}
