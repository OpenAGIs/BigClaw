package refill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadLocalIssueMapsMissingAndInvalidPayloads(t *testing.T) {
	tempDir := t.TempDir()

	missingPath := filepath.Join(tempDir, "missing-local-issues.json")
	issues, err := readLocalIssueMaps(missingPath)
	if err != nil {
		t.Fatalf("read missing local issue store: %v", err)
	}
	if issues != nil {
		t.Fatalf("expected missing local issue store to return nil issues, got %+v", issues)
	}

	invalidPath := filepath.Join(tempDir, "invalid-local-issues.json")
	if err := os.WriteFile(invalidPath, []byte(`{"issues":`), 0o644); err != nil {
		t.Fatalf("write invalid local issue store: %v", err)
	}
	if _, err := readLocalIssueMaps(invalidPath); err == nil || !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatalf("expected invalid JSON read to fail, got %v", err)
	}
}

func TestLoadLocalIssueStoreNormalizesAbsolutePathAndMissingIssues(t *testing.T) {
	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	relativePath, err := filepath.Rel(cwd, filepath.Join(tempDir, "missing-local-issues.json"))
	if err != nil {
		t.Fatalf("derive relative path: %v", err)
	}

	store, err := LoadLocalIssueStore(relativePath)
	if err != nil {
		t.Fatalf("load local issue store from relative path: %v", err)
	}
	if store.path != filepath.Join(tempDir, "missing-local-issues.json") {
		t.Fatalf("expected absolute store path, got %q", store.path)
	}
	if issues := store.Issues(); len(issues) != 0 {
		t.Fatalf("expected missing store to load with no issues, got %+v", issues)
	}
}

