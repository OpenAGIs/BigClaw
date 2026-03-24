# BIGCLAW-179 Workpad

## Plan
- Inspect the existing operations dashboard and engineering overview reporting surfaces to find the narrowest extension point for a control-center multi-task heat ranking and congestion localization panel.
- Implement issue-scoped data structures, analytics derivation, and markdown rendering in the operations reporting module without changing unrelated report flows.
- Add or update targeted tests for the new panel content and supporting analytics behavior.
- Run targeted validation commands, capture exact results, then commit and push the branch.

## Acceptance
- The control-center reporting surface exposes a dedicated multi-task heat ranking and congestion localization panel.
- Panel content is derived from deterministic analytics over run/task inputs and surfaces actionable ranking and congestion details.
- Existing dashboard/report generation continues to work with the new panel present.
- Targeted automated tests cover the new behavior.

## Validation
- Run focused test commands for operations analytics/report rendering.
- Review generated diff for issue scope.
- Commit on `BIGCLAW-179` and push to `origin/BIGCLAW-179`.
