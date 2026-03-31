Issue: BIG-GO-1027

Plan
- Finish a narrow residual tranche by replacing `tests/test_dsl.py` with Go-native workflow definition validation and runner coverage.
- Add a lightweight `internal/workflow` definition runner that validates step kinds, renders report/journal paths, writes artifacts, and reuses the existing acceptance-gate semantics.
- Delete `tests/test_dsl.py` once equivalent Go-native definition parsing, invalid-step rejection, end-to-end artifact writing, and manual-approval closure behaviors are covered.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the DSL residual Python test tranche and corresponding Go workflow coverage.
- Repository `.py` file count decreases after removing `tests/test_dsl.py`.
- Targeted Go tests cover workflow definition parsing/rendering, invalid-step rejection, end-to-end report/journal artifact writing, and manual-approval acceptance closure.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `gofmt -w bigclaw-go/internal/workflow/definition.go bigclaw-go/internal/workflow/runner.go bigclaw-go/internal/workflow/definition_test.go bigclaw-go/internal/workflow/runner_test.go`
- `cd bigclaw-go && go test ./internal/workflow -run 'Test(DefinitionParsesAndRendersTemplates|DefinitionValidateRejectsUnknownStepKinds|RunDefinitionWritesArtifactsAndAcceptsCompleteEvidence|RunDefinitionManualApprovalClosesHighRiskTask)' -count=1`
- `git diff --stat`
- `git status --short`
