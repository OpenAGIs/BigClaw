package memory

import (
	"path/filepath"
	"reflect"
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

	suggestion, err := store.SuggestRules(current, 3)
	if err != nil {
		t.Fatalf("suggest rules: %v", err)
	}
	if !reflect.DeepEqual(suggestion.MatchedTaskIDs, []string{"BIG-501-prev"}) {
		t.Fatalf("unexpected matched task ids: %+v", suggestion.MatchedTaskIDs)
	}
	if !reflect.DeepEqual(suggestion.AcceptanceCriteria, []string{"unit-tests", "report-shared"}) {
		t.Fatalf("unexpected acceptance criteria: %+v", suggestion.AcceptanceCriteria)
	}
	if !reflect.DeepEqual(suggestion.ValidationPlan, []string{"pytest", "smoke-test"}) {
		t.Fatalf("unexpected validation plan: %+v", suggestion.ValidationPlan)
	}
}
