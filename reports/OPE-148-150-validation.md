# V3-stable Productization Validation (OPE-148 ~ OPE-150)

## Scope
- OPE-148: 前端回归加固（Timeline Inspector + JSON 注入安全）
- OPE-149: 服务入口（`python -m bigclaw serve`）
- OPE-150: 运行监控（`/health` + `/metrics`）

## Implementation Summary
- Added `src/bigclaw/service.py`
  - Threading HTTP server for static pages
  - Health endpoint: `/health` (JSON)
  - Metrics endpoint: `/metrics` (Prometheus text)
  - Monitoring state: request/error counters + recent requests
- Added `src/bigclaw/__main__.py`
  - CLI entry: `python -m bigclaw serve --host --port --dir`
- Updated `src/bigclaw/run_detail.py`
  - Timeline payload now uses precomputed `timeline_json` with safe script-breakout escaping
- Added tests
  - `tests/test_service.py`
  - new regression case in `tests/test_observability.py`

## Validation Evidence

### 1) Unit tests
```bash
PYTHONPATH=src python3 -m pytest -q
```
Result: PASS (all tests)

### 2) Service entry validation
```bash
PYTHONPATH=src python3 -m bigclaw serve --host 127.0.0.1 --port 8010 --dir reports/webtest
```
Probed endpoints:
- `GET /` -> 200
- `GET /health` -> 200 + `{"status":"ok", ...}`
- `GET /metrics` -> 200 + contains `bigclaw_http_requests_total`

### 3) Runtime monitoring log sample
```text
127.0.0.1 - - [11/Mar/2026 16:45:29] "GET / HTTP/1.1" 200 -
127.0.0.1 - - [11/Mar/2026 16:45:29] "GET /health HTTP/1.1" 200 -
127.0.0.1 - - [11/Mar/2026 16:45:29] "GET /metrics HTTP/1.1" 200 -
```

## Acceptance
- [x] Front-end regression fix verified by test and local run
- [x] Service entry available via module CLI
- [x] Monitoring endpoints available and measurable
