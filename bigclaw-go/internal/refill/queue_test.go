package refill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
