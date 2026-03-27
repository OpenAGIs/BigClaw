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
- [x] Start `BIG-VNEXT-GO-107` by replacing the legacy Python dev smoke helper with a Go-native `bigclawctl` command and updated docs.
- [x] Replace `scripts/create_issues.py` with a Go-native `bigclawctl issue-bootstrap sync` flow and leave the Python path as a compatibility shim.
- [x] Replace `scripts/dev_bootstrap.sh` with a Go-native `bigclawctl dev-bootstrap` flow and leave the shell path as a compatibility shim.

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
- [x] `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestRunDevSmokeJSON|TestPrintRootUsageIncludesGoMigration'`
- [x] `bash scripts/ops/bigclawctl dev-smoke --json`
- [x] `cd bigclaw-go && go test ./internal/issuebootstrap ./cmd/bigclawctl`
- [x] `bash scripts/create_issues.py v1 --dry-run --json`
- [x] `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestRunDevBootstrapJSON|TestRunDevSmokeJSON|TestPrintRootUsageIncludesGoMigration'`
- [x] `bash scripts/dev_bootstrap.sh --help`
- [x] `bash scripts/ops/bigclawctl dev-bootstrap --help`
- [x] `bash scripts/ops/bigclawctl issue-bootstrap sync v1 --dry-run --json`
- [x] `rg -n "python|Python|pytest|PYTHONPATH|ruff|pre-commit|build" README.md workflow.md bigclaw-go/README.md docs/BigClaw-AgentHub-Integration-Alignment.md docs/go-mainline-cutover-handoff.md`
- [x] `cd bigclaw-go && go test ./internal/regression ./internal/api ./internal/repo ./internal/reporting ./internal/product`
- [x] `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl -run 'Test(Frozen|Compile|Freeze|RunLegacyPython)'`
- [x] `bash scripts/ops/bigclawctl legacy-python freeze-audit --json`
- [x] `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- [x] `rg --files src/bigclaw tests scripts | head -n 20`
- [x] `cd bigclaw-go && go test ./...`
- [x] `bash scripts/ops/bigclawctl dev-smoke --json`
- [x] `cd bigclaw-go && go test ./internal/continuationgate ./cmd/bigclawctl`
- [x] `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_policy_gate.go --repo-root .. --scorecard bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json --output bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json --pretty`
- [x] `bash bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py --scorecard bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json --output bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`
- [x] `bash -n bigclaw-go/scripts/e2e/run_all.sh`
- [x] `cd bigclaw-go && go test ./internal/benchmarkmatrix`
- [x] `bash bigclaw-go/scripts/benchmark/run_matrix.py --help`
- [x] `cd bigclaw-go && go run ./scripts/benchmark/run_matrix.go --help`
- [x] `cd bigclaw-go && go test ./internal/soaklocal ./internal/benchmarkmatrix`
- [x] `bash bigclaw-go/scripts/benchmark/soak_local.py --help`
- [x] `cd bigclaw-go && go run ./scripts/benchmark/soak_local.go --help`
- [x] `cd bigclaw-go && go test ./internal/continuationscorecard`
- [x] `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_scorecard.go --repo-root .. --output bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json --pretty`
- [x] `bash bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py --output bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`
- [x] `cd bigclaw-go && go test ./internal/capacitycert`
- [x] `cd bigclaw-go && go run ./scripts/benchmark/capacity_certification.go --repo-root ..`
- [x] `bash bigclaw-go/scripts/benchmark/capacity_certification.py --help`

### Notes

- The repo already merged the earlier Go-mainline cutover, but root-level Python code and non-Go operators remain as migration-only or still-active surfaces. This issue is therefore focused on the next-stage retirement/execution plan rather than redoing the completed cutover.
- Keep changes scoped to migration planning, inventory, tracker seeding, and the first executable Go-owned planning slice.
- Continuation focus: `BIG-VNEXT-GO-104` first-batch progress by migrating `scripts/ops/*workspace*` wrapper behavior into `bigclaw-go/cmd/bigclawctl`.
- Current first-batch progress: `bigclawctl workspace bootstrap` now applies the historical repo/cache-key defaults directly in Go, `workspace validate` now accepts the legacy wrapper flag forms (`--issues` list, `--report-file`, `--no-cleanup`), and `scripts/ops/symphony_workspace_validate.py` no longer carries local argument-translation logic.
- Latest continuation progress: `bigclawctl local-issues` now accepts the historical `state` alias plus positional `state/comment` arguments, allowing `scripts/ops/bigclaw-issue` to drop its local tracker argument-rewrite logic and forward directly to the Go CLI.
- Latest wrapper alignment: `scripts/ops/symphony_workspace_bootstrap.py` now matches `scripts/ops/bigclaw_workspace_bootstrap.py` by forwarding directly to `workspace bootstrap`, eliminating the last workspace-wrapper command-shape mismatch.
- Latest operator-surface shift: README, the refill queue markdown, and the queue markdown generator now recommend `bash scripts/ops/bigclawctl local-issues ...` as the primary tracker CLI, with `bigclaw-issue` reduced to a compatibility alias instead of the default operator entrypoint.
- Latest CLI visibility fix: the `bigclawctl` root help now exposes `go-migration`, and repo metadata/docs now describe the remaining Python-named ops wrappers as compatibility shims over the Go CLI rather than active Python tooling.
- New continuation focus: begin `BIG-VNEXT-GO-107` by moving the minimal `scripts/dev_smoke.py` scheduler smoke assertion into a Go-native `bigclawctl dev-smoke` entrypoint.
- Latest `BIG-VNEXT-GO-107` progress: `bigclawctl dev-smoke` now provides a Go-native scheduler smoke check, README recommends that command in the Go smoke path, and `scripts/dev_smoke.py` now points legacy users at the Go command instead of the old manual service boot sequence.
- New continuation focus: migrate `scripts/create_issues.py` into Go so the remaining developer bootstrap helpers stop depending on Python for GitHub issue seeding.
- Latest `BIG-VNEXT-GO-107` progress: `bigclawctl issue-bootstrap sync` now owns the built-in PRD issue-plan seeding logic, supports dry-run preview, and `scripts/create_issues.py` has been reduced to a shell shim over the Go command.
- Latest `BIG-VNEXT-GO-107` progress: `bigclawctl dev-bootstrap` now owns the developer bootstrap flow, including optional legacy Python setup, and `scripts/dev_bootstrap.sh` has been reduced to a shell shim over the Go command.
- Latest `BIG-VNEXT-GO-107` alignment: README now presents `bash scripts/ops/bigclawctl dev-bootstrap` and `bigclawctl issue-bootstrap sync` as the primary developer entrypoints, with the shell and `.py` shim names retained only in an explicit compatibility section.
- New continuation focus: advance `BIG-VNEXT-GO-109` by cutting contributor-facing docs over to the Go-only workflow and removing active mixed-language validation/setup guidance from human-facing entrypoints.
- Latest `BIG-VNEXT-GO-109` progress: README, `bigclaw-go/README.md`, `docs/go-mainline-cutover-handoff.md`, and `docs/BigClaw-AgentHub-Integration-Alignment.md` now present Go-first contributor flows and validation commands only, while generated migration artifacts continue to retain the historical compatibility inventory outside the contributor quick-start path.
- New continuation focus: advance `BIG-VNEXT-GO-108` by turning the legacy-python helper into an actual frozen-tree audit instead of a shim-only compile smoke check.
- Latest `BIG-VNEXT-GO-108` progress: `src/bigclaw` now carries an explicit frozen-tree README plus top-level freeze markers on the remaining legacy entrypoints, and `bigclawctl legacy-python freeze-audit` now inventories the root compatibility tree, verifies the frozen README, and checks the key entrypoints for migration-only markers from the Go CLI.
- New continuation focus: advance `BIG-VNEXT-GO-106` by removing Python-package bootstrap assumptions from CI and treating the root packaging metadata as compatibility-only.
- Latest `BIG-VNEXT-GO-106` progress: `.github/workflows/ci.yml` now runs Go mainline tests and Go smoke by default, with a separate explicit legacy-compatibility audit job for `freeze-audit` and `compile-check`, so CI no longer installs or builds the root Python package as the canonical mainline.
- New continuation focus: start `BIG-VNEXT-GO-105` by replacing individual Python validation harnesses under `bigclaw-go/scripts/e2e` with Go-owned implementations instead of treating the whole harness tree as one migration cliff.
- Latest `BIG-VNEXT-GO-105` progress: the validation-bundle continuation policy gate now has a Go-native implementation under `bigclaw-go/internal/continuationgate` plus a Go script entrypoint, `run_all.sh` now invokes the Go gate directly, and the legacy `.py` path has been reduced to a shim over the Go command.
- Latest `BIG-VNEXT-GO-105` progress: `scripts/benchmark/run_matrix.py` now routes to a Go-native benchmark-matrix runner backed by `bigclaw-go/internal/benchmarkmatrix`, so another Python harness entrypoint has been collapsed to a shim while preserving the existing command shape.
- Latest `BIG-VNEXT-GO-105` progress: `scripts/benchmark/soak_local.py` now routes to a Go-native soak runner backed by `bigclaw-go/internal/soaklocal`, and the benchmark matrix no longer shells out to Python for its soak scenarios.
- Latest `BIG-VNEXT-GO-105` progress: the validation-bundle continuation scorecard now has a Go-native implementation under `bigclaw-go/internal/continuationscorecard`, `run_all.sh` now invokes the Go scorecard directly, and the legacy `.py` scorecard path has been reduced to a shim over the Go command.
- Latest `BIG-VNEXT-GO-105` progress: `scripts/benchmark/capacity_certification.py` now routes to a Go-native capacity certification generator backed by `bigclaw-go/internal/capacitycert`, with a matching Go test and refreshed checked-in capacity certification artifacts.
