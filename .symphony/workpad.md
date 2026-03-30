# BIG-GO-983 Workpad

## Scope

Targeted `src/bigclaw/**` core-module cleanup batch for files that already have a Go-owned mainline replacement and no remaining in-repo Python imports beyond legacy package exports.

Candidate batch:

- `src/bigclaw/mapping.py`
- `src/bigclaw/issue_archive.py`
- `src/bigclaw/parallel_refill.py`
- `src/bigclaw/pilot.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `src/bigclaw/cost_control.py`
- `src/bigclaw/roadmap.py`
- `src/bigclaw/connectors.py`

Planned retainers for this lane:

- none

Current repository Python file count before this lane: `116`
Current `src/bigclaw/**` Python file count before this lane: `45`

## Plan

1. Confirm each candidate file is either unreferenced in the remaining Python tree or only reachable through stale package exports.
2. Validate the existing Go replacement paths for intake mapping, issue archive, refill/bootstrap tooling, and pilot reporting.
3. Delete the selected Python files and remove stale imports/exports from `src/bigclaw/__init__.py`.
4. Run targeted Go validation for the replacement packages and recount Python files.
5. Record per-file keep/delete rationale, exact commands, and before/after counts.

## Acceptance

- Produce the exact `BIG-GO-983` batch file list and disposition.
- Reduce Python files under `src/bigclaw/**` by removing the safely migrated subset.
- Keep changes scoped to this core-module cleanup batch.
- Report the repository-wide and `src/bigclaw/**` Python file-count impact.
- Record exact validation commands and results.

## Validation

- `cd bigclaw-go && go test ./internal/intake ./internal/issuearchive ./internal/refill ./internal/pilot ./cmd/bigclawctl`
- `python3 -m compileall src/bigclaw/__init__.py`
- `rg --files src/bigclaw -g '*.py' | wc -l`
- `rg --files -g '*.py' | wc -l`
- `git status --short`

## Results

### File Disposition

- `src/bigclaw/mapping.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; parity matrix maps the full surface to `bigclaw-go/internal/intake/mapping.go`, and Go intake tests already own the active mapping contract.
- `src/bigclaw/issue_archive.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; the equivalent archive/audit/report surface exists in `bigclaw-go/internal/issuearchive/archive.go` with package tests.
- `src/bigclaw/parallel_refill.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; cutover docs assign refill ownership to `bigclaw-go/internal/refill/*`, and the queue behavior is covered by Go refill tests plus `cmd/bigclawctl`.
- `src/bigclaw/pilot.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; the same implementation-report surface exists in `bigclaw-go/internal/pilot/report.go` with Go tests.
- `src/bigclaw/workspace_bootstrap_cli.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; cutover docs assign bootstrap CLI ownership to `bigclaw-go/cmd/bigclawctl` and `bigclaw-go/internal/bootstrap/*`, which are already covered by Go tests.
- `src/bigclaw/cost_control.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; equivalent budget-evaluation behavior exists in `bigclaw-go/internal/costcontrol/controller.go` with package tests.
- `src/bigclaw/roadmap.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; Go regression coverage in `bigclaw-go/internal/regression/roadmap_contract_test.go` now owns the canonical execution-pack roadmap contract.
- `src/bigclaw/connectors.py`
  - Deleted.
  - Reason: no remaining in-repo Python imports; parity matrix maps `SourceIssue` and connector stubs to `bigclaw-go/internal/intake/types.go` and `bigclaw-go/internal/intake/connector.go`, both covered by Go intake tests.
- `src/bigclaw/__init__.py`
  - Updated.
  - Reason: removed stale package-level imports/exports for deleted modules so `import bigclaw` no longer hard-fails on removed files.

### Python File Count Impact

- Repository Python files before: `116`
- Repository Python files after: `108`
- `src/bigclaw/**` Python files before: `45`
- `src/bigclaw/**` Python files after: `37`
- Net reduction: `8`

### Validation Record

- `cd bigclaw-go && go test ./internal/intake ./internal/issuearchive ./internal/refill ./internal/pilot ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/internal/intake	1.125s`
  - Result: `ok  	bigclaw-go/internal/issuearchive	1.556s`
  - Result: `ok  	bigclaw-go/internal/refill	4.718s`
  - Result: `ok  	bigclaw-go/internal/pilot	1.840s`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	3.818s`
- `python3 -m compileall src/bigclaw/__init__.py`
  - Result: `Compiling 'src/bigclaw/__init__.py'...`
- `cd bigclaw-go && go test ./internal/costcontrol ./internal/regression`
  - Result: `ok  	bigclaw-go/internal/costcontrol	1.088s`
  - Result: `ok  	bigclaw-go/internal/regression	1.332s`
- `cd bigclaw-go && go test ./internal/intake`
  - Result: `ok  	bigclaw-go/internal/intake	(cached)`
- `git status --short`
  - Result: before the third commit, only `src/bigclaw/__init__.py` and `src/bigclaw/connectors.py` changed.
