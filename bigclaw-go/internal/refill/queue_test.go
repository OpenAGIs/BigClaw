package refill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubTempFile struct {
	name      string
	chmodErr  error
	writeErr  error
	closeErr  error
	writes    [][]byte
	closed    bool
}

func (f *stubTempFile) Name() string { return f.name }

func (f *stubTempFile) Chmod(mode os.FileMode) error { return f.chmodErr }

func (f *stubTempFile) Write(body []byte) (int, error) {
	f.writes = append(f.writes, append([]byte{}, body...))
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return len(body), nil
}

func (f *stubTempFile) Close() error {
	f.closed = true
	return f.closeErr
}

func TestLoadQueueNormalizesAbsolutePathAndAccessors(t *testing.T) {
	queuePath := filepath.Join(t.TempDir(), "queue.json")
	if err := os.WriteFile(queuePath, []byte(`{
  "project": {
    "slug_id": "project-slug"
  },
  "policy": {
    "activate_state_id": "state-123"
  },
  "issue_order": [
    "BIG-PAR-408"
  ],
  "issues": [
    {
      "identifier": "BIG-PAR-408",
      "title": "Add refill queue load/save helper coverage",
      "track": "Go Mainline Follow-ups",
      "status": "Todo"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write queue fixture: %v", err)
	}

	queue, err := LoadQueue(queuePath)
	if err != nil {
		t.Fatalf("load queue: %v", err)
	}
	if queue.queuePath != queuePath {
		t.Fatalf("expected absolute queue path to persist, got %q", queue.queuePath)
	}
	if got := queue.ProjectSlug(); got != "project-slug" {
		t.Fatalf("expected project slug accessor, got %q", got)
	}
	if got := queue.ActivateStateID(); got != "state-123" {
		t.Fatalf("expected activate state id accessor, got %q", got)
	}
	if got := queue.ActivateStateName(); got != "In Progress" {
		t.Fatalf("expected activate state name fallback, got %q", got)
	}
}

func TestLoadQueueMissingAndInvalidPayloads(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-queue.json")
	if _, err := LoadQueue(missingPath); err == nil {
		t.Fatal("expected missing queue file to fail")
	}

	invalidPath := filepath.Join(t.TempDir(), "invalid-queue.json")
	if err := os.WriteFile(invalidPath, []byte(`{"issues":`), 0o644); err != nil {
		t.Fatalf("write invalid queue fixture: %v", err)
	}
	if _, err := LoadQueue(invalidPath); err == nil || !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatalf("expected invalid queue payload to fail, got %v", err)
	}
}

func TestIssueStateMapRecordsIdentifiers(t *testing.T) {
	issues := []TrackedIssue{
		{Identifier: "BIG-GOM-301", StateName: "Todo"},
		{Identifier: "BIG-GOM-302", StateName: "Todo"},
	}
	stateMap := IssueStateMap(issues)
	if stateMap["BIG-GOM-301"] != "Todo" || stateMap["BIG-GOM-302"] != "Todo" {
		t.Fatalf("unexpected state map: %+v", stateMap)
	}
}

func TestParallelIssueQueueRunnableCountTreatsFullyDoneQueueAsDrained(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-001", "BIG-PAR-002"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-001", Status: "Done"},
				{Identifier: "BIG-PAR-002", Status: "Done"},
			},
		},
	}
	if got := queue.RunnableCount(); got != 0 {
		t.Fatalf("expected drained runnable count, got %d", got)
	}
}

func TestParallelIssueQueueRunnableCountReturnsZeroWhenOrderMissing(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-414", Status: "Todo"},
			},
		},
	}
	if got := queue.RunnableCount(); got != 0 {
		t.Fatalf("expected zero runnable count when issue order is empty, got %d", got)
	}
}

func TestParallelIssueQueueRunnableCountForStatesReturnsZeroWhenOrderMissing(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-424", Status: "Todo"},
			},
		},
	}

	liveStates := map[string]string{
		"BIG-PAR-424": "Done",
		"BIG-PAR-999": "In Progress",
	}
	if got := queue.RunnableCountForStates(liveStates); got != 0 {
		t.Fatalf("expected zero runnable count when issue order is empty, got %d", got)
	}
}

func TestParallelIssueQueueRunnableCountDoesNotDrainWhenMetadataMissing(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-001", "BIG-PAR-002"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-001", Status: "Done"},
			},
		},
	}
	if got := queue.RunnableCount(); got == 0 {
		t.Fatalf("expected runnable count for missing metadata, got %d", got)
	}
}

func TestParallelIssueQueueRunnableCountForStatesPrefersLiveStateMap(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-224", "BIG-PAR-225", "BIG-PAR-226", "BIG-PAR-227"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-224", Status: "In Progress"},
				{Identifier: "BIG-PAR-225", Status: "In Progress"},
				{Identifier: "BIG-PAR-226", Status: "Todo"},
				{Identifier: "BIG-PAR-227", Status: "Todo"},
			},
		},
	}
	liveStates := map[string]string{
		"BIG-PAR-224": "Done",
		"BIG-PAR-225": "Done",
		"BIG-PAR-226": "Done",
		"BIG-PAR-227": "Done",
	}
	if got := queue.RunnableCountForStates(liveStates); got != 0 {
		t.Fatalf("expected live state map to mark queue drained, got %d", got)
	}
}

func TestParallelIssueQueueRunnableCountForStatesDoesNotDrainWhenMetadataMissing(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-411", "BIG-PAR-412"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-411", Status: "Done"},
			},
		},
	}
	liveStates := map[string]string{
		"BIG-PAR-411": "Done",
	}
	if got := queue.RunnableCountForStates(liveStates); got == 0 {
		t.Fatalf("expected missing queue metadata to keep runnable count non-zero, got %d", got)
	}
}

func TestParallelIssueQueueRunnableCountForStatesTrimsOverridesAndIgnoresBlanks(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-419", "BIG-PAR-420", "BIG-PAR-421"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-419", Status: "Done"},
				{Identifier: "BIG-PAR-420", Status: "Backlog"},
				{Identifier: "BIG-PAR-421", Status: "Done"},
			},
		},
	}

	liveStates := map[string]string{
		"":            "Done",
		"BIG-PAR-419": "   ",
		"BIG-PAR-420": " Done. ",
		"BIG-PAR-421": " In Progress ",
	}
	if got := queue.RunnableCountForStates(liveStates); got != 1 {
		t.Fatalf("expected trimmed live state overrides to leave one runnable issue, got %d", got)
	}
}

func TestParallelIssueQueueRefreshRecentBatchesFromStates(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-241", "BIG-PAR-242", "BIG-PAR-243"},
		},
	}
	queue.payload.Policy.RefillStates = []string{"Todo", "Backlog"}
	states := map[string]string{
		"BIG-PAR-241": "In Progress",
		"BIG-PAR-242": "Done",
		"BIG-PAR-243": "Backlog",
	}
	updated := queue.RefreshRecentBatchesFromStates(states)
	if !updated {
		t.Fatalf("expected recent batches to report a change")
	}
	if !equalStringSlices(queue.payload.RecentBatches.Active, []string{"BIG-PAR-241"}) {
		t.Fatalf("unexpected active batches: %v", queue.payload.RecentBatches.Active)
	}
	if !equalStringSlices(queue.payload.RecentBatches.Completed, []string{"BIG-PAR-242"}) {
		t.Fatalf("unexpected completed batches: %v", queue.payload.RecentBatches.Completed)
	}
	if !equalStringSlices(queue.payload.RecentBatches.Standby, []string{"BIG-PAR-243"}) {
		t.Fatalf("unexpected standby batches: %v", queue.payload.RecentBatches.Standby)
	}
	if queue.RefreshRecentBatchesFromStates(states) {
		t.Fatalf("expected no change when data is already fresh")
	}
}

func TestIssueRecordStateMapTrimsStatusesAndSkipsBlankIdentifiers(t *testing.T) {
	states := issueRecordStateMap([]IssueRecord{
		{Identifier: "", Status: "Done"},
		{Identifier: "BIG-PAR-399", Status: " In Progress "},
		{Identifier: "BIG-PAR-400", Status: "Done. "},
	})

	if len(states) != 2 {
		t.Fatalf("expected only non-blank identifiers, got %+v", states)
	}
	if states["BIG-PAR-399"] != "In Progress" {
		t.Fatalf("expected trimmed in-progress status, got %+v", states)
	}
	if states["BIG-PAR-400"] != "Done." {
		t.Fatalf("expected status trimming to preserve content, got %+v", states)
	}
}

func TestEqualStringSlicesDetectsLengthAndOrderChanges(t *testing.T) {
	if !equalStringSlices([]string{"BIG-PAR-399", "BIG-PAR-400"}, []string{"BIG-PAR-399", "BIG-PAR-400"}) {
		t.Fatal("expected identical slices to compare equal")
	}
	if equalStringSlices([]string{"BIG-PAR-399"}, []string{"BIG-PAR-399", "BIG-PAR-400"}) {
		t.Fatal("expected length mismatch to compare false")
	}
	if equalStringSlices([]string{"BIG-PAR-399", "BIG-PAR-400"}, []string{"BIG-PAR-400", "BIG-PAR-399"}) {
		t.Fatal("expected order mismatch to compare false")
	}
}

func TestParallelIssueQueueRefreshRecentBatchesFromStatesEmptyOrPunctuatedStates(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			Policy: struct {
				TargetInProgress  int      `json:"target_in_progress"`
				ActivateStateID   string   `json:"activate_state_id"`
				ActivateStateName string   `json:"activate_state_name"`
				RefillStates      []string `json:"refill_states"`
				BlockedReason     string   `json:"blocked_reason,omitempty"`
			}{
				ActivateStateName: "In Progress",
				RefillStates:      []string{"Todo", "Backlog"},
			},
			IssueOrder: []string{"BIG-PAR-399", "BIG-PAR-400", "BIG-PAR-401", "BIG-PAR-402"},
		},
	}

	if queue.RefreshRecentBatchesFromStates(nil) {
		t.Fatal("expected empty state map to report no refresh")
	}

	updated := queue.RefreshRecentBatchesFromStates(map[string]string{
		"BIG-PAR-399": "In Progress",
		"BIG-PAR-400": "Done.",
		"BIG-PAR-401": " Backlog ",
		"BIG-PAR-402": "",
	})
	if !updated {
		t.Fatal("expected punctuated states to refresh recent batches")
	}
	if !equalStringSlices(queue.payload.RecentBatches.Active, []string{"BIG-PAR-399"}) {
		t.Fatalf("unexpected active batch set: %+v", queue.payload.RecentBatches.Active)
	}
	if !equalStringSlices(queue.payload.RecentBatches.Completed, []string{"BIG-PAR-400"}) {
		t.Fatalf("unexpected completed batch set: %+v", queue.payload.RecentBatches.Completed)
	}
	if !equalStringSlices(queue.payload.RecentBatches.Standby, []string{"BIG-PAR-401"}) {
		t.Fatalf("unexpected standby batch set: %+v", queue.payload.RecentBatches.Standby)
	}
}

func TestParallelIssueQueueSaveFailsWhenParentPathIsAFile(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "queue-parent")
	if err := os.WriteFile(parentFile, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("write parent file fixture: %v", err)
	}

	queue := &ParallelIssueQueue{
		queuePath: filepath.Join(parentFile, "queue.json"),
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-418"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-418", Title: "Add refill save-parent failure coverage", Status: "In Progress"},
			},
		},
	}

	if err := queue.Save(); err == nil {
		t.Fatal("expected save to fail when parent path is a file")
	}
}

func TestParallelIssueQueueSetRecentBatchMovesClearsAndValidatesIdentifiers(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-419", "BIG-PAR-420", "BIG-PAR-421"},
			RecentBatches: struct {
				Completed []string `json:"completed"`
				Active    []string `json:"active"`
				Standby   []string `json:"standby"`
			}{
				Completed: []string{"BIG-PAR-420"},
				Active:    []string{"BIG-PAR-419"},
				Standby:   []string{"BIG-PAR-421"},
			},
		},
	}

	changed, err := queue.SetRecentBatch("active", "BIG-PAR-420")
	if err != nil {
		t.Fatalf("move issue to active batch: %v", err)
	}
	if !changed {
		t.Fatal("expected moving issue between recent batches to report a change")
	}
	if !equalStringSlices(queue.payload.RecentBatches.Completed, []string{}) {
		t.Fatalf("expected completed recent batch to be cleared, got %+v", queue.payload.RecentBatches.Completed)
	}
	if !equalStringSlices(queue.payload.RecentBatches.Active, []string{"BIG-PAR-419", "BIG-PAR-420"}) {
		t.Fatalf("expected active recent batch to preserve queue order, got %+v", queue.payload.RecentBatches.Active)
	}

	changed, err = queue.SetRecentBatch("none", "BIG-PAR-420")
	if err != nil {
		t.Fatalf("clear issue recent batch: %v", err)
	}
	if !changed {
		t.Fatal("expected clearing issue from recent batches to report a change")
	}
	if !equalStringSlices(queue.payload.RecentBatches.Active, []string{"BIG-PAR-419"}) {
		t.Fatalf("expected active recent batch to remove cleared identifier, got %+v", queue.payload.RecentBatches.Active)
	}

	if changed, err := queue.SetRecentBatch("invalid", "BIG-PAR-420"); err == nil || changed {
		t.Fatalf("expected invalid batch name to fail, got changed=%v err=%v", changed, err)
	}
	if changed, err := queue.SetRecentBatch("standby", "   "); err == nil || changed {
		t.Fatalf("expected blank identifier to fail, got changed=%v err=%v", changed, err)
	}
}

func TestQueueIdentifierHelpersNormalizeRemovalUniquenessAndOrder(t *testing.T) {
	items, removed := withoutIdentifier([]string{" BIG-PAR-402 ", "BIG-PAR-401", "big-par-402"}, "big-par-402")
	if !removed {
		t.Fatal("expected identifier removal to report true")
	}
	if !equalStringSlices(items, []string{"BIG-PAR-401"}) {
		t.Fatalf("unexpected items after identifier removal: %+v", items)
	}

	unchanged, removed := withoutIdentifier([]string{"BIG-PAR-401"}, "BIG-PAR-999")
	if removed {
		t.Fatal("expected missing identifier removal to report false")
	}
	if !equalStringSlices(unchanged, []string{"BIG-PAR-401"}) {
		t.Fatalf("expected unmatched removal to preserve items, got %+v", unchanged)
	}

	unique := uniqueIdentifiers([]string{" BIG-PAR-402 ", "", "big-par-402", "BIG-PAR-401", "big-par-401", "BIG-PAR-403"})
	if !equalStringSlices(unique, []string{"BIG-PAR-402", "BIG-PAR-401", "BIG-PAR-403"}) {
		t.Fatalf("unexpected unique identifiers: %+v", unique)
	}

	ordered := orderByIssueOrder([]string{"BIG-PAR-403", "BIG-PAR-405", "big-par-401", "BIG-PAR-404"}, []string{"BIG-PAR-401", "BIG-PAR-403"})
	if !equalStringSlices(ordered, []string{"big-par-401", "BIG-PAR-403", "BIG-PAR-404", "BIG-PAR-405"}) {
		t.Fatalf("unexpected identifier order: %+v", ordered)
	}
}

func TestAppendIdentifierOnceRejectsBlankAndDuplicateIdentifiers(t *testing.T) {
	items := []string{"BIG-PAR-411"}
	if appendIdentifierOnce(&items, "   ") {
		t.Fatal("expected blank identifier append to report false")
	}
	if appendIdentifierOnce(&items, " big-par-411 ") {
		t.Fatal("expected case-insensitive duplicate append to report false")
	}
	if !appendIdentifierOnce(&items, "BIG-PAR-412") {
		t.Fatal("expected unique identifier append to report true")
	}
	if !equalStringSlices(items, []string{"BIG-PAR-411", "BIG-PAR-412"}) {
		t.Fatalf("unexpected appended identifiers: %+v", items)
	}
}

func TestQueueSelectionAndAccessorHelpers(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			Policy: struct {
				TargetInProgress  int      `json:"target_in_progress"`
				ActivateStateID   string   `json:"activate_state_id"`
				ActivateStateName string   `json:"activate_state_name"`
				RefillStates      []string `json:"refill_states"`
				BlockedReason     string   `json:"blocked_reason,omitempty"`
			}{
				TargetInProgress:  2,
				ActivateStateName: "In Progress",
				RefillStates:      []string{"Todo", "Backlog"},
			},
			IssueOrder: []string{"BIG-PAR-403", "BIG-PAR-404", "BIG-PAR-405"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-403", Status: "Todo"},
				{Identifier: "BIG-PAR-404", Status: "Backlog"},
				{Identifier: "BIG-PAR-405", Status: "Done"},
			},
		},
	}

	if got := queue.IssueIdentifiers(); !equalStringSlices(got, []string{"BIG-PAR-403", "BIG-PAR-404", "BIG-PAR-405"}) {
		t.Fatalf("unexpected IssueIdentifiers result: %+v", got)
	}

	active := SortedActive([]TrackedIssue{
		{Identifier: "BIG-PAR-405", StateName: "Blocked"},
		{Identifier: "BIG-PAR-404", StateName: "In Progress"},
		{Identifier: "BIG-PAR-403", StateName: "In Progress"},
	})
	if !equalStringSlices(active, []string{"BIG-PAR-403", "BIG-PAR-404"}) {
		t.Fatalf("unexpected SortedActive result: %+v", active)
	}

	candidates := queue.SelectCandidates(
		map[string]struct{}{"BIG-PAR-404": {}},
		map[string]string{
			"BIG-PAR-403": " Todo ",
			"BIG-PAR-404": "Backlog",
			"BIG-PAR-405": "Done",
		},
		nil,
	)
	if !equalStringSlices(candidates, []string{"BIG-PAR-403"}) {
		t.Fatalf("expected only remaining runnable candidate, got %+v", candidates)
	}

	candidates = queue.SelectCandidates(
		map[string]struct{}{"BIG-PAR-403": {}},
		map[string]string{
			"BIG-PAR-403": "Todo",
			"BIG-PAR-404": "Backlog",
			"BIG-PAR-405": "Done",
		},
		nil,
	)
	if !equalStringSlices(candidates, []string{"BIG-PAR-404"}) {
		t.Fatalf("expected active identifiers to be skipped from candidates, got %+v", candidates)
	}

	target := 3
	candidates = queue.SelectCandidates(
		map[string]struct{}{},
		map[string]string{
			"BIG-PAR-403": "Todo",
			"BIG-PAR-404": "Backlog",
			"BIG-PAR-405": "Done",
		},
		&target,
	)
	if !equalStringSlices(candidates, []string{"BIG-PAR-403", "BIG-PAR-404"}) {
		t.Fatalf("expected override target to return two refill candidates, got %+v", candidates)
	}

	noneNeeded := queue.SelectCandidates(
		map[string]struct{}{"BIG-PAR-403": {}, "BIG-PAR-404": {}},
		map[string]string{"BIG-PAR-403": "Todo", "BIG-PAR-404": "Backlog"},
		nil,
	)
	if len(noneNeeded) != 0 {
		t.Fatalf("expected no candidates when target already met, got %+v", noneNeeded)
	}
}

func TestQueueStatusSyncAndNormalizationHelpers(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			Issues: []IssueRecord{
				{Identifier: "", Status: "Todo"},
				{Identifier: "BIG-PAR-405", Status: "Todo"},
				{Identifier: "BIG-PAR-406", Status: "Done"},
			},
		},
	}

	updated := queue.SyncStatusFromStates(map[string]string{
		"":            "In Progress",
		"BIG-PAR-405": " In Progress ",
		"BIG-PAR-406": "   ",
		"BIG-PAR-999": "Todo",
	})
	if updated != 1 {
		t.Fatalf("expected one synced status update, got %d", updated)
	}
	if queue.payload.Issues[1].Status != "In Progress" {
		t.Fatalf("expected synced status trim, got %+v", queue.payload.Issues)
	}
	if queue.payload.Issues[2].Status != "Done" {
		t.Fatalf("expected blank incoming status to be ignored, got %+v", queue.payload.Issues)
	}
	if queue.SyncStatusFromStates(map[string]string{"BIG-PAR-405": "In Progress"}) != 0 {
		t.Fatal("expected identical status sync to be a no-op")
	}

	if got := statusNormalize(" Done. "); got != "done" {
		t.Fatalf("expected status normalization to trim and lowercase, got %q", got)
	}
	if got := statusNormalize("   "); got != "" {
		t.Fatalf("expected blank status normalization to return empty string, got %q", got)
	}
	if !isTerminalStatus("Done.") {
		t.Fatal("expected done status to be terminal")
	}
	if isTerminalStatus("   ") {
		t.Fatal("expected blank status to be non-terminal")
	}
	if isTerminalStatus("In Progress") {
		t.Fatal("expected in-progress status to be non-terminal")
	}
	if isTerminalState("done") != true {
		t.Fatal("expected isTerminalState to proxy terminal statuses")
	}
}

func TestParallelIssueQueueSavePreservesBlockedReasonAndRecentBatches(t *testing.T) {
	queuePath := filepath.Join(t.TempDir(), "queue.json")
	queue := &ParallelIssueQueue{
		queuePath: queuePath,
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-230"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-230", Status: "Todo"},
			},
		},
	}
	queue.payload.Policy.BlockedReason = "local tracker owns live issue state"
	queue.payload.RecentBatches.Completed = []string{"BIG-PAR-229"}
	queue.payload.RecentBatches.Active = []string{"BIG-PAR-230"}
	queue.payload.RecentBatches.Standby = []string{}

	if err := queue.Save(); err != nil {
		t.Fatalf("save queue: %v", err)
	}

	body, err := os.ReadFile(queuePath)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"blocked_reason": "local tracker owns live issue state"`) {
		t.Fatalf("expected blocked_reason to persist, got %s", text)
	}
	if !strings.Contains(text, `"recent_batches"`) {
		t.Fatalf("expected recent_batches to persist, got %s", text)
	}
	if !strings.Contains(text, `"completed": [`) || !strings.Contains(text, `"active": [`) || !strings.Contains(text, `"standby": []`) {
		t.Fatalf("expected recent_batches fields to persist, got %s", text)
	}
}

func TestParallelIssueQueueSaveCreatesParentDirectoryAndDoesNotEscapeHTML(t *testing.T) {
	queuePath := filepath.Join(t.TempDir(), "nested", "queue.json")
	queue := &ParallelIssueQueue{
		queuePath: queuePath,
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-408"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-408", Title: "arrow -> literal", Track: "Go Mainline Follow-ups", Status: "Todo"},
			},
		},
	}

	if err := queue.Save(); err != nil {
		t.Fatalf("save queue in nested directory: %v", err)
	}

	body, err := os.ReadFile(queuePath)
	if err != nil {
		t.Fatalf("read saved queue: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "arrow -> literal") {
		t.Fatalf("expected raw arrow token to persist, got %s", text)
	}
	if strings.Contains(text, `\u003e`) {
		t.Fatalf("expected html escaping to stay disabled, got %s", text)
	}
}