func TestLoadLocalIssueStoreAndReloadPropagateDecodeFailures(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{"issues":[{"identifier":"BIG-PAR-407"}]}`), 0o644); err != nil {
		t.Fatalf("write initial local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load initial local issue store: %v", err)
	}

	if err := os.WriteFile(storePath, []byte(`{"issues":`), 0o644); err != nil {
		t.Fatalf("corrupt local issue store: %v", err)
	}
	if err := store.Reload(); err == nil || !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatalf("expected reload to surface decode error, got %v", err)
	}

	if _, err := LoadLocalIssueStore(storePath); err == nil || !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatalf("expected load to surface decode error, got %v", err)
	}
}

func TestNormalizeLocalIssueMapsSkipsNonMapEntries(t *testing.T) {
	items := []any{
		map[string]any{"identifier": "BIG-PAR-399"},
		"skip-me",
		42,
		map[string]any{"identifier": "BIG-PAR-400"},
	}

	issues := normalizeLocalIssueMaps(items)
	if len(issues) != 2 {
		t.Fatalf("expected only map entries to survive, got %+v", issues)
	}
	if issues[0]["identifier"] != "BIG-PAR-399" || issues[1]["identifier"] != "BIG-PAR-400" {
		t.Fatalf("unexpected normalized issues: %+v", issues)
	}
}

func TestIssueCommentListNormalizesCommentCollections(t *testing.T) {
	comments := issueCommentList([]any{
		map[string]any{"body": "kept"},
		"skip-me",
		map[string]any{"body": "also kept"},
	})
	if len(comments) != 2 {
		t.Fatalf("expected only map comments, got %+v", comments)
	}
	if comments[0]["body"] != "kept" || comments[1]["body"] != "also kept" {
		t.Fatalf("unexpected normalized comments: %+v", comments)
	}

	empty := issueCommentList("not-a-list")
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected non-list comments to normalize to empty slice, got %+v", empty)
	}
}

func TestLocalIssueScalarHelpersNormalizeTypes(t *testing.T) {
	issue := map[string]any{
		"priority_float":       float64(3),
		"priority_int":         4,
		"priority_bad":         "high",
		"assigned_true":        true,
		"assigned_bad":         "yes",
		"labels_any":           []any{" ops ", "", 12},
		"labels_strings":       []string{" one ", "", "two"},
		"labels_bad":           99,
		"labels_nil_explicit":  nil,
	}

	if got := mapInt(issue, "priority_float"); got != 3 {
		t.Fatalf("expected float priority to coerce to 3, got %d", got)
	}
	if got := mapInt(issue, "priority_int"); got != 4 {
		t.Fatalf("expected int priority to round-trip, got %d", got)
	}
	if got := mapInt(issue, "priority_bad"); got != 0 {
		t.Fatalf("expected invalid priority to return zero, got %d", got)
	}
	if got := mapInt(issue, "missing_priority"); got != 0 {
		t.Fatalf("expected missing priority to return zero, got %d", got)
	}

	if !mapBool(issue, "assigned_true") {
		t.Fatal("expected bool helper to preserve true")
	}
	if mapBool(issue, "assigned_bad") {
		t.Fatal("expected non-bool value to return false")
	}
	if mapBool(issue, "missing_assigned") {
		t.Fatal("expected missing bool value to return false")
	}

	labelsAny := mapStringSlice(issue, "labels_any")
	if len(labelsAny) != 2 || labelsAny[0] != "ops" || labelsAny[1] != "12" {
		t.Fatalf("expected []any labels to normalize, got %+v", labelsAny)
	}
	labelsStrings := mapStringSlice(issue, "labels_strings")
	if len(labelsStrings) != 2 || labelsStrings[0] != "one" || labelsStrings[1] != "two" {
		t.Fatalf("expected []string labels to normalize, got %+v", labelsStrings)
	}
	if got := mapStringSlice(issue, "labels_bad"); got != nil {
		t.Fatalf("expected invalid label type to return nil, got %+v", got)
	}
	if got := mapStringSlice(issue, "labels_nil_explicit"); got != nil {
		t.Fatalf("expected explicit nil label value to return nil, got %+v", got)
	}
	if got := mapStringSlice(issue, "missing_labels"); got != nil {
		t.Fatalf("expected missing labels to return nil, got %+v", got)
	}
}

func TestLocalIssueStoreAccessorsAndCreateIssue(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-403",
      "identifier": "BIG-PAR-403",
      "title": "Accessor coverage",
      "description": "  detail preserved  ",
      "state": "In Progress",
      "priority": 2,
      "labels": [" refill ", "coverage"],
      "assigned_to_worker": true,
      "created_at": "2026-03-26T10:00:00Z",
      "updated_at": "2026-03-26T10:10:00Z"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	issues := store.Issues()
	if len(issues) != 1 {
		t.Fatalf("expected one issue from Issues accessor, got %+v", issues)
	}
	if issues[0].Identifier != "BIG-PAR-403" || issues[0].Priority != 2 || !issues[0].AssignedToWorker {
		t.Fatalf("unexpected Issues accessor payload: %+v", issues[0])
	}
	if len(issues[0].Labels) != 2 || issues[0].Labels[0] != "refill" || issues[0].Labels[1] != "coverage" {
		t.Fatalf("expected normalized labels from Issues accessor, got %+v", issues[0].Labels)
	}

	found, ok := store.FindIssue("big-par-403")
	if !ok {
		t.Fatal("expected FindIssue to resolve identifier case-insensitively")
	}
	if found.ID != "big-par-403" || found.Title != "Accessor coverage" {
		t.Fatalf("unexpected FindIssue result: %+v", found)
	}
	if _, ok := store.FindIssue("BIG-PAR-999"); ok {
		t.Fatal("expected FindIssue miss for unknown reference")
	}

	createdAt := time.Date(2026, 3, 26, 11, 30, 45, 500_000_000, time.UTC)
	created, err := store.CreateIssue(LocalIssueCreateParams{
		Identifier:       "BIG-PAR-404",
		Title:            "Create issue coverage",
		Description:      "added by test",
		State:            "",
		Priority:         3,
		Labels:           []string{"refill", "helpers"},
		AssignedToWorker: true,
		CreatedAt:        createdAt,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if created.ID != "big-par-404" {
		t.Fatalf("expected generated id from identifier, got %+v", created)
	}
	if created.State != "Todo" {
		t.Fatalf("expected default Todo state, got %+v", created)
	}
	if created.CreatedAt != "2026-03-26T11:30:45Z" || created.UpdatedAt != "2026-03-26T11:30:45Z" {
		t.Fatalf("expected truncated timestamps, got %+v", created)
	}

	reloaded, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("reload local issue store: %v", err)
	}
	createdFound, ok := reloaded.FindIssue("BIG-PAR-404")
	if !ok {
		t.Fatal("expected created issue to persist to store")
	}
	if createdFound.State != "Todo" || len(createdFound.Labels) != 2 || !createdFound.AssignedToWorker {
		t.Fatalf("unexpected persisted created issue: %+v", createdFound)
	}

	if _, err := reloaded.CreateIssue(LocalIssueCreateParams{
		Identifier: "BIG-PAR-404",
		Title:      "Duplicate issue",
	}); err == nil {
		t.Fatal("expected duplicate identifier create to fail")
	}

	autoCreated, err := reloaded.CreateIssue(LocalIssueCreateParams{
		ID:               "custom-big-par-405",
		Identifier:       "BIG-PAR-405",
		Title:            "Auto timestamp issue",
		State:            "In Progress",
		AssignedToWorker: false,
	})
	if err != nil {
		t.Fatalf("create issue with auto timestamp: %v", err)
	}
	if autoCreated.ID != "custom-big-par-405" || autoCreated.State != "In Progress" {
		t.Fatalf("expected explicit id and state to persist, got %+v", autoCreated)
	}
	if autoCreated.CreatedAt == "" || autoCreated.UpdatedAt == "" {
		t.Fatalf("expected zero-time create to auto-populate timestamps, got %+v", autoCreated)
	}
}

func TestLocalIssueStoreCreateIssueRequiresIdentifierAndTitle(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{"issues":[]}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	if _, err := store.CreateIssue(LocalIssueCreateParams{Identifier: "   ", Title: "missing identifier"}); err == nil {
		t.Fatal("expected missing identifier to fail")
	}
	if _, err := store.CreateIssue(LocalIssueCreateParams{Identifier: "BIG-PAR-406", Title: "   "}); err == nil {
		t.Fatal("expected missing title to fail")
	}
}

func TestDecodeLocalIssueMapsCoversArrayObjectAndErrorPaths(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantLen int
		wantErr string
	}{
		{
			name:    "empty body",
			body:    "   ",
			wantLen: 0,
		},
		{
			name:    "top level array",
			body:    `[{"identifier":"BIG-PAR-405"},"skip",{"identifier":"BIG-PAR-406"}]`,
			wantLen: 2,
		},
		{
			name:    "issues object",
			body:    `{"issues":[{"identifier":"BIG-PAR-405"},"skip"]}`,
			wantLen: 1,
		},
		{
			name:    "missing issues key",
			body:    `{"unexpected":[]}`,
			wantErr: "invalid local issue store payload",
		},
		{
			name:    "invalid issues list",
			body:    `{"issues":"bad"}`,
			wantErr: "invalid local issue list",
		},
		{
			name:    "invalid root type",
			body:    `42`,
			wantErr: "invalid local issue store payload",
		},
		{
			name:    "invalid json",
			body:    `{"issues":`,
			wantErr: "unexpected end of JSON input",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues, err := decodeLocalIssueMaps([]byte(test.body))
			if test.wantErr != "" {
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("expected error %q, got issues=%+v err=%v", test.wantErr, issues, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("decode local issue maps: %v", err)
			}
			if len(issues) != test.wantLen {
				t.Fatalf("expected %d issues, got %+v", test.wantLen, issues)
			}
		})
	}
}

func TestLocalIssueStoreUpdateIssueStatePreservesExtraFields(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "Todo",
      "labels": ["go", "tooling"],
      "assigned_to_worker": true,
      "branch_name": "feat/big-gom-307",
      "comments": [
        {"body": "seed comment", "created_at": "2026-03-18T09:00:00Z"}
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	updatedState, err := store.UpdateIssueState("BIG-GOM-307", "In Progress", time.Date(2026, 3, 18, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("update issue state: %v", err)
	}
	if updatedState != "In Progress" {
		t.Fatalf("expected updated state, got %q", updatedState)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	if !strings.Contains(string(body), `"branch_name": "feat/big-gom-307"`) {
		t.Fatalf("expected extra fields to be preserved, got %s", string(body))
	}
	if !strings.Contains(string(body), `"comments": [`) {
		t.Fatalf("expected comments to be preserved, got %s", string(body))
	}
	if !strings.Contains(string(body), `"updated_at": "2026-03-18T15:00:00Z"`) {
		t.Fatalf("expected updated_at to refresh, got %s", string(body))
	}
}

func TestLocalIssueStoreUpdateIssueStateReturnsNotFound(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{"issues":[]}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	updatedState, err := store.UpdateIssueState("BIG-PAR-999", "Done", time.Date(2026, 3, 25, 18, 56, 0, 0, time.UTC))
	if err == nil || err != ErrLocalIssueNotFound {
		t.Fatalf("expected ErrLocalIssueNotFound, got updatedState=%q err=%v", updatedState, err)
	}
}

func TestLocalIssueStoreIssueStatesFiltersRequestedStates(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-gom-301", "identifier": "BIG-GOM-301", "state": "In Progress"},
    {"id": "big-gom-303", "identifier": "BIG-GOM-303", "state": "Todo"},
    {"id": "big-gom-305", "identifier": "BIG-GOM-305", "state": "Backlog"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	issues := store.IssueStates([]string{"In Progress", "Todo"})
	if len(issues) != 2 {
		t.Fatalf("expected 2 active issues, got %d", len(issues))
	}
	if issues[0].Identifier != "BIG-GOM-301" || issues[1].Identifier != "BIG-GOM-303" {
		t.Fatalf("unexpected filtered issues: %+v", issues)
	}
}

func TestLocalIssueStoreAddCommentAppendsAndUpdatesTimestamp(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "In Progress",
      "comments": [
        {"author": "codex", "created_at": "2026-03-18T09:00:00Z", "body": "seed comment"}
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	if err := store.AddComment("BIG-GOM-307", LocalIssueComment{
		Author:    "codex",
		Body:      "validation passed",
		CreatedAt: time.Date(2026, 3, 20, 10, 45, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("add comment: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if strings.Count(text, `"body":`) != 2 {
		t.Fatalf("expected appended comment list, got %s", text)
	}
	if !strings.Contains(text, `"body": "validation passed"`) {
		t.Fatalf("expected appended comment body, got %s", text)
	}
	if !strings.Contains(text, `"updated_at": "2026-03-20T10:45:00Z"`) {
		t.Fatalf("expected updated_at refresh, got %s", text)
	}
}

func TestLocalIssueStoreUpdateIssueStateReloadsLatestStateBeforeSaving(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-411",
      "identifier": "BIG-PAR-411",
      "title": "state update coverage",
      "state": "Todo"
    },
    {
      "id": "big-par-412",
      "identifier": "BIG-PAR-412",
      "title": "second writer state update coverage",
      "state": "Todo"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	storeA, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store A: %v", err)
	}
	storeB, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store B: %v", err)
	}

	if _, err := storeA.UpdateIssueState("BIG-PAR-411", "In Progress", time.Date(2026, 3, 25, 18, 56, 1, 0, time.UTC)); err != nil {
		t.Fatalf("update issue state with store A: %v", err)
	}
	if _, err := storeB.UpdateIssueState("BIG-PAR-412", "Done", time.Date(2026, 3, 25, 18, 56, 2, 0, time.UTC)); err != nil {
		t.Fatalf("update issue state with store B: %v", err)
	}

	reloaded, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("reload local issue store: %v", err)
	}
	first, ok := reloaded.FindIssue("BIG-PAR-411")
	if !ok || first.State != "In Progress" {
		t.Fatalf("expected first state update to persist, got %+v found=%v", first, ok)
	}
	second, ok := reloaded.FindIssue("BIG-PAR-412")
	if !ok || second.State != "Done" {
		t.Fatalf("expected second state update to persist, got %+v found=%v", second, ok)
	}
}

func TestLocalIssueStoreSaveDoesNotEscapeArrowTokens(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-gom-307",
      "identifier": "BIG-GOM-307",
      "title": "Toolchain migration",
      "state": "Todo",
      "comments": [
        {"author": "codex", "created_at": "2026-03-18T09:00:00Z", "body": "seed -> ok"}
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	if _, err := store.UpdateIssueState("BIG-GOM-307", "In Progress", time.Date(2026, 3, 18, 15, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("update issue state: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "seed -> ok") {
		t.Fatalf("expected arrow token to persist, got %s", text)
	}
	if strings.Contains(text, `\\u003e`) {
		t.Fatalf("expected no HTML escaping, got %s", text)
	}
}

func TestLocalIssueStoreSaveWritesEmptyIssuesPayload(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	store := &LocalIssueStore{path: storePath}

	if err := store.Save(); err != nil {
		t.Fatalf("save empty local issue store: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read saved local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"issues": []`) {
		t.Fatalf("expected empty issues payload after save, got %s", text)
	}
}

func TestLocalIssueStoreSaveUnlockedCreatesNestedDirectory(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "nested", "tracker", "local-issues.json")
	store := &LocalIssueStore{path: storePath}

	if err := store.saveUnlocked(); err != nil {
		t.Fatalf("save unlocked local issue store: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read saved local issue store: %v", err)
	}
	if !strings.Contains(string(body), `"issues": []`) {
		t.Fatalf("expected empty issues payload after saveUnlocked, got %s", string(body))
	}
}

func TestLocalIssueStoreSavePersistsMutatedIssueMapWithoutEscapingHTML(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	store := &LocalIssueStore{
		path: storePath,
		issueMap: []map[string]any{
			{
				"id":         "big-par-409",
				"identifier": "BIG-PAR-409",
				"title":      "save path coverage -> literal",
				"state":      "In Progress",
			},
		},
	}

	if err := store.Save(); err != nil {
		t.Fatalf("save mutated local issue store: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read mutated local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "save path coverage -> literal") {
		t.Fatalf("expected title to persist without alteration, got %s", text)
	}
	if strings.Contains(text, `\u003e`) {
		t.Fatalf("expected html escaping to stay disabled, got %s", text)
	}
}

func TestLocalIssueStoreReloadRefreshesInMemorySnapshot(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-par-243", "identifier": "BIG-PAR-243", "state": "Todo"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}
	if issues := store.IssueStates([]string{"Todo"}); len(issues) != 1 {
		t.Fatalf("expected initial todo issue, got %+v", issues)
	}

	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {"id": "big-par-243", "identifier": "BIG-PAR-243", "state": "Done"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("rewrite local issue store: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("reload local issue store: %v", err)
	}

	if issues := store.IssueStates([]string{"Todo"}); len(issues) != 0 {
		t.Fatalf("expected no todo issues after reload, got %+v", issues)
	}
	if issues := store.IssueStates([]string{"Done"}); len(issues) != 1 || issues[0].Identifier != "BIG-PAR-243" {
		t.Fatalf("expected reloaded done issue, got %+v", issues)
	}
}

func TestLocalIssueStoreAddCommentReloadsLatestStateBeforeSaving(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-241",
      "identifier": "BIG-PAR-241",
      "title": "Serialize local tracker writes with an explicit lock",
      "state": "In Progress",
      "comments": []
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	storeA, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store A: %v", err)
	}
	storeB, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store B: %v", err)
	}

	if err := storeA.AddComment("BIG-PAR-241", LocalIssueComment{
		Author:    "codex",
		Body:      "first writer",
		CreatedAt: time.Date(2026, 3, 23, 2, 45, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("add comment with store A: %v", err)
	}
	if err := storeB.AddComment("BIG-PAR-241", LocalIssueComment{
		Author:    "codex",
		Body:      "second writer",
		CreatedAt: time.Date(2026, 3, 23, 2, 45, 1, 0, time.UTC),
	}); err != nil {
		t.Fatalf("add comment with store B: %v", err)
	}

	body, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read local issue store: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"body": "first writer"`) || !strings.Contains(text, `"body": "second writer"`) {
		t.Fatalf("expected both comments to persist after stale reload protection, got %s", text)
	}
}

func TestLocalIssueStoreCreateIssueReloadsLatestStateBeforeSaving(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{"issues":[]}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	storeA, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store A: %v", err)
	}
	storeB, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store B: %v", err)
	}

	if _, err := storeA.CreateIssue(LocalIssueCreateParams{
		Identifier: "BIG-PAR-409",
		Title:      "first writer",
		CreatedAt:  time.Date(2026, 3, 25, 18, 45, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create issue with store A: %v", err)
	}
	if _, err := storeB.CreateIssue(LocalIssueCreateParams{
		Identifier: "BIG-PAR-410",
		Title:      "second writer",
		CreatedAt:  time.Date(2026, 3, 25, 18, 45, 1, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create issue with store B: %v", err)
	}

	reloaded, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("reload local issue store: %v", err)
	}
	if _, ok := reloaded.FindIssue("BIG-PAR-409"); !ok {
		t.Fatal("expected first created issue to persist after stale write protection")
	}
	if _, ok := reloaded.FindIssue("BIG-PAR-410"); !ok {
		t.Fatal("expected second created issue to persist after stale write protection")
	}
}

func TestLocalIssueStoreAddCommentRetriesTransientLockFile(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "local-issues.json")
	if err := os.WriteFile(storePath, []byte(`{
  "issues": [
    {
      "id": "big-par-241",
      "identifier": "BIG-PAR-241",
      "title": "Serialize local tracker writes with an explicit lock",
      "state": "In Progress"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write local issue store: %v", err)
	}

	store, err := LoadLocalIssueStore(storePath)
	if err != nil {
		t.Fatalf("load local issue store: %v", err)
	}

	lockPath := storePath + ".lock"
	if err := os.WriteFile(lockPath, []byte("held"), 0o644); err != nil {
		t.Fatalf("write transient lock file: %v", err)
	}
	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = os.Remove(lockPath)
	}()

	if err := store.AddComment("BIG-PAR-241", LocalIssueComment{
		Author:    "codex",
		Body:      "lock released",
		CreatedAt: time.Date(2026, 3, 23, 2, 46, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("expected transient lock retry to succeed, got %v", err)
	}
}
