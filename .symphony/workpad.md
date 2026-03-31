Issue: BIG-GO-1028

Plan
- Retire `tests/test_design_system.py` by adding an isolated Go-native design-system compatibility surface in a new `bigclaw-go/internal/designsystem` package without modifying the unrelated in-flight edits already present in `internal/reporting` or `internal/planning`.
- Port the component governance, console top-bar, information architecture, and UI acceptance assertions into a new Go `_test.go` file that exercises release-readiness audits, route resolution, and acceptance reporting.
- Delete the migrated Python test file so this tranche reduces repository `.py` inventory immediately.
- Run targeted file-count checks and Go tests; record exact commands and outcomes for final closeout.
- Commit only the scoped issue changes and push the branch to the remote.

Acceptance
- Changes remain scoped to the selected tranche-3 Python test deletion and directly supporting Go-native design-system files.
- Repository `.py` file count decreases by deleting the migrated Python test file.
- Repository `.go` file count increases only for the new Go-native design-system compatibility files.
- `pyproject.toml`, `setup.py`, and `setup.cfg` remain unchanged.
- Final report includes the impact on `.py` count, `.go` count, and `pyproject/setup*` files.

Validation
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l`
- `cd bigclaw-go && go test ./internal/designsystem -run 'TestComponentReleaseReadyRequiresDocsAccessibilityAndStates|TestDesignSystemRoundTripPreservesManifestShape|TestDesignSystemAuditSurfacesReleaseGapsAndOrphanTokens|TestDesignSystemAuditFlagsUndefinedTokenReferences|TestDesignSystemAuditRoundTripPreservesGovernanceFindings|TestRenderDesignSystemReportSummarizesInventoryAndGaps|TestConsoleTopBarRoundTripPreservesCommandEntryManifest|TestConsoleTopBarAuditChecksTicketCapabilitiesAndShortcuts|TestConsoleTopBarAuditFlagsMissingGlobalEntryCapabilities|TestRenderConsoleTopBarReportSummarizesGlobalHeaderAndShell|TestInformationArchitectureRoundTripAndRouteResolution|TestInformationArchitectureAuditFlagsDuplicatesSecondaryGapsAndOrphans|TestInformationArchitectureAuditRoundTripAndReport|TestUIAcceptanceSuiteRoundTripPreservesAcceptanceManifest|TestUIAcceptanceAuditDetectsPermissionAccuracyPerfUsabilityAndAuditGaps|TestRenderUIAcceptanceReportSummarizesReleaseReadiness'`
- `git diff --stat`
- `git status --short`