func TestParallelIssueQueueSaveFailsWhenTargetPathIsDirectory(t *testing.T) {
	queueDir := filepath.Join(t.TempDir(), "queue-dir")
	if err := os.MkdirAll(queueDir, 0o755); err != nil {
		t.Fatalf("mkdir queue dir fixture: %v", err)
	}

	queue := &ParallelIssueQueue{
		queuePath: queueDir,
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-414"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-414", Title: "save edge coverage", Track: "Go Mainline Follow-ups", Status: "Todo"},
			},
		},
	}

	if err := queue.Save(); err == nil {
		t.Fatal("expected queue save to fail when target path is a directory")
	}
}

func TestParallelIssueQueueSavePropagatesCloseAndRenameFailures(t *testing.T) {
	queue := &ParallelIssueQueue{
		queuePath: filepath.Join(t.TempDir(), "queue.json"),
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-425"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-425", Title: "Make refill queue save failures testable", Track: "Go Mainline Follow-ups", Status: "In Progress"},
			},
		},
	}

	originalCreateTemp := queueCreateTemp
	originalRename := queueRename
	t.Cleanup(func() {
		queueCreateTemp = originalCreateTemp
		queueRename = originalRename
	})

	closeErr := errors.New("close temp queue file")
	queueCreateTemp = func(dir string, pattern string) (tempFile, error) {
		return &stubTempFile{name: filepath.Join(dir, "queue-close.tmp"), closeErr: closeErr}, nil
	}
	queueRename = func(oldPath string, newPath string) error {
		t.Fatal("did not expect rename after close failure")
		return nil
	}
	if err := queue.Save(); !errors.Is(err, closeErr) {
		t.Fatalf("expected close failure to propagate, got %v", err)
	}

	renameErr := errors.New("rename queue file")
	queueCreateTemp = func(dir string, pattern string) (tempFile, error) {
		return &stubTempFile{name: filepath.Join(dir, "queue-rename.tmp")}, nil
	}
	queueRename = func(oldPath string, newPath string) error {
		if oldPath != filepath.Join(filepath.Dir(queue.queuePath), "queue-rename.tmp") || newPath != queue.queuePath {
			t.Fatalf("unexpected rename paths old=%q new=%q", oldPath, newPath)
		}
		return renameErr
	}
	if err := queue.Save(); !errors.Is(err, renameErr) {
		t.Fatalf("expected rename failure to propagate, got %v", err)
	}
}

