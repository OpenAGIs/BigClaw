## Codex Workpad

### Issue

- `LOCAL-1`
- `BIG-vNext-Go-001 全仓库推进到100% Go语言实现`

### Plan

- [x] Audit the repo state and existing Go-mainline cutover artifacts to identify the remaining Python and non-Go runtime surface.
- [x] Add a Go-owned migration inventory/planning surface under `bigclaw-go` so the repo can generate a deterministic Go-only migration plan and inventory artifact.
- [x] Generate repo-native migration outputs that include acceptance scope, runtime/script/test/toolchain inventory, at least 10 parallel Symphony slices, and branch/PR/validation strategy.
- [x] Seed the first migration batch in the local tracker so the parallel plan is executable instead of documentation-only.
- [x] Run targeted validation and record the exact commands/results for closeout.
- [x] Commit and push the issue branch to `origin`.
- [x] Advance the first executable migration batch by folding legacy workspace wrapper semantics into the Go CLI and shrinking the shell compatibility layer.

### Acceptance Criteria

- [x] A repo-native Go-only migration plan exists and clearly defines phased cutover from mixed-language to Go-only development.
- [x] The current Python and non-Go runtime/script/test/control/toolchain surface is inventoried in a generated artifact.
- [x] At least 10 parallelizable migration slices are defined with explicit scope and sequencing guidance for Symphony.
- [x] Branch, PR, and validation strategy is documented and tied to the generated plan.
- [x] The first batch of migration slices is created in the local tracker for immediate parallel execution.

### Validation

- [x] `cd bigclaw-go && go test ./internal/migration ./cmd/bigclawctl`
- [x] `cd bigclaw-go && go run ./cmd/bigclawctl go-migration plan --repo .. --json-out ../docs/reports/go-only-migration-inventory.json --md-out ../docs/go-only-migration-plan.md`
- [x] `cd bigclaw-go && go test ./internal/regression`
- [x] `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestRunWorkspace(Bootstrap|Validate)'`
- [x] `cd bigclaw-go && go test ./cmd/bigclawctl`
- [x] `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestRunLocalIssues(StateAliasSupportsPositionalArguments|SetState|Comment)'`
- [x] `bash scripts/ops/bigclaw-issue state --help`
- [x] `bash scripts/ops/symphony_workspace_bootstrap.py --help`
- [x] `bash scripts/ops/bigclaw_workspace_bootstrap.py --help`
- [x] `cd bigclaw-go && go test ./internal/refill ./cmd/bigclawctl`
- [x] `rg -n "bigclaw-issue|bigclawctl local-issues" README.md docs/parallel-refill-queue.md bigclaw-go/internal/refill/queue_markdown.go`
- [x] `cd bigclaw-go && go run ./cmd/bigclawctl --help`

### Notes

- The repo already merged the earlier Go-mainline cutover, but root-level Python code and non-Go operators remain as migration-only or still-active surfaces. This issue is therefore focused on the next-stage retirement/execution plan rather than redoing the completed cutover.
- Keep changes scoped to migration planning, inventory, tracker seeding, and the first executable Go-owned planning slice.
- Continuation focus: `BIG-VNEXT-GO-104` first-batch progress by migrating `scripts/ops/*workspace*` wrapper behavior into `bigclaw-go/cmd/bigclawctl`.
- Current first-batch progress: `bigclawctl workspace bootstrap` now applies the historical repo/cache-key defaults directly in Go, `workspace validate` now accepts the legacy wrapper flag forms (`--issues` list, `--report-file`, `--no-cleanup`), and `scripts/ops/symphony_workspace_validate.py` no longer carries local argument-translation logic.
- Latest continuation progress: `bigclawctl local-issues` now accepts the historical `state` alias plus positional `state/comment` arguments, allowing `scripts/ops/bigclaw-issue` to drop its local tracker argument-rewrite logic and forward directly to the Go CLI.
- Latest wrapper alignment: `scripts/ops/symphony_workspace_bootstrap.py` now matches `scripts/ops/bigclaw_workspace_bootstrap.py` by forwarding directly to `workspace bootstrap`, eliminating the last workspace-wrapper command-shape mismatch.
- Latest operator-surface shift: README, the refill queue markdown, and the queue markdown generator now recommend `bash scripts/ops/bigclawctl local-issues ...` as the primary tracker CLI, with `bigclaw-issue` reduced to a compatibility alias instead of the default operator entrypoint.
- Latest CLI visibility fix: the `bigclawctl` root help now exposes `go-migration`, and repo metadata/docs now describe the remaining Python-named ops wrappers as compatibility shims over the Go CLI rather than active Python tooling.
