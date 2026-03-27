# Repo Governance And Board Migration Plan

Issue: `BIG-GO-905`

## Objective

Move the repo governance, repo board, and repo triage Python surfaces onto Go-owned capability packages without expanding scope into unrelated repo transport or workflow migration.

## Current Ownership Snapshot

Python source of truth today:
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_triage.py`

Go surfaces in this slice:
- `bigclaw-go/internal/repo/governance.go`
- `bigclaw-go/internal/repo/board.go`
- `bigclaw-go/internal/repo/triage.go`
- `bigclaw-go/internal/repo/governance_test.go`
- `bigclaw-go/internal/repo/board_test.go`
- `bigclaw-go/internal/repo/repo_surfaces_test.go`

## Executable Migration Plan

1. Lock the capability boundary in `bigclaw-go/internal/repo`.
   - Keep repo governance, board, and lineage-aware approval triage contracts together.
   - Avoid mixing these contracts into scheduler, workflow, or API packages until the capability contract is stable.
2. Mirror Python behavior first.
   - Preserve permission names, role policies, required audit fields, post/reply semantics, target filters, and lineage-based triage recommendations.
   - Treat Go behavior parity tests as the cutover gate.
3. Add Go-native integration points second.
   - Wire `internal/api` and `cmd/bigclawctl` to the Go repo capability package instead of re-encoding rules inline.
   - Keep Python as compatibility-only until all callers stop importing the Python modules.
4. Retire Python call sites last.
   - Remove or freeze Python entrypoints only after API/CLI/reporting flows use the Go owners and parity tests stay green.

## First Batch Implementation / Renovation List

Completed in this slice:
- `bigclaw-go/internal/repo/governance.go`
  - Exposes the repo permission matrix, audit-field contract, and clone-safe capability inventory accessors.
- `bigclaw-go/internal/repo/board.go`
  - Mirrors Python repo-board post serialization defaults, post/reply creation, filtering, and repo-post to collaboration-comment conversion.
- `bigclaw-go/internal/repo/triage.go`
  - Keeps lineage-aware repo triage recommendation and approval-evidence packet ownership in the same Go capability family.
- `bigclaw-go/internal/repo/board_test.go`
  - Adds Python-parity tests for `RepoPostFromMap`, `ToMap`, `ToCollaborationComment`, and board reply/filter behavior.

Follow-on slices after this branch:
- Move repo-board consumers in reporting/service layers to `bigclaw-go/internal/repo`.
- Decide whether `bigclaw-go/internal/triage/repo.go` should stay as an API-facing adapter or collapse onto `internal/repo/triage.go`.
- Remove remaining Python imports of `repo_governance`, `repo_board`, and `repo_triage` once Go callers exist.

## Validation Commands

Primary regression commands for this slice:
- `cd bigclaw-go && go test ./internal/repo/...`
- `cd bigclaw-go && go test ./internal/triage ./internal/governance`

Recommended parity spot-check while Python remains present:
- `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_governance.py tests/test_repo_triage.py`

## Regression Surface

Required regression coverage:
- Repo permission allow/deny decisions by actor role.
- Required audit-field lists for repo actions.
- Repo board post numbering, reply inheritance, and channel/target filtering.
- Repo post serialization defaults, especially `target_surface="task"` and RFC3339 timestamps.
- Repo triage lineage recommendation rules and approval-evidence packet contents.

Likely affected consumers:
- `bigclaw-go/internal/api/v2.go`
- `bigclaw-go/internal/product/console.go`
- Future repo collaboration/reporting slices under `bigclaw-go/internal/reporting`

## Branch / PR Recommendation

Branch:
- `big-go-905`

PR recommendation:
- Title: `BIG-GO-905: migrate repo governance board capability surfaces to Go`
- Scope note: keep this PR limited to repo governance/board/triage contracts plus migration documentation; defer API/CLI rewiring to follow-up PRs.

## Risks

- There are now two Go repo-triage homes: `internal/repo/triage.go` and `internal/triage/repo.go`. Without a clear adapter boundary, future edits can drift.
- `RepoPostFromMap` currently accepts loose map input for Python parity; if external callers depend on richer metadata decoding, that needs explicit typing before broad API exposure.
- Python compatibility remains in place, so true cutover is not complete until callers are migrated and dead imports are removed.
