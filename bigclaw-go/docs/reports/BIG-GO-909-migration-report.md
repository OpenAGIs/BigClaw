# BIG-GO-909 Migration Report

## Scope

This issue closes the current Go migration slice for these repo-native surfaces:

- `github sync`
- `repo links`
- `repo registry`
- `repo collaboration`

The implementation stays bounded to `bigclaw-go/internal/githubsync`,
`bigclaw-go/internal/repo`, and issue-specific documentation.

## Current Go Ownership

| Surface | Go status | Evidence |
| --- | --- | --- |
| `github sync` | Implemented | `bigclaw-go/internal/githubsync/sync.go`, `bigclaw-go/cmd/bigclawctl/main.go` |
| `repo links` | Implemented | `bigclaw-go/internal/repo/links.go` |
| `repo registry` | Implemented | `bigclaw-go/internal/repo/registry.go` |
| `repo collaboration` | Completed in this issue | `bigclaw-go/internal/repo/board.go`, `bigclaw-go/internal/repo/collaboration.go` |

## First-Batch Implementation / Adaptation Checklist

- [x] Confirm the Go CLI already owns `github-sync install|status|sync`.
- [x] Confirm run commit link validation and accepted-hash selection already exist in Go.
- [x] Confirm repo space/channel/agent resolution already exists in Go.
- [x] Add Go repo-collaboration primitives mirroring the Python `collaboration.py` thread/comment/decision model.
- [x] Add repo-board to collaboration-comment conversion in Go.
- [x] Add targeted regression coverage for collaboration merge, audit reconstruction, and rendering.
- [x] Record validation commands, regression surface, branch/PR guidance, and risks in-repo.

## Regression Surface

- `bigclaw-go/internal/githubsync`
  - detached HEAD reporting
  - default-branch sync fallback
  - dirty worktree / ahead / behind / diverged handling
  - hook installation and config-lock retry behavior
- `bigclaw-go/internal/repo/links.go`
  - run commit role validation
  - accepted commit selection
- `bigclaw-go/internal/repo/registry.go`
  - project-to-space resolution
  - default channel derivation
  - cached repo-agent resolution
- `bigclaw-go/internal/repo/board.go`
  - post/reply/filter behavior
  - repo-board post conversion into collaboration comments
- `bigclaw-go/internal/repo/collaboration.go`
  - thread construction
  - merge ordering
  - audit-to-thread reconstruction
  - recommendation derivation
  - text rendering

## Validation Commands

- `cd bigclaw-go && go test ./internal/githubsync ./internal/repo`

## Validation Results

- `2026-03-27`: `cd bigclaw-go && go test ./internal/githubsync ./internal/repo`
- Result:
  - `ok  	bigclaw-go/internal/githubsync	3.328s`
  - `ok  	bigclaw-go/internal/repo	1.074s`

## Branch / PR Suggestion

- Branch: `symphony/BIG-GO-909`
- PR title: `BIG-GO-909: migrate GitHub sync and repo collaboration surfaces to Go`
- PR summary:
  - keep `github sync`, `repo links`, and `repo registry` on the existing Go owner path
  - add missing repo-collaboration primitives in `internal/repo`
  - lock the migration with targeted `go test ./internal/githubsync ./internal/repo`

## Risks

- The Go repo collaboration slice currently mirrors the Python text-thread behavior, but not the HTML rendering surface. If downstream UI surfaces still depend on Python-only HTML helpers, those should be migrated in a separate bounded issue.
- `repo registry` parity is runtime-focused. Python-specific dictionary round-trip helpers are not reintroduced in Go because the active Go runtime uses typed structs; adding generic map serialization should be justified by a concrete caller.
- `github sync` still shells out to `git`. Environment-level git differences remain an integration risk outside unit coverage.
