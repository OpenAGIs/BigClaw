# BIG-GO-1087 Workpad

## Plan
- Confirm every repo path that still treats `scripts/ops/symphony_workspace_validate.py` as a primary entrypoint.
- Delete `scripts/ops/symphony_workspace_validate.py` so the repository `.py` count drops for this slice.
- Update the remaining scoped docs and regression coverage to point at `bash scripts/ops/bigclawctl workspace validate` as the default validation entry.
- Run targeted file-count, Go test, and entrypoint validation commands, then commit and push the branch.

## Acceptance
- `scripts/ops/symphony_workspace_validate.py` is deleted from the repository.
- No scoped default execution path for workspace validation in this slice still points at the deleted Python wrapper.
- Targeted regression coverage still verifies legacy workspace validate flag translation through the Go code.
- Repository `.py` count is lower after the change than before it.
- The change set remains scoped to `BIG-GO-1087`.

## Validation
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `git status --short`

## Validation Results
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l` -> before deletion `23`, after deletion `22`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl` -> `ok  	bigclaw-go/internal/legacyshim	0.413s`; `ok  	bigclaw-go/cmd/bigclawctl	3.219s`
- `bash scripts/ops/bigclawctl workspace validate --help` -> passed and printed `usage: bigclawctl workspace validate [flags]`
- `git status --short` -> modified `.symphony/workpad.md`, modified `docs/go-cli-script-migration-plan.md`, deleted `scripts/ops/symphony_workspace_validate.py`
