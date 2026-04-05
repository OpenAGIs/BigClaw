# BIG-GO-1476 Workpad

## Plan
- Confirm the remaining bootstrap/workspace helper Python assets that still overlap the Go CLI implementation.
- Remove obsolete Python files that are already replaced by `bigclaw-go/internal/bootstrap` and `bigclaw-go/cmd/bigclawctl`.
- Trim the Python compatibility/test metadata that only existed to support those deleted workspace wrapper surfaces.
- Add an issue-scoped migration note documenting deleted files, Go ownership, and the resulting Python file-count reduction.
- Run targeted validation covering Go bootstrap/workspace behavior plus repo-level file-count evidence.
- Commit the changes and push the branch.

## Acceptance
- The repository contains fewer physical `.py` files than before the change.
- Deleted workspace/bootstrap helper surfaces are documented with their Go replacement owner or an explicit delete rationale.
- No new parallel Python workspace/bootstrap path is introduced.
- Targeted validation proves the Go CLI still owns bootstrap/workspace behavior after the deletions.

## Validation
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/bootstrap`
- `python3 -m pytest tests/test_legacy_shim.py`
- `rg --files -g '*.py' | wc -l`
- `git diff --stat`
