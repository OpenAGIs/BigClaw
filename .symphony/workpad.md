# BIGCLAW-175 Workpad

## Plan
1. Inspect the existing Go worker/control and Python queue control-center paths to find the narrowest extension points for host-profile and node-capacity baseline data.
2. Add scoped baseline structures for host / node / pool capacity on the ClawHost side, using existing runtime node IDs and executor capabilities to aggregate parallel limits.
3. Surface the aggregated host-profile parallel capacity through the control-facing snapshot/reporting path and extend the BigClaw queue control center so each host type shows supported parallel capacity.
4. Add focused Go and Python tests proving capacity aggregation and rendered control-center output.
5. Run targeted validation, then commit and push the issue-scoped branch changes.

## Acceptance
- Add structures that can describe host, node, and pool capacity.
- Control center displays the parallel capacity for each host type.
- Tests prove the parallel capacity aggregation works.

## Validation
- Run targeted Go and Python tests for the new capacity model and control center surface.
- Planned commands:
  - `cd bigclaw-go && go test ./internal/worker ./internal/api ./cmd/bigclawd`
- Record exact commands and results in the final report.
- Verify the issue branch is committed and pushed to `origin`.
