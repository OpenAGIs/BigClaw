package migration

func LegacyBootstrapSyncModuleReplacements() []LegacyModuleReplacement {
	return []LegacyModuleReplacement{
		{
			RetiredPythonModule: "src/bigclaw/github_sync.py",
			ReplacementKind:     "go-github-sync",
			GoReplacements: []string{
				"bigclaw-go/internal/githubsync/sync.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/githubsync/sync_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python github-sync surface is replaced by the Go GitHub sync install, inspect, and push guarantees",
		},
		{
			RetiredPythonModule: "src/bigclaw/workspace_bootstrap.py",
			ReplacementKind:     "go-workspace-bootstrap",
			GoReplacements: []string{
				"bigclaw-go/internal/bootstrap/bootstrap.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/bootstrap/bootstrap_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python workspace-bootstrap surface is replaced by the Go bootstrap engine and bigclawctl entrypoint",
		},
		{
			RetiredPythonModule: "src/bigclaw/workspace_bootstrap_cli.py",
			ReplacementKind:     "go-bootstrap-cli",
			GoReplacements: []string{
				"bigclaw-go/internal/bootstrap/bootstrap.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/bootstrap/bootstrap_test.go",
				"bigclaw-go/cmd/bigclawctl/main_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python workspace-bootstrap CLI is replaced by the Go bootstrap engine and bigclawctl operator entrypoint",
		},
		{
			RetiredPythonModule: "src/bigclaw/workspace_bootstrap_validation.py",
			ReplacementKind:     "go-bootstrap-validation",
			GoReplacements: []string{
				"bigclaw-go/internal/bootstrap/bootstrap.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/bootstrap/bootstrap_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python workspace-bootstrap validation surface is replaced by the Go bootstrap validation and cleanup flow",
		},
		{
			RetiredPythonModule: "src/bigclaw/parallel_refill.py",
			ReplacementKind:     "go-refill-queue",
			GoReplacements: []string{
				"bigclaw-go/internal/refill/queue.go",
				"bigclaw-go/internal/refill/local_store.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/internal/refill/queue_test.go",
				"bigclaw-go/internal/refill/local_store_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python parallel-refill surface is replaced by the Go refill queue, tracker-neutral local store, and bigclawctl refill entrypoint",
		},
		{
			RetiredPythonModule: "src/bigclaw/service.py",
			ReplacementKind:     "go-mainline-service",
			GoReplacements: []string{
				"bigclaw-go/cmd/bigclawd/main.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/cmd/bigclawd/main_test.go",
				"bigclaw-go/cmd/bigclawctl/main_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python service entrypoint is replaced by the Go bigclawd daemon and bigclawctl control entrypoints",
		},
		{
			RetiredPythonModule: "src/bigclaw/__main__.py",
			ReplacementKind:     "go-mainline-entrypoint",
			GoReplacements: []string{
				"bigclaw-go/cmd/bigclawd/main.go",
				"bigclaw-go/cmd/bigclawctl/main.go",
			},
			EvidencePaths: []string{
				"bigclaw-go/cmd/bigclawd/main_test.go",
				"bigclaw-go/cmd/bigclawctl/main_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
			Status: "retired Python __main__ entrypoint is replaced by the Go daemon and operator CLIs",
		},
	}
}