func TestParallelIssueQueueSavePropagatesChmodFailure(t *testing.T) {
	queue := &ParallelIssueQueue{
		queuePath: filepath.Join(t.TempDir(), "queue.json"),
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-427"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-427", Title: "Add refill temp-save branch coverage", Track: "Go Mainline Follow-ups", Status: "In Progress"},
			},
		},
	}

	originalCreateTemp := queueCreateTemp
	originalRename := queueRename
	t.Cleanup(func() {
		queueCreateTemp = originalCreateTemp
		queueRename = originalRename
	})

	chmodErr := errors.New("chmod temp queue file")
	queueCreateTemp = func(dir string, pattern string) (tempFile, error) {
		return &stubTempFile{name: filepath.Join(dir, "queue-chmod.tmp"), chmodErr: chmodErr}, nil
	}
	queueRename = func(oldPath string, newPath string) error {
		t.Fatal("did not expect rename after chmod failure")
		return nil
	}
	if err := queue.Save(); !errors.Is(err, chmodErr) {
		t.Fatalf("expected chmod failure to propagate, got %v", err)
	}
}

func TestParallelIssueQueueUpsertIssueCreatesAndUpdatesQueueRecord(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-238"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-238", Title: "seed queue entry", Track: "Go Mainline Follow-ups", Status: "Todo"},
			},
		},
	}

	action, orderAdded, err := queue.UpsertIssue(IssueRecord{
		Identifier: "BIG-PAR-239",
		Title:      "sync recent batches",
		Track:      "Go Mainline Follow-ups",
		Status:     "Todo",
	})
	if err != nil {
		t.Fatalf("upsert create: %v", err)
	}
	if action != "created" || !orderAdded {
		t.Fatalf("expected created action with order add, got action=%s orderAdded=%v", action, orderAdded)
	}
	if len(queue.payload.Issues) != 2 || queue.payload.IssueOrder[len(queue.payload.IssueOrder)-1] != "BIG-PAR-239" {
		t.Fatalf("expected queue append, got %+v", queue.payload)
	}

	action, orderAdded, err = queue.UpsertIssue(IssueRecord{
		Identifier: "BIG-PAR-239",
		Title:      "sync recent_batches metadata from local tracker",
		Track:      "Go Mainline Follow-ups",
		Status:     "In Progress",
	})
	if err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if action != "updated" || orderAdded {
		t.Fatalf("expected updated action without order add, got action=%s orderAdded=%v", action, orderAdded)
	}
	if queue.payload.Issues[1].Status != "In Progress" || queue.payload.Issues[1].Title != "sync recent_batches metadata from local tracker" {
		t.Fatalf("expected queue record update, got %+v", queue.payload.Issues[1])
	}
}

