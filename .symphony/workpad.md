# BIG-GO-927

## Plan
- Inventory the current non-Go assets for `test_github_sync.py`, `test_repo_links.py`, `test_repo_registry.py`, and `test_repo_collaboration.py`.
- Map each Python test to an existing Go package under `bigclaw-go/internal`, identify uncovered behavior, and implement the minimum missing Go surface.
- Add or extend targeted Go tests for repo sync, repo links, repo registry, and collaboration/repo board behavior.
- Remove the migrated Python tests once equivalent Go coverage is in place.
- Run targeted validation commands, record exact commands and outcomes, then commit and push the issue branch.

## Acceptance
- Current Python / non-Go assets for the four target areas are explicitly identified.
- Go replacement coverage exists for repo sync, repo links, repo registry, and repo collaboration semantics.
- First batch of Go implementation and tests lands directly in this issue branch.
- Conditions for deleting the legacy Python assets are explicit:
  - matching behavior is covered by targeted Go tests
  - targeted Go tests pass locally
  - no remaining production code path depends on the deleted Python test files
- Regression validation commands are explicit and recorded with results.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-927/bigclaw-go && go test ./internal/githubsync ./internal/repo`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-927/bigclaw-go && go test ./...`

## Validation Results
- `2026-03-28`: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-927/bigclaw-go && go test ./internal/githubsync ./internal/repo`
  - Result: passed
- `2026-03-28`: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-927/bigclaw-go && go test ./...`
  - Result: passed
