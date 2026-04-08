package migration

func LegacyTestContractSweepZReplacements() []LegacyTestContractReplacement {
	return []LegacyTestContractReplacement{
		{
			RetiredPythonTest: "tests/test_repo_collaboration.py",
			ReplacementKind:   "go-repo-collaboration-thread-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/collaboration/thread.go",
				"bigclaw-go/internal/collaboration/thread_test.go",
				"bigclaw-go/internal/repo/board.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/repo/repo_surfaces_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
				"docs/go-mainline-cutover-handoff.md",
			},
			Status: "retired Python repo collaboration coverage is replaced by the Go-native collaboration thread merge surface and repo discussion board contract",
		},
		{
			RetiredPythonTest: "tests/test_repo_gateway.py",
			ReplacementKind:   "go-repo-gateway-normalization-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/repo/gateway.go",
				"bigclaw-go/internal/repo/repo_surfaces_test.go",
				"bigclaw-go/internal/repo/commits.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"docs/go-mainline-cutover-handoff.md",
			},
			Status: "retired Python repo gateway coverage is replaced by the Go repo gateway normalization and commit surface",
		},
		{
			RetiredPythonTest: "tests/test_repo_governance.py",
			ReplacementKind:   "go-repo-governance-contract-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/repo/governance.go",
				"bigclaw-go/internal/repo/governance_test.go",
				"bigclaw-go/internal/repo/plane.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"docs/go-mainline-cutover-handoff.md",
			},
			Status: "retired Python repo governance coverage is replaced by the Go permission contract and repo control-plane surface",
		},
		{
			RetiredPythonTest: "tests/test_repo_registry.py",
			ReplacementKind:   "go-repo-registry-routing-surface",
			GoReplacements: []string{
				"bigclaw-go/internal/repo/registry.go",
				"bigclaw-go/internal/repo/repo_surfaces_test.go",
				"bigclaw-go/internal/repo/links.go",
			},
			EvidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"docs/go-mainline-cutover-handoff.md",
			},
			Status: "retired Python repo registry coverage is replaced by the Go repo registry, channel routing, and run-commit link surface",
		},
	}
}