func TestParallelIssueQueueUpsertIssueValidationDefaultStatusAndOrderRepair(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-408", Title: "Add refill queue load/save helper coverage", Track: "Go Mainline Follow-ups", Status: "Todo"},
			},
		},
	}

	if _, _, err := queue.UpsertIssue(IssueRecord{Identifier: " ", Title: "missing", Track: "track"}); err == nil {
		t.Fatal("expected blank identifier to fail")
	}
	if _, _, err := queue.UpsertIssue(IssueRecord{Identifier: "BIG-PAR-409", Title: " ", Track: "track"}); err == nil {
		t.Fatal("expected blank title to fail")
	}
	if _, _, err := queue.UpsertIssue(IssueRecord{Identifier: "BIG-PAR-409", Title: "title", Track: " "}); err == nil {
		t.Fatal("expected blank track to fail")
	}

	action, orderAdded, err := queue.UpsertIssue(IssueRecord{
		Identifier: " big-par-408 ",
		Title:      "Add refill queue load/save helper coverage",
		Track:      "Go Mainline Follow-ups",
		Status:     " ",
	})
	if err != nil {
		t.Fatalf("upsert existing queue entry: %v", err)
	}
	if action != "updated" || !orderAdded {
		t.Fatalf("expected order repair to report updated with order add, got action=%s orderAdded=%v", action, orderAdded)
	}
	if !equalStringSlices(queue.payload.IssueOrder, []string{"BIG-PAR-408"}) {
		t.Fatalf("expected repaired issue order, got %+v", queue.payload.IssueOrder)
	}
	if queue.payload.Issues[0].Status != "Todo" {
		t.Fatalf("expected blank status to default to Todo, got %+v", queue.payload.Issues[0])
	}

	action, orderAdded, err = queue.UpsertIssue(IssueRecord{
		Identifier: "BIG-PAR-408",
		Title:      "Add refill queue load/save helper coverage",
		Track:      "Go Mainline Follow-ups",
		Status:     "Todo",
	})
	if err != nil {
		t.Fatalf("upsert unchanged queue entry: %v", err)
	}
	if action != "exists" || orderAdded {
		t.Fatalf("expected unchanged upsert to report exists without order add, got action=%s orderAdded=%v", action, orderAdded)
	}
}

