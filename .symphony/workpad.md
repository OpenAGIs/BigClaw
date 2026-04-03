# BIG-GO-1149

## Plan
- confirm the current repository Python baseline and whether the benchmark, e2e, migration, and top-level candidate entrypoints are already removed
- add or tighten regression coverage so the full `bigclaw-go/scripts` automation tree stays Python-free and the benchmark plus migration lanes keep Go-only replacement guidance
- refresh the Go CLI migration doc anywhere it still implies additional Python automation entrypoints remain in `bigclaw-go/scripts/*`
- run targeted validation for the updated regression/doc surfaces, capture exact commands and results, then commit and push the scoped change

## Acceptance
- real candidate Python entrypoint surfaces for this lane are covered by repository regression checks, including benchmark and migration paths under `bigclaw-go/scripts/*`
- Go replacement or compatibility entrypoints for benchmark and migration automation remain validated by tests/docs
- repository Python count is recorded with the exact command and outcome; if the baseline is already zero, record that literal further reduction is blocked from this checkout rather than fabricating a count drop

## Validation
- `find . -name '*.py' | wc -l`
- targeted `go test` for the updated regression package(s)
- targeted `go run ./cmd/bigclawctl automation ... --help` spot checks for benchmark and migration entrypoints
- targeted grep/doc checks if needed to confirm benchmark and migration entrypoint documentation
