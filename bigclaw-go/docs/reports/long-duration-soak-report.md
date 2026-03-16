# Long-Duration Soak Report

## Scope

- Run date: 2026-03-13
- Command: `python3 scripts/benchmark/soak_local.py --autostart --count 2000 --workers 24 --timeout-seconds 480 --report-path docs/reports/soak-local-2000x24.json`
- Goal: extend the earlier `1k+` burst proof into a longer local soak window for epic closure readiness.

## Result

- `2000 tasks x 24 submit workers`: `219.167s` elapsed
- Throughput: `9.125 tasks/s`
- Terminal outcome: `2000 succeeded`, `0 failed`
- Sample traces preserved `trace_id`, emitted `scheduler.routed`, and reached `task.completed`

## Meaning

This run adds a same-day longer-duration local soak to the existing benchmark package. It is still local evidence rather than production certification, but it materially reduces the remaining closure gap around sustained control-plane stability.

## Artifact

- `docs/reports/soak-local-2000x24.json`

## Follow-Up Digest

- `docs/reports/scale-validation-follow-up-digest.md` consolidates this local `2000x24` soak proof with the remaining queue and capacity-certification follow-up caveats.