func TestParallelIssueQueueUpsertIssueTrimsCreateFieldsAndUpdatesTrackInPlace(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-420"},
			Issues: []IssueRecord{
				{Identifier: " BIG-PAR-420 ", Title: "Existing title", Track: "Legacy Track", Status: "Todo"},
			},
		},
	}

	action, orderAdded, err := queue.UpsertIssue(IssueRecord{
		Identifier: " BIG-PAR-421 ",
		Title:      "  Trimmed title  ",
		Track:      " Go Mainline Follow-ups ",
		Status:     " ",
	})
	if err != nil {
		t.Fatalf("create trimmed queue entry: %v", err)
	}
	if action != "created" || !orderAdded {
		t.Fatalf("expected trimmed create to report created with order add, got action=%s orderAdded=%v", action, orderAdded)
	}
	if got := queue.payload.Issues[1]; got.Identifier != "BIG-PAR-421" || got.Title != "Trimmed title" || got.Track != "Go Mainline Follow-ups" || got.Status != "Todo" {
		t.Fatalf("expected created queue entry to store trimmed values and default status, got %+v", got)
	}

	action, orderAdded, err = queue.UpsertIssue(IssueRecord{
		Identifier: "big-par-420",
		Title:      "Existing title",
		Track:      "Go Mainline Follow-ups",
		Status:     "Todo",
	})
	if err != nil {
		t.Fatalf("update existing queue track: %v", err)
	}
	if action != "updated" || orderAdded {
		t.Fatalf("expected case-insensitive upsert to update in place without order add, got action=%s orderAdded=%v", action, orderAdded)
	}
	if got := queue.payload.Issues[0].Track; got != "Go Mainline Follow-ups" {
		t.Fatalf("expected existing queue entry track update, got %q", got)
	}
	if !equalStringSlices(queue.payload.IssueOrder, []string{"BIG-PAR-420", "BIG-PAR-421"}) {
		t.Fatalf("expected queue order to avoid duplicate identifiers, got %+v", queue.payload.IssueOrder)
	}
}

