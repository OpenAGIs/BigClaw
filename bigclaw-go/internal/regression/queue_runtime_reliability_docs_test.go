package regression

import (
	"os"
	"strings"
	"testing"
)

func TestQueueRuntimeReliabilityDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)

	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/queue-reliability-report.md",
			substrings: []string{
				"# Queue Reliability Report",
				"internal/queue/memory_queue_test.go",
				"internal/queue/file_queue_test.go",
				"internal/queue/sqlite_queue_test.go",
				"internal/api/server_test.go",
				"docs/reports/live-validation-summary.json",
				"GET /deadletters",
				"POST /deadletters/{id}/replay",
				"`1k` and `10k` task no-duplicate-consumption lanes",
			},
		},
		{
			path: "docs/reports/lease-recovery-report.md",
			substrings: []string{
				"# Lease Recovery Report",
				"internal/queue/sqlite_queue_test.go::TestSQLiteQueueLeaseExpiresAndCanBeReacquired",
				"internal/queue/sqlite_queue_test.go::TestSQLiteQueueProcesses1000TasksWithoutDuplicateLease",
				"internal/worker/runtime.go",
				"internal/worker/runtime_test.go",
				"Expired SQLite leases become available for reacquisition by a different worker.",
				"`database is locked`",
				"`1k` and `10k` task lanes",
			},
		},
		{
			path: "docs/reports/state-machine-validation-report.md",
			substrings: []string{
				"# State Machine Validation Report",
				"internal/domain/state_machine_test.go",
				"internal/api/server_test.go",
				"internal/worker/runtime_test.go",
				"`queued -> leased` is accepted",
				"`queued -> succeeded` are rejected",
				"queue -> lease -> start -> terminal order",
			},
		},
	}

	for _, tc := range cases {
		body := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(body, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}

	for _, relative := range []string{
		"internal/queue/memory_queue_test.go",
		"internal/queue/file_queue_test.go",
		"internal/queue/sqlite_queue_test.go",
		"internal/api/server_test.go",
		"internal/domain/state_machine_test.go",
		"internal/worker/runtime.go",
		"internal/worker/runtime_test.go",
		"docs/reports/live-validation-summary.json",
	} {
		if _, err := os.Stat(resolveRepoPath(repoRoot, relative)); err != nil {
			t.Fatalf("expected referenced queue/runtime evidence %q to exist: %v", relative, err)
		}
	}
}
