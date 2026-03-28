# BIG-GO-923 Validation

## Commands

1. `cd bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl`
2. `cd bigclaw-go && go test ./...`
3. `git status --short`
4. `cd bigclaw-go && go test ./internal/product ./internal/testharness ./internal/regression ./cmd/bigclawctl`
5. `cd bigclaw-go && go test ./internal/product`
6. `cd bigclaw-go && go test ./cmd/bigclawctl`
7. `cd bigclaw-go && go test ./internal/governance`
8. `cd bigclaw-go && go test ./internal/bootstrap`
9. `cd bigclaw-go && go test ./internal/bootstrap`
10. `cd bigclaw-go && go test ./internal/reporting`
11. `cd bigclaw-go && go test ./internal/regression`
12. `cd bigclaw-go && go test ./internal/regression`
13. `cd bigclaw-go && go test ./internal/regression`
14. `cd bigclaw-go && go test ./internal/policy ./internal/regression`
15. `git status --short`
16. `cd bigclaw-go && go test ./internal/repo`
17. `git status --short`
18. `cd bigclaw-go && go test ./internal/repo`
19. `git status --short`
20. `cd bigclaw-go && go test ./internal/intake ./internal/risk ./internal/scheduler`
21. `git status --short`
22. `cd bigclaw-go && go test ./internal/queue`
23. `git status --short`
24. `cd bigclaw-go && go test ./internal/workflow ./internal/billing`
25. `git status --short`
26. `cd bigclaw-go && go test ./internal/observability ./internal/events`
27. `git status --short`
28. `cd bigclaw-go && go test ./internal/reporting`
29. `git status --short`
30. `cd bigclaw-go && go test ./internal/githubsync`
31. `git status --short`

## Results

1. `cd bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl`
   Result: `ok  	bigclaw-go/internal/testharness	0.392s`, `ok  	bigclaw-go/internal/regression	0.543s`, `ok  	bigclaw-go/cmd/bigclawctl	3.715s`
2. `cd bigclaw-go && go test ./...`
   Result: passed across all packages; notable tail output: `ok  	bigclaw-go/internal/queue	31.685s`, `ok  	bigclaw-go/internal/refill	6.801s`, `ok  	bigclaw-go/internal/regression	(cached)`, `ok  	bigclaw-go/internal/testharness	(cached)`, `ok  	bigclaw-go/internal/workflow	4.527s`
3. `git status --short`
   Result after implementation and before commit: modified/new files only within `.symphony/`, `bigclaw-go/`, and `reports/BIG-GO-923-validation.md` for this issue scope
4. `cd bigclaw-go && go test ./internal/product ./internal/testharness ./internal/regression ./cmd/bigclawctl`
   Result: `ok  	bigclaw-go/internal/product	0.430s`, `ok  	bigclaw-go/internal/testharness	(cached)`, `ok  	bigclaw-go/internal/regression	(cached)`, `ok  	bigclaw-go/cmd/bigclawctl	(cached)`
5. `cd bigclaw-go && go test ./internal/product`
   Result: `ok  	bigclaw-go/internal/product	0.890s`
6. `cd bigclaw-go && go test ./cmd/bigclawctl`
   Result: `ok  	bigclaw-go/cmd/bigclawctl	4.474s`
7. `cd bigclaw-go && go test ./internal/governance`
   Result: `ok  	bigclaw-go/internal/governance	(cached)`
8. `cd bigclaw-go && go test ./internal/bootstrap`
   Result: `ok  	bigclaw-go/internal/bootstrap	3.453s`
9. `cd bigclaw-go && go test ./internal/bootstrap`
   Result: `ok  	bigclaw-go/internal/bootstrap	4.031s`
10. `cd bigclaw-go && go test ./internal/reporting`
   Result: `ok  	bigclaw-go/internal/reporting	0.832s`
11. `cd bigclaw-go && go test ./internal/regression`
   Result: `ok  	bigclaw-go/internal/regression	1.324s`
12. `cd bigclaw-go && go test ./internal/regression`
   Result: `ok  	bigclaw-go/internal/regression	(cached)`
13. `cd bigclaw-go && go test ./internal/regression`
   Result: `ok  	bigclaw-go/internal/regression	(cached)`
14. `cd bigclaw-go && go test ./internal/policy ./internal/regression`
   Result: `ok  	bigclaw-go/internal/policy	(cached)`, `ok  	bigclaw-go/internal/regression	0.725s`
15. `git status --short`
   Result: `M bigclaw-go/docs/reports/pytest-harness-migration.md`, `M reports/BIG-GO-923-validation.md`, `?? bigclaw-go/internal/policy/validation_report_policy.go`, `?? bigclaw-go/internal/policy/validation_report_policy_test.go`, `?? bigclaw-go/internal/regression/live_validation_index_markdown_test.go`
16. `cd bigclaw-go && go test ./internal/repo`
   Result: `ok  	bigclaw-go/internal/repo	1.389s`
17. `git status --short`
   Result: `M bigclaw-go/docs/reports/pytest-harness-migration.md`, `M bigclaw-go/internal/repo/repo_surfaces_test.go`, `M reports/BIG-GO-923-validation.md`
18. `cd bigclaw-go && go test ./internal/repo`
   Result: `ok  	bigclaw-go/internal/repo	3.163s`
19. `git status --short`
   Result: `M bigclaw-go/docs/reports/pytest-harness-migration.md`, `M bigclaw-go/internal/repo/board.go`, `M bigclaw-go/internal/repo/repo_surfaces_test.go`, `M reports/BIG-GO-923-validation.md`
20. `cd bigclaw-go && go test ./internal/intake ./internal/risk ./internal/scheduler`
   Result: `ok  	bigclaw-go/internal/intake	(cached)`, `ok  	bigclaw-go/internal/risk	(cached)`, `ok  	bigclaw-go/internal/scheduler	(cached)`
21. `git status --short`
   Result: `M bigclaw-go/docs/reports/pytest-harness-migration.md`, `M reports/BIG-GO-923-validation.md`
22. `cd bigclaw-go && go test ./internal/queue`
   Result: `ok  	bigclaw-go/internal/queue	24.920s`
23. `git status --short`
   Result: `M bigclaw-go/internal/queue/file_queue.go`, `M bigclaw-go/internal/queue/file_queue_test.go`
24. `cd bigclaw-go && go test ./internal/workflow ./internal/billing`
   Result: `ok  	bigclaw-go/internal/workflow	(cached)`, `ok  	bigclaw-go/internal/billing	(cached)`
25. `git status --short`
   Result: `M bigclaw-go/docs/reports/pytest-harness-migration.md`, `M reports/BIG-GO-923-validation.md`
26. `cd bigclaw-go && go test ./internal/observability ./internal/events`
   Result: `ok  	bigclaw-go/internal/observability	(cached)`, `ok  	bigclaw-go/internal/events	(cached)`
27. `git status --short`
   Result: `M bigclaw-go/docs/reports/pytest-harness-migration.md`, `M reports/BIG-GO-923-validation.md`
28. `cd bigclaw-go && go test ./internal/reporting`
   Result: `ok  	bigclaw-go/internal/reporting	(cached)`
29. `git status --short`
   Result: `M bigclaw-go/docs/reports/pytest-harness-migration.md`, `M reports/BIG-GO-923-validation.md`
30. `cd bigclaw-go && go test ./internal/githubsync`
   Result: `ok  	bigclaw-go/internal/githubsync	(cached)`
31. `git status --short`
   Result: `M bigclaw-go/docs/reports/pytest-harness-migration.md`, `M reports/BIG-GO-923-validation.md`
