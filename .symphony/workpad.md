Issue: BIG-GO-1027

Plan
- Finish a narrow residual tranche by replacing `tests/test_runtime_matrix.py` with Go-native runtime-compat coverage.
- Add a small `internal/runtimecompat` package that preserves the frozen Python tool-runtime and worker lifecycle contract, while reusing the existing scheduler legacy-medium helper for the sandbox-medium expectation.
- Delete `tests/test_runtime_matrix.py` once equivalent Go-native worker lifecycle, tool policy/audit chain, and legacy medium-routing behaviors are covered.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the runtime-matrix residual Python test tranche and corresponding Go compatibility coverage.
- Repository `.py` file count decreases after removing `tests/test_runtime_matrix.py`.
- Targeted Go tests cover multi-tool worker lifecycle completion, legacy medium routing expectations, and tool policy blocking/success audit chains.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `gofmt -w bigclaw-go/internal/runtimecompat/runtime.go bigclaw-go/internal/runtimecompat/runtime_test.go`
- `cd bigclaw-go && go test ./internal/runtimecompat -run 'Test(RuntimeExecutesMultipleToolsAndCompletesLifecycle|ToolRuntimePolicyAndAuditChain)' -count=1`
- `cd bigclaw-go && go test ./internal/scheduler -run 'TestLegacyMediumDecision' -count=1`
- `git diff --stat`
- `git status --short`
