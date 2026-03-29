package regression

import (
	"strings"
	"testing"
)

func TestBIGGO943RuntimeServiceLaneDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/big-go-943-runtime-service-orchestration-lane.md",
			substrings: []string{
				"src/bigclaw/runtime.py",
				"src/bigclaw/service.py",
				"src/bigclaw/scheduler.py",
				"src/bigclaw/workflow.py",
				"src/bigclaw/orchestration.py",
				"src/bigclaw/queue.py",
				"bigclaw-go/internal/worker/runtime.go",
				"bigclaw-go/internal/scheduler/scheduler.go",
				"bigclaw-go/internal/workflow/engine.go",
				"bigclaw-go/internal/workflow/orchestration.go",
				"bigclaw-go/internal/queue/queue.go",
				"Delete `tests/test_runtime.py`, `tests/test_scheduler.py`,",
			},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{
				"BIG-GO-943",
				"docs/reports/big-go-943-runtime-service-orchestration-lane.md",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
