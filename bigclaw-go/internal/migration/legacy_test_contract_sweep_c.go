package migration

func LegacyTestContractSweepCReplacements() []LegacyTestContractReplacement {
	return []LegacyTestContractReplacement{
		{
			RetiredPythonTest: "tests/test_issue_archive.py",
			ReplacementKind:   "go-issue-priority-archive-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/issuearchive/archive.go",
				"bigclaw-go/internal/issuearchive/archive_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/reports/big-go-1596-go-only-sweep-refill.md",
				"reports/BIG-GO-948-validation.md",
			},
			Status: "retired Python issue archive coverage is replaced by the Go issue-priority archive manifest, audit, and reporting surface",
		},
		{
			RetiredPythonTest: "tests/test_pilot.py",
			ReplacementKind:   "go-pilot-readiness-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/pilot/report.go",
				"bigclaw-go/internal/pilot/report_test.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/docs/reports/big-go-1597-python-asset-sweep.md",
				"reports/BIG-GO-948-validation.md",
			},
			Status: "retired Python pilot coverage is replaced by the Go pilot readiness report and implementation-result surface",
		},
	}
}
