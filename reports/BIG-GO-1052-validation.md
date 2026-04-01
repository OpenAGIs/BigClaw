# BIG-GO-1052 Validation

Date: 2026-04-01

## Scope

Issue: `BIG-GO-1052`

Title: `Go-replacement V: remove bigclaw-go e2e Python helpers tranche 1`

This lane locks the first `bigclaw-go/scripts/e2e` migration tranche to a Go-only
operator surface by:

- asserting the removed tranche-1 Python helpers stay absent
- asserting `scripts/e2e/run_all.sh` continues to route through Go CLI entrypoints
- removing the remaining e2e documentation wording that still implied Python helper
  prerequisites or Python-shaped smoke entrypoints

## Delivered

- added `TestE2EScriptsStayGoOnly` in
  `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`, which:
  - fails if `bigclaw-go/scripts/e2e/` contains any `.py` file
  - asserts `scripts/e2e/run_all.sh` still uses:
    - `go run ./cmd/bigclawctl automation e2e run-task-smoke`
    - `go run ./cmd/bigclawctl automation e2e export-validation-bundle`
    - `go run ./cmd/bigclawctl automation e2e continuation-scorecard`
    - `go run ./cmd/bigclawctl automation e2e continuation-policy-gate`
- added tranche-level migration coverage in
  `bigclaw-go/internal/regression/top_level_module_purge_tranche12_test.go`, which:
  - asserts the retired tranche-1 helpers remain absent:
    - `run_task_smoke.py`
    - `export_validation_bundle.py`
    - `validation_bundle_continuation_scorecard.py`
    - `validation_bundle_continuation_policy_gate.py`
    - `broker_failover_stub_matrix.py`
    - `mixed_workload_matrix.py`
    - `cross_process_coordination_surface.py`
    - `subscriber_takeover_fault_matrix.py`
    - `external_store_validation.py`
    - `multi_node_shared_queue.py`
  - asserts the Go replacements and retained shell wrapper exist
- added `bigclaw-go/internal/regression/e2e_entrypoint_docs_test.go`, which
  fails if active README/workflow/e2e validation surfaces drift back to retired
  tranche-1 Python helper names
- updated the operator-facing docs so `scripts/e2e/` is described as a
  Go-and-shell-only surface:
  - `bigclaw-go/README.md`
  - `bigclaw-go/docs/e2e-validation.md`
  - `bigclaw-go/docs/go-cli-script-migration.md`

## Validation

### Python file counts

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go/scripts/e2e -name '*.py' | wc -l
```

Result:

```text
0
```

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052 -path '/Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/.git' -prune -o -name '*.py' -print | wc -l
```

Result:

```text
50
```

Note: the tranche-1 `bigclaw-go/scripts/e2e/*.py` helpers were already absent in this
checkout before the lane changes landed, so this issue enforces the Go-only state and
removes residual entrypoint drift rather than performing a fresh in-branch file deletion.

### Targeted Go tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go test ./cmd/bigclawctl ./internal/regression
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	3.577s
ok  	bigclaw-go/internal/regression	0.492s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go test ./internal/regression ./cmd/bigclawctl
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.511s
ok  	bigclaw-go/cmd/bigclawctl	(cached)
```

### Go entrypoint help checks

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e run-task-smoke [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e export-validation-bundle [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e continuation-policy-gate [flags]`

### Diff hygiene

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052 && git diff --check
```

Result: exit code `0`

## Branch

- Branch: `feat/BIG-GO-1052-go-e2e-tranche1-regression`
- Implementation commit before this validation report: `ef82780f438b1c9bf6e7238c7ad92fe27f1b295c`

## Residual Risk

- This lane does not reduce the repo-wide `.py` count inside this branch because the target
  `bigclaw-go/scripts/e2e/*.py` tranche had already been deleted before the branch started.
- The Go-only contract now depends on the new regression coverage and on keeping operator docs
  aligned with `bigclawctl automation e2e ...` plus the retained shell wrappers.
