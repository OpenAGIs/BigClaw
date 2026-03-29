# BIG-GO-944 Workpad

## Scope

Lane 4 repo governance/reporting modules for the Go-only materialization pass.

Python lane files in scope:
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/risk.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/memory.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/observability.py`

Existing Go owners already present:
- `bigclaw-go/internal/repo/governance.go`
- `bigclaw-go/internal/governance/freeze.go`
- `bigclaw-go/internal/reporting/reporting.go`
- `bigclaw-go/internal/risk/risk.go`
- `bigclaw-go/internal/observability/audit.go`
- `bigclaw-go/internal/observability/recorder.go`
- `bigclaw-go/internal/intake/mapping.go`

Expected new or expanded Go owners in this issue:
- `bigclaw-go/internal/planning/*`
- `bigclaw-go/internal/memory/*`
- `bigclaw-go/internal/observability/*` for repo-sync/run-closeout/runtime ledger shapes

File ownership mapping for this lane:
- `src/bigclaw/repo_governance.py` -> `bigclaw-go/internal/repo/governance.go` already present
- `src/bigclaw/governance.py` -> `bigclaw-go/internal/governance/freeze.go` already present
- `src/bigclaw/risk.py` -> `bigclaw-go/internal/risk/risk.go` already present
- `src/bigclaw/mapping.py` -> `bigclaw-go/internal/intake/mapping.go` already present
- `src/bigclaw/operations.py` -> `bigclaw-go/internal/reporting/reporting.go` already present for lane-owned operations/reporting surfaces
- `src/bigclaw/reports.py` -> `bigclaw-go/internal/reporting/reporting.go` already present for lane-owned reporting surfaces
- `src/bigclaw/planning.py` -> new `bigclaw-go/internal/planning/planning.go`
- `src/bigclaw/memory.py` -> new `bigclaw-go/internal/memory/store.go`
- `src/bigclaw/observability.py` -> existing `bigclaw-go/internal/observability/audit.go` and `recorder.go`, expanded with new `runtime.go`

## Acceptance

- Identify the exact lane file inventory and map each file to an existing Go owner, a new Go owner, or a temporary compatibility hold.
- Add Go replacements for missing lane surfaces:
  - planning backlog and gate evaluation
  - task memory persistence and suggestion rules
  - observability runtime closeout/ledger data shapes still only present in Python
- Keep changes scoped to lane modules only.
- Record exact validation commands and results.
- Leave explicit residual-risk notes for any Python files that remain because active repo consumers still import them.

## Plan

1. Add `internal/planning` with candidate backlog, entry gate, gate evaluation, reporting, and canned seed builders matching the current Python semantics used by the repo.
2. Add `internal/memory` with `Pattern` and `TaskStore` equivalents for persisted task memory and acceptance/validation suggestion rules.
3. Expand `internal/observability` with repo sync audit, run closeout, runtime run record, and JSON ledger persistence.
4. Add focused Go tests for each new or expanded surface.
5. Run targeted `go test` commands for the touched packages.
6. Commit and push a branch for `BIG-GO-944`.

## Validation

Planned commands:
- `cd bigclaw-go && go test ./internal/planning ./internal/memory ./internal/observability ./internal/governance ./internal/reporting ./internal/risk ./internal/repo`
- `cd bigclaw-go && go test ./...`

Result log:
- `cd bigclaw-go && go test ./internal/planning ./internal/memory ./internal/observability ./internal/governance ./internal/reporting ./internal/risk ./internal/repo`
  - `ok  	bigclaw-go/internal/planning	0.421s`
  - `ok  	bigclaw-go/internal/memory	0.855s`
  - `ok  	bigclaw-go/internal/observability	1.230s`
  - `ok  	bigclaw-go/internal/governance	1.656s`
  - `ok  	bigclaw-go/internal/reporting	2.032s`
  - `ok  	bigclaw-go/internal/risk	2.471s`
  - `ok  	bigclaw-go/internal/repo	2.881s`
- `cd bigclaw-go && go test ./...`
  - pass; full tree succeeded, including `internal/api`, `internal/queue`, `internal/regression`, `internal/worker`, and `cmd/*`

## Residual Risks

- The Python package under `src/bigclaw` still has active imports and Python tests across the repo, so this issue can reduce lane-specific Python-only ownership but cannot delete all Python compatibility surfaces without a wider cutover.
- `src/bigclaw/reports.py` and `src/bigclaw/operations.py` are broader than this lane and still back Python tests and non-Go entrypoints; this issue should avoid destabilizing them.
