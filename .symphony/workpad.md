Issue: BIG-GO-1028

Plan
- Retire `tests/test_audit_events.py` by tightening Go-native audit-spec coverage in `bigclaw-go/internal/observability` for the remaining manual-takeover, budget-override, and flow-handoff event contracts.
- Keep the change set scoped to the owned observability test surface plus the Python test deletion and required workpad update only.
- Delete the migrated Python test file so this tranche reduces repository `.py` inventory immediately.
- Run targeted file-count checks and Go tests; record exact commands and outcomes for final closeout.
- Commit only the scoped issue changes and push the branch to the remote.

Acceptance
- Changes remain scoped to the selected tranche-3 Python test deletion and directly supporting Go-native tests.
- Repository `.py` file count decreases by deleting the migrated Python test file.
- Repository `.go` file count remains unchanged.
- `pyproject.toml`, `setup.py`, and `setup.cfg` remain unchanged.
- Final report includes the impact on `.py` count, `.go` count, and `pyproject/setup*` files.

Validation
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l`
- `cd bigclaw-go && go test ./internal/observability -run 'TestP0AuditEventSpecsDefineRequiredOperationalEvents|TestMissingRequiredFieldsForEventUsesTopLevelAuditIdentifiers|TestMissingRequiredFieldsForManualTakeoverEventUsesTopLevelAuditIdentifiers|TestMissingRequiredFieldsForBudgetOverrideAndFlowHandoffEvents|TestMissingRequiredFieldsForEventReturnsSpecGaps|TestJSONLAuditSinkRejectsMalformedKnownAuditEvent|TestRecordSpecEventRejectsMalformedAuditEventsBeforeMutation|TestRecordSpecEventAcceptsWellFormedAuditEvents'`
- `git diff --stat`
- `git status --short`
