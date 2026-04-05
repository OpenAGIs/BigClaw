# BIG-GO-1361 Validation

Title: `Go-only refill 1361: src/bigclaw core module removal sweep`

## Summary

This lane adds a checked-in Go/native replacement registry for the retired
`src/bigclaw` core modules that were not covered by the earlier
model/runtime-specific registry.

## Commands

### `find . -name '*.py' | wc -l`

```text
0
```

### `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1361LegacyCoreModuleReplacement(ManifestMatchesRetiredModules|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`

```text
ok  	bigclaw-go/internal/regression	1.082s
```
