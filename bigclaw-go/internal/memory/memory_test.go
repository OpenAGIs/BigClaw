package memory

import (
	"path/filepath"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestStoreReusesHistoryAndInjectsRules(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "memory", "task-patterns.json"))

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

	suggestion, err := store.SuggestRules(current)
	if err != nil {
		t.Fatalf("suggest rules: %v", err)
	}

	if !contains(suggestion.MatchedTaskIDs, "BIG-501-prev") {
		t.Fatalf("expected matched task ids to include previous task, got %v", suggestion.MatchedTaskIDs)
	}
	if !contains(suggestion.AcceptanceCriteria, "report-shared") {
		t.Fatalf("expected inherited acceptance criteria, got %v", suggestion.AcceptanceCriteria)
	}
	if !contains(suggestion.ValidationPlan, "smoke-test") {
		t.Fatalf("expected inherited validation plan, got %v", suggestion.ValidationPlan)
	}
	if !contains(suggestion.AcceptanceCriteria, "unit-tests") {
		t.Fatalf("expected current acceptance criteria to be retained, got %v", suggestion.AcceptanceCriteria)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
