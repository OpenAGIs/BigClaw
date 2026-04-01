## BIG-GO-1046

### Plan
- Inspect Python entrypoints under `scripts/ops` and map each one to its existing Go-backed `bigclawctl` command.
- Delete the Python shims in `scripts/ops` and replace them with non-Python entrypoints that preserve current invocation behavior.
- Add or update targeted Go tests for the compatibility surface where behavior is not already covered.
- Run targeted validation, confirm the repository Python-file count drops, then commit and push the scoped change set.

### Acceptance
- Python files under `scripts/ops` that act as automation/workspace entrypoints are deleted.
- Replacement entrypoints route to Go-backed commands and preserve the required arguments and defaults.
- Any new compatibility behavior is covered by targeted Go tests.
- `find . -name "*.py" | wc -l` is lower than the starting count of `67`.
- Final commit message/body lists deleted Python files and added Go files or Go tests.

### Validation
- `go test ./bigclaw-go/cmd/bigclawctl -run '...'`
- Execute replacement entrypoints with `--help` or equivalent smoke checks where safe.
- `find . -name "*.py" | wc -l`
- `git status --short`

### Validation Results
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1046/bigclaw-go && go test ./cmd/bigclawctl -run 'TestOpsWrapperScriptsDispatchExpectedArgs|TestRunWorkspaceValidateJSONOutputDoesNotEscapeArrowTokens|TestRunWorkspaceBootstrapJSONOutputDoesNotEscapeArrowTokens|TestRunGitHubSyncInstallJSONOutputDoesNotEscapeArrowTokens|TestRunRefillHelpPrintsDefaultsAndExitsZero'`
  - `ok  	bigclaw-go/cmd/bigclawctl	1.342s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1046 && bash scripts/ops/bigclaw-github-sync --help >/tmp/bigclaw-github-sync.help && bash scripts/ops/bigclaw-refill-queue --help >/tmp/bigclaw-refill-queue.help && bash scripts/ops/bigclaw-workspace-bootstrap --help >/tmp/bigclaw-workspace-bootstrap.help && bash scripts/ops/symphony-workspace-bootstrap --help >/tmp/symphony-workspace-bootstrap.help && bash scripts/ops/symphony-workspace-validate --help >/tmp/symphony-workspace-validate.help && printf 'wrapper smoke checks passed\n'`
  - `wrapper smoke checks passed`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1046 && find . -name '*.py' | wc -l`
  - `62`
