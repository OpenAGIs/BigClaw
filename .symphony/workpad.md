## BIG-GO-909 Workpad

### Scope

- Migrate and document the Go-side ownership for `github sync`, `repo links`, `repo registry`, and `repo collaboration`.
- Keep the slice bounded to the existing repo/go migration lane under `bigclaw-go/internal/repo`, `bigclaw-go/internal/githubsync`, and issue-specific docs.

### Plan

- Audit Python-to-Go parity for the four targeted repo surfaces and identify the missing production/test/documentation pieces.
- Implement the missing Go repo collaboration primitives needed to match the existing Python behavior without broadening the API surface.
- Extend targeted Go tests to cover the migrated collaboration path alongside the existing GitHub sync, repo links, and repo registry coverage.
- Record the migration plan, first-batch implementation checklist, regression surface, validation commands, branch/PR suggestion, and risks in a dedicated issue report.
- Commit the issue-scoped changes and push a dedicated remote branch for `BIG-GO-909`.

### Acceptance

- Go has a concrete migration outcome for all four targeted surfaces: `github sync`, `repo links`, `repo registry`, and `repo collaboration`.
- The repo collaboration gap is implemented in Go with targeted regression coverage.
- The repository contains an executable migration note listing the current implementation state, first-batch follow-ups, validation commands, regression surface, branch/PR guidance, and risks.
- Targeted tests run successfully and their exact commands/results are recorded in the issue report.

### Validation

- `cd bigclaw-go && go test ./internal/githubsync ./internal/repo`

### Regression Surface

- Git repository inspection / auto-push behavior in `internal/githubsync`.
- Run commit link validation and accepted commit selection in `internal/repo/links.go`.
- Repo space / agent resolution in `internal/repo/registry.go`.
- Repo-board to collaboration-thread conversion and merged collaboration rendering in `internal/repo`.

### Notes

- This issue is a bounded migration subtask. Avoid unrelated CLI, API, workflow, or control-plane rewrites unless a targeted test proves they are required for the four named surfaces.