func TestParallelIssueQueueSetRecentBatchMovesIdentifiersBetweenBuckets(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			IssueOrder: []string{"BIG-PAR-406", "BIG-PAR-407", "BIG-PAR-408", "BIG-PAR-409"},
			RecentBatches: struct {
				Completed []string `json:"completed"`
				Active    []string `json:"active"`
				Standby   []string `json:"standby"`
			}{
				Completed: []string{"BIG-PAR-406"},
				Active:    []string{"BIG-PAR-407"},
				Standby:   []string{"BIG-PAR-408"},
			},
		},
	}

	if _, err := queue.SetRecentBatch("active", "   "); err == nil {
		t.Fatal("expected blank identifier to fail")
	}
	if _, err := queue.SetRecentBatch("unknown", "BIG-PAR-409"); err == nil {
		t.Fatal("expected unknown batch name to fail")
	}

	changed, err := queue.SetRecentBatch("completed", "BIG-PAR-408")
	if err != nil {
		t.Fatalf("move identifier to completed batch: %v", err)
	}
	if !changed {
		t.Fatal("expected moving identifier between batches to report change")
	}
	if !equalStringSlices(queue.payload.RecentBatches.Completed, []string{"BIG-PAR-406", "BIG-PAR-408"}) {
		t.Fatalf("unexpected completed batch ordering: %+v", queue.payload.RecentBatches.Completed)
	}
	if len(queue.payload.RecentBatches.Standby) != 0 {
		t.Fatalf("expected standby batch removal after move, got %+v", queue.payload.RecentBatches.Standby)
	}

	changed, err = queue.SetRecentBatch("none", "BIG-PAR-407")
	if err != nil {
		t.Fatalf("remove identifier from all recent batches: %v", err)
	}
	if !changed {
		t.Fatal("expected removing identifier from active batch to report change")
	}
	if len(queue.payload.RecentBatches.Active) != 0 {
		t.Fatalf("expected active batch removal, got %+v", queue.payload.RecentBatches.Active)
	}

	changed, err = queue.SetRecentBatch("", "BIG-PAR-409")
	if err != nil {
		t.Fatalf("clear batch for untouched identifier: %v", err)
	}
	if changed {
		t.Fatalf("expected no-op clear for untouched identifier, got changed=%v", changed)
	}
}

