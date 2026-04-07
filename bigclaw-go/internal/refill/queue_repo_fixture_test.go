package refill

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestParallelIssueQueueRepoFixtureSelectionStaysAligned(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	queuePath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "docs", "parallel-refill-queue.json")

	queue, err := LoadQueue(queuePath)
	if err != nil {
		t.Fatalf("load repo fixture queue: %v", err)
	}

	identifiers := queue.IssueIdentifiers()
	if got := queue.ProjectSlug(); got != "53e33900c67e" {
		t.Fatalf("project slug = %q, want %q", got, "53e33900c67e")
	}
	if got := queue.TargetInProgress(); got != 2 {
		t.Fatalf("target in progress = %d, want 2", got)
	}
	if len(identifiers) == 0 {
		t.Fatal("expected queue fixture identifiers")
	}
	seen := make(map[string]struct{}, len(identifiers))
	for _, identifier := range identifiers {
		if _, exists := seen[identifier]; exists {
			t.Fatalf("duplicate identifier in queue fixture: %s", identifier)
		}
		seen[identifier] = struct{}{}
	}
	if got := queue.IssueOrder(); len(got) < 6 ||
		got[0] != "BIG-GOM-301" ||
		got[1] != "BIG-GOM-302" ||
		got[2] != "BIG-GOM-303" ||
		got[3] != "BIG-GOM-304" {
		t.Fatalf("unexpected queue order prefix: %+v", got[:minInt(len(got), 6)])
	}

	issueStates := IssueStateMap([]TrackedIssue{
		{Identifier: "BIG-GOM-301", StateName: "Todo"},
		{Identifier: "BIG-GOM-302", StateName: "Todo"},
		{Identifier: "BIG-GOM-303", StateName: "Todo"},
		{Identifier: "BIG-GOM-304", StateName: "Todo"},
		{Identifier: "BIG-GOM-305", StateName: "Backlog"},
		{Identifier: "BIG-GOM-306", StateName: "Backlog"},
	})
	candidates := queue.SelectCandidates(map[string]struct{}{}, issueStates, nil)
	if len(candidates) != 2 || candidates[0] != "BIG-GOM-301" || candidates[1] != "BIG-GOM-302" {
		t.Fatalf("unexpected refill candidates: %+v", candidates)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
