package memory

import (
	"path/filepath"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestTaskStoreReusesHistoryAndInjectsRules(t *testing.T) {
	store := NewTaskStore(filepath.Join(t.TempDir(), "memory", "task-patterns.json"))

	previous := domain.Task{
		ID:                 "BIG-501-prev",
		Source:             "github",
		Title:              "Previous queue rollout",
		Description:        "",
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
		Description:        "",
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
		t.Fatalf("expected previous task match, got %+v", suggestion.MatchedTaskIDs)
	}
	assertContains(t, suggestion.AcceptanceCriteria, "report-shared")
	assertContains(t, suggestion.AcceptanceCriteria, "unit-tests")
	assertContains(t, suggestion.ValidationPlan, "smoke-test")
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in %+v", want, values)
}
