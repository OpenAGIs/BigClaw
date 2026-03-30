# Long-Duration Soak Report

## Scope

- Run date: 2026-03-13
- Command: `go run ./cmd/bigclawctl automation benchmark soak-local --autostart --count 2000 --workers 24 --timeout-seconds 480 --report-path docs/reports/soak-local-2000x24.json`
- Goal: extend the earlier `1k+` burst proof into a longer local soak window for epic closure readiness.

## Result

- `2000 tasks x 24 submit workers`: `219.167s` elapsed
- Throughput: `9.125 tasks/s`
- Terminal outcome: `2000 succeeded`, `0 failed`
- Sample traces preserved `trace_id`, emitted `scheduler.routed`, and reached `task.completed`

## Meaning

This run adds a same-day longer-duration local soak to the existing benchmark package. On its own it is still local evidence rather than production certification, but it now serves as the ceiling lane for `docs/reports/capacity-certification-matrix.json`.

Within the current repo-native certification slice, `2000x24` is treated as:

- the checked-in local ceiling envelope,
- a `0 failure` sustained run,
- a saturation check against the `1000x24` lane, where throughput drops only from `9.607` to `9.125 tasks/s` (`5.02%`).

## Artifact

- `docs/reports/soak-local-2000x24.json`
- `docs/reports/capacity-certification-matrix.json`
- `docs/reports/capacity-certification-report.md`
