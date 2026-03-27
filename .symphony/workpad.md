# BIG-GO-902 Workpad

## Plan

1. Inspect existing `scripts/*.py` automation entrypoints and current `bigclaw-go` CLI commands to identify the smallest migration slice that delivers a real Go CLI path and a repeatable migration template.
2. Implement first-batch Go CLI subcommands for the selected high-frequency script layer entrypoints, keeping changes scoped to command wiring, shared helpers, and migration documentation.
3. Preserve a compatibility-layer plan by documenting legacy Python entrypoints, their Go replacements, validation commands, and remaining follow-up items.
4. Run targeted tests for the touched Go CLI packages and record exact commands plus results in the final report.
5. Commit the scoped changes and push the branch to the configured remote.

## Acceptance

- Produce an executable migration plan for moving Python script entrypoints to Go CLI subcommands.
- Land a first batch of Go CLI implementations or adaptations for selected automation entrypoints.
- Document validation commands, regression surface, branch/PR recommendation, and migration risks.

## Validation

- `go test ./cmd/bigclawctl/...`
- Additional targeted `go test` commands for any new shared package touched by the implementation.
- Manual CLI smoke checks with `go run ./cmd/bigclawctl --help` and targeted subcommand help where relevant.