func TestParallelIssueQueueSyncRecentBatchesFromStates(t *testing.T) {
	queue := &ParallelIssueQueue{
		payload: QueuePayload{
			Policy: struct {
				TargetInProgress  int      `json:"target_in_progress"`
				ActivateStateID   string   `json:"activate_state_id"`
				ActivateStateName string   `json:"activate_state_name"`
				RefillStates      []string `json:"refill_states"`
				BlockedReason     string   `json:"blocked_reason,omitempty"`
			}{
				TargetInProgress:  2,
				ActivateStateName: "In Progress",
				RefillStates:      []string{"Todo", "Backlog"},
			},
			IssueOrder: []string{"BIG-PAR-235", "BIG-PAR-238", "BIG-PAR-239", "BIG-PAR-240"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-235", Status: "Todo"},
				{Identifier: "BIG-PAR-238", Status: "Todo"},
				{Identifier: "BIG-PAR-239", Status: "Todo"},
				{Identifier: "BIG-PAR-240", Status: "Todo"},
			},
		},
	}

	updates := queue.SyncRecentBatchesFromStates(map[string]string{
		"BIG-PAR-235": "Done",
		"BIG-PAR-238": "In Progress",
		"BIG-PAR-239": "In Progress",
		"BIG-PAR-240": "Todo",
	})
	if updates != 3 {
		t.Fatalf("expected three recent batch updates, got %d", updates)
	}

	snapshot := queue.RecentBatchesSnapshot()
	if !stringSlicesEqual(snapshot.Completed, []string{"BIG-PAR-235"}) {
		t.Fatalf("unexpected completed snapshot: %+v", snapshot)
	}
	if !stringSlicesEqual(snapshot.Active, []string{"BIG-PAR-238", "BIG-PAR-239"}) {
		t.Fatalf("unexpected active snapshot: %+v", snapshot)
	}
	if !stringSlicesEqual(snapshot.Standby, []string{"BIG-PAR-240"}) {
		t.Fatalf("unexpected standby snapshot: %+v", snapshot)
	}
}

