package policy

import (
	"path/filepath"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestTaskMemoryStoreReusesHistoryAndInjectsRules(t *testing.T) {
	store := NewTaskMemoryStore(filepath.Join(t.TempDir(), "memory", "task-patterns.json"))

	previous := domain.Task{
		ID:                 "BIG-501-prev",
		Source:             "github",
		Title:              "Previous queue rollout",
		Labels:             []string{"queue", "platform"},
		RequiredTools:      []string{"github", "browser"},
		AcceptanceCriteria: []string{"report-shared"},
		ValidationPlan:     []string{"pytest", "smoke-test"},
	}
	if err := store.RememberSuccess(previous, "queue migration done"); err != nil {
		t.Fatalf("remember success: %v", err)
	}

	current := domain.Task{
		ID:                 "BIG-501-new",
		Source:             "github",
		Title:              "New queue hardening",
		Labels:             []string{"queue"},
		RequiredTools:      []string{"github"},
		AcceptanceCriteria: []string{"unit-tests"},
		ValidationPlan:     []string{"pytest"},
	}
	suggestion, err := store.SuggestRules(current, 3)
	if err != nil {
		t.Fatalf("suggest rules: %v", err)
	}

	if len(suggestion.MatchedTaskIDs) != 1 || suggestion.MatchedTaskIDs[0] != "BIG-501-prev" {
		t.Fatalf("expected previous task to match, got %+v", suggestion)
	}
	if !containsString(suggestion.AcceptanceCriteria, "report-shared") {
		t.Fatalf("expected acceptance criteria merge, got %+v", suggestion)
	}
	if !containsString(suggestion.ValidationPlan, "smoke-test") {
		t.Fatalf("expected validation plan merge, got %+v", suggestion)
	}
	if !containsString(suggestion.AcceptanceCriteria, "unit-tests") {
		t.Fatalf("expected original acceptance criteria retained, got %+v", suggestion)
	}
}
