# V3-stable+++ Validation (OPE-157 ~ OPE-159)

## Delivered
- `/alerts` endpoint with level/message/error_rate
- `/metrics.json` adds `bigclaw_http_error_rate` and `health_summary`
- monitor page displays Error Rate + Health cards
- rolling_5m retained and validated

## Verification
- `PYTHONPATH=src python3 -m pytest -q` -> PASS
- Runtime probe:
  - `GET /alerts` -> 200 (`level=ok`)
  - `GET /metrics.json` -> 200 (`bigclaw_http_error_rate`, `health_summary` present)
  - `GET /monitor` -> 200

## Request log sample
- GET /alerts 200
- GET /metrics.json 200
- GET /monitor 200