func TestParallelIssueQueueSaveMarkdownRendersCurrentBatchAndOrder(t *testing.T) {
	queuePath := filepath.Join(t.TempDir(), "queue.json")
	markdownPath := filepath.Join(t.TempDir(), "queue.md")
	queue := &ParallelIssueQueue{
		queuePath: queuePath,
		payload: QueuePayload{
			Policy: struct {
				TargetInProgress  int      `json:"target_in_progress"`
				ActivateStateID   string   `json:"activate_state_id"`
				ActivateStateName string   `json:"activate_state_name"`
				RefillStates      []string `json:"refill_states"`
				BlockedReason     string   `json:"blocked_reason,omitempty"`
			}{
				TargetInProgress:  2,
				ActivateStateName: "In Progress",
				RefillStates:      []string{"Todo", "Backlog"},
			},
			RecentBatches: struct {
				Completed []string `json:"completed"`
				Active    []string `json:"active"`
				Standby   []string `json:"standby"`
			}{
				Completed: []string{"BIG-PAR-244", "BIG-PAR-245"},
				Active:    []string{"BIG-PAR-247"},
				Standby:   []string{"BIG-PAR-248"},
			},
			IssueOrder: []string{"BIG-PAR-244", "BIG-PAR-245", "BIG-PAR-247", "BIG-PAR-248"},
			Issues: []IssueRecord{
				{Identifier: "BIG-PAR-244", Title: "refresh queue docs", Status: "Done"},
				{Identifier: "BIG-PAR-245", Title: "open branch PR", Status: "Done"},
				{Identifier: "BIG-PAR-247", Title: "sync queue markdown", Status: "In Progress"},
				{Identifier: "BIG-PAR-248", Title: "follow-on refill slice", Status: "Todo"},
			},
		},
	}

	written, err := queue.SaveMarkdown(markdownPath, time.Date(2026, 3, 23, 3, 45, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("save markdown: %v", err)
	}
	if !written {
		t.Fatalf("expected markdown write")
	}

	body, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "Current repo tranche status as of March 23, 2026") {
		t.Fatalf("expected generated status date, got %s", text)
	}
	if !strings.Contains(text, "`BIG-PAR-247` — sync queue markdown") {
		t.Fatalf("expected active issue summary, got %s", text)
	}
	if !strings.Contains(text, "4. `BIG-PAR-248`") {
		t.Fatalf("expected numbered canonical order, got %s", text)
	}
}
