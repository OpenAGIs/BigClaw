# BIG-GO-1483 Validation

Issue: `BIG-GO-1483`

## Summary

The current `BIG-GO-1483` branch baseline is already Python-free for
`bigclaw-go/scripts` and for the repository as a whole. This lane therefore
landed the remaining active caller/doc cleanup by removing deleted
`bigclaw-go/scripts/*.py` path guidance from the live migration plan and
keeping the supported surface on the Go CLI replacements.

## Before / After Evidence

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go/scripts -type f -name '*.py' | sort
```

Before:

```text
<no output>
```

After:

```text
<no output>
```

Count command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go/scripts -type f -name '*.py' | wc -l
```

Count result:

```text
0
```

Caller reference command:

```bash
rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!local-issues.json' --glob '!bigclaw-go/internal/regression/**' --glob '!.symphony/**' 'bigclaw-go/scripts/.*\.py' README.md docs scripts .github bigclaw-go | sort | wc -l
```

Caller reference result:

```text
0
```

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483 -type f -name '*.py' | sort
```

Before:

```text
<no output>
```

After:

```text
<no output>
```

Count command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483 -type f -name '*.py' | wc -l
```

Count result:

```text
0
```

## Targeted Validation

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160(MigrationDocsListGoReplacements|CandidatePythonFilesRemainDeleted)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.314s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go && go test -count=1 ./internal/regression -run 'TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestE2EMigrationDocListsOnlyActiveEntrypoints'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.182s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160(MigrationDocsListGoReplacements|CandidatePythonFilesRemainDeleted)$|TestBIGGO1483(MigrationPlanListsOnlyGoOrShellBigClawScriptEntrypoints|LaneReportCapturesCallerCutoverState)$|TestE2EMigrationDocListsOnlyActiveEntrypoints|TestE2EScriptDirectoryStaysPythonFree'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.284s
```

## Blocker

The issue requirement to reduce the actual repository Python file count is not
achievable from this branch baseline because the checked-out repository already
contains zero tracked `.py` files before the change. This lane is limited to
cleaning the remaining active Go-migration guidance and recording exact
baseline evidence.
