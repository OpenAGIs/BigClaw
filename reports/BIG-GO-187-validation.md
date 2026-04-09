# BIG-GO-187 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-187`

Title: `Broad repo Python reduction sweep AA`

This issue reduced residual legacy Python reference density in the active
migration-doc surface while preserving the repo's already Python-free physical
asset baseline.

## Delivered

- Replaced `.symphony/workpad.md` with the BIG-GO-187 plan, acceptance
  criteria, and targeted validation commands before code edits landed.
- Compacted repeated legacy `.py` enumerations in:
  - `docs/go-cli-script-migration-plan.md`
  - `docs/go-mainline-cutover-issue-pack.md`
  - `docs/go-mainline-cutover-handoff.md`
- Added `bigclaw-go/internal/regression/big_go_187_zero_python_guard_test.go`
  to pin per-doc legacy-reference budgets, required grouped path forms, and
  the absence of the previously expanded examples.
- Added `bigclaw-go/docs/reports/big-go-187-python-asset-sweep.md` to document
  the reduction and the exact validation surface.

## Reduction Evidence

Command:

```bash
for f in docs/go-cli-script-migration-plan.md docs/go-mainline-cutover-issue-pack.md docs/go-mainline-cutover-handoff.md; do before=$(git show HEAD:"$f" | rg -o 'src/bigclaw/[^`[:space:]]+|scripts/[^`[:space:]]+\.py|bigclaw-go/scripts/[^`[:space:]]+\.py|python3|\.py' | wc -l | tr -d ' '); after=$(rg -o 'src/bigclaw/[^`[:space:]]+|scripts/[^`[:space:]]+\.py|bigclaw-go/scripts/[^`[:space:]]+\.py|python3|\.py' "$f" | wc -l | tr -d ' '); printf '%s %s %s\n' "$f" "$before" "$after"; done
```

Result:

```text
docs/go-cli-script-migration-plan.md 31 10
docs/go-mainline-cutover-issue-pack.md 83 24
docs/go-mainline-cutover-handoff.md 10 3
```

## Validation

### Repository Python inventory

Command:

```bash
find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort
```

Result:

```text
[no output]
```

### Target doc/code directories Python inventory

Command:

```bash
find docs bigclaw-go/internal bigclaw-go/cmd -type f -name '*.py' -print 2>/dev/null | sort
```

Result:

```text
[no output]
```

### Residual reference audit

Command:

```bash
rg -n "python3|\.py\b|#!/usr/bin/env python|#!/usr/bin/python" docs bigclaw-go/internal bigclaw-go/cmd --glob '!bigclaw-go/internal/regression/**' --glob '!bigclaw-go/docs/reports/**' | head -n 200
```

Result:

```text
docs/symphony-repo-bootstrap-template.md:12:- `src/<your_package>/workspace_bootstrap.py`
docs/symphony-repo-bootstrap-template.md:13:- `src/<your_package>/workspace_bootstrap_cli.py`
docs/go-mainline-cutover-handoff.md:26:- `PYTHONPATH=src python3 - <<"... legacy shim assertions ..."`
docs/go-mainline-cutover-handoff.md:31:  held in `src/bigclaw/{models,connectors,mapping,dsl}.py` and the later
docs/go-mainline-cutover-handoff.md:33:  `src/bigclaw/{governance,observability,operations,orchestration,pilot}.py`.
bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go:278:	stubGo := `#!/usr/bin/env python3
docs/go-cli-script-migration-plan.md:18:- `bigclaw-go/scripts/benchmark/{capacity_certification,capacity_certification_test,run_matrix,soak_local}.py`
docs/go-cli-script-migration-plan.md:19:- `bigclaw-go/scripts/e2e/{broker_failover_stub_matrix,broker_failover_stub_matrix_test,cross_process_coordination_surface,export_validation_bundle,export_validation_bundle_test,external_store_validation,mixed_workload_matrix,multi_node_shared_queue,multi_node_shared_queue_test,run_all_test,run_task_smoke,subscriber_takeover_fault_matrix,validation_bundle_continuation_policy_gate,validation_bundle_continuation_policy_gate_test,validation_bundle_continuation_scorecard}.py`
docs/go-cli-script-migration-plan.md:20:- `bigclaw-go/scripts/migration/{export_live_shadow_bundle,live_shadow_scorecard,shadow_compare,shadow_matrix}.py`
docs/go-cli-script-migration-plan.md:21:- `scripts/{create_issues,dev_smoke}.py`
docs/go-cli-script-migration-plan.md:29:- retired `scripts/create_issues.py`; use `bigclawctl create-issues`
docs/go-cli-script-migration-plan.md:34:- retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`
docs/go-cli-script-migration-plan.md:36:- retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`
docs/go-cli-script-migration-plan.md:37:- retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`
docs/go-cli-script-migration-plan.md:38:- retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`
docs/go-cli-script-migration-plan.md:136:  `scripts/ops/bigclaw_github_sync.py` must stay deleted and hooks/workflow/CI should invoke
docs/go-mainline-cutover-issue-pack.md:97:- `src/bigclaw/{models,connectors,mapping,dsl}.py`
docs/go-mainline-cutover-issue-pack.md:118:- remaining `models.py` contract structs still need to be folded into the existing Go runtime / orchestration packages instead of copied into one compatibility file
docs/go-mainline-cutover-issue-pack.md:129:- `src/bigclaw/{risk,governance,execution_contract,audit_events}.py`
docs/go-mainline-cutover-issue-pack.md:142:- `bigclaw-go/internal/governance/freeze.go` now owns the Go scope-freeze backlog board and governance audit surface migrated from `src/bigclaw/governance.py`
docs/go-mainline-cutover-issue-pack.md:143:- `bigclaw-go/internal/contract/execution.go` now owns the Go execution contract, permission matrix, and operations API contract migrated from `src/bigclaw/execution_contract.py`
docs/go-mainline-cutover-issue-pack.md:144:- `bigclaw-go/internal/observability/audit_spec.go` now owns the canonical P0 audit event spec registry migrated from `src/bigclaw/audit_events.py`
docs/go-mainline-cutover-issue-pack.md:160:- `src/bigclaw/{runtime,scheduler,orchestration,workflow,queue}.py`
docs/go-mainline-cutover-issue-pack.md:186:- `src/bigclaw/{observability,reports,evaluation,operations}.py`
docs/go-mainline-cutover-issue-pack.md:211:- `src/bigclaw/{repo_triage,run_detail,dashboard_run_contract,operations,saved_views}.py`
docs/go-mainline-cutover-issue-pack.md:235:- `src/bigclaw/{repo_links,repo_commits,repo_gateway,repo_plane,repo_board,repo_registry,repo_governance}.py`
docs/go-mainline-cutover-issue-pack.md:252:- `bigclaw-go/internal/repo/governance.go` now ports `src/bigclaw/repo_governance.py` into a Go-owned repo permission matrix and audit-field contract
docs/go-mainline-cutover-issue-pack.md:264:- `src/bigclaw/{github_sync,workspace_bootstrap,workspace_bootstrap_cli,workspace_bootstrap_validation,parallel_refill}.py`
docs/go-mainline-cutover-issue-pack.md:265:- `scripts/ops/bigclaw_github_sync.py`
docs/go-mainline-cutover-issue-pack.md:266:- retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`
docs/go-mainline-cutover-issue-pack.md:286:- `workflow.md`, `.githooks/post-commit`, and `.githooks/post-rewrite` invoke the Go-first toolchain by default, and the legacy `scripts/ops/bigclaw_github_sync.py` wrapper has been removed
docs/go-mainline-cutover-issue-pack.md:298:- `src/bigclaw/{service,__main__}.py`
docs/go-mainline-cutover-issue-pack.md:365:- `src/bigclaw/risk.py`
docs/go-mainline-cutover-issue-pack.md:366:- remaining active consumers of `src/bigclaw/{governance,execution_contract,audit_events}.py`
docs/go-mainline-cutover-issue-pack.md:380:- `src/bigclaw/{runtime,scheduler,orchestration,workflow,queue}.py`
docs/go-mainline-cutover-issue-pack.md:395:- `src/bigclaw/{observability,reports,operations,evaluation,run_detail,dashboard_run_contract,planning}.py`
docs/go-mainline-cutover-issue-pack.md:411:- `src/bigclaw/{collaboration,repo_board,repo_commits,repo_gateway,repo_governance,repo_links,repo_plane,repo_registry,repo_triage,issue_archive,roadmap}.py`
docs/go-mainline-cutover-issue-pack.md:426:- `src/bigclaw/{console_ia,design_system,saved_views,ui_review}.py`
docs/go-mainline-cutover-issue-pack.md:427:- remaining operator-facing parts of `src/bigclaw/service.py`
docs/go-mainline-cutover-issue-pack.md:441:- `src/bigclaw/{github_sync,parallel_refill,workspace_bootstrap,workspace_bootstrap_cli,workspace_bootstrap_validation,service,__main__}.py`
docs/go-domain-intake-parity-matrix.md:17:### Retired `src/bigclaw/models.py`
docs/go-domain-intake-parity-matrix.md:40:### `src/bigclaw/connectors.py`
docs/go-domain-intake-parity-matrix.md:46:### `src/bigclaw/mapping.py`
docs/go-domain-intake-parity-matrix.md:52:### `src/bigclaw/dsl.py`
docs/go-domain-intake-parity-matrix.md:66:- The retired Python `models.py` surface was split by responsibility into
docs/go-domain-intake-parity-matrix.md:78:  and the active worktree no longer carries tracked `.py` source files.
bigclaw-go/cmd/bigclawctl/automation_commands_test.go:30:		if strings.HasSuffix(entry.Name(), ".py") {
bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go:6:			RetiredPythonTest: "tests/test_design_system.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go:20:			RetiredPythonTest: "tests/test_dsl.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go:34:			RetiredPythonTest: "tests/test_evaluation.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go:47:			RetiredPythonTest: "tests/test_parallel_validation_bundle.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_x.go:6:			RetiredPythonTest: "tests/test_audit_events.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_x.go:21:			RetiredPythonTest: "tests/test_connectors.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_x.go:36:			RetiredPythonTest: "tests/test_console_ia.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_x.go:51:			RetiredPythonTest: "tests/test_dashboard_run_contract.py",
bigclaw-go/internal/observability/task_run_test.go:86:			DirtyPaths:      []string{"src/bigclaw/workflow.py"},
bigclaw-go/internal/migration/legacy_model_runtime_modules.go:14:			RetiredPythonModule: "src/bigclaw/models.py",
bigclaw-go/internal/migration/legacy_model_runtime_modules.go:31:			RetiredPythonModule: "src/bigclaw/runtime.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go:14:			RetiredPythonTest: "tests/test_control_center.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go:28:			RetiredPythonTest: "tests/test_operations.py",
bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go:43:			RetiredPythonTest: "tests/test_ui_review.py",
bigclaw-go/internal/executor/ray_test.go:39:	result := runner.Execute(context.Background(), domain.Task{ID: "OPE-182", Title: "run on ray", Entrypoint: "python app.py"})
```

### Targeted regression guard

Command:

```bash
cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO187' -count=1
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.192s
```

### Git status before commit

Command:

```bash
git status --short
```

Result:

```text
 M .symphony/workpad.md
 M docs/go-cli-script-migration-plan.md
 M docs/go-mainline-cutover-handoff.md
 M docs/go-mainline-cutover-issue-pack.md
?? bigclaw-go/docs/reports/big-go-187-python-asset-sweep.md
?? bigclaw-go/internal/regression/big_go_187_zero_python_guard_test.go
?? reports/BIG-GO-187-status.json
?? reports/BIG-GO-187-validation.md
```

## Residual Risk

- This lane reduces legacy Python reference density only in the scoped active
  migration-doc surface; other intentional history and compatibility references
  remain elsewhere in docs and migration tests.
