package memory

import (
	"path/filepath"
	"reflect"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestTaskStoreRemembersPatternsAndSuggestsRules(t *testing.T) {
	store := NewTaskStore(filepath.Join(t.TempDir(), "memory", "task-patterns.json"))
	if err := store.RememberSuccess(domain.Task{
		ID:                 "BIG-501-a",
		Title:              "High risk repo rollout",
		Labels:             []string{"prod", "ops"},
		RequiredTools:      []string{"github", "browser"},
		AcceptanceCriteria: []string{"repo rollout documented"},
		ValidationPlan:     []string{"go test ./internal/repo"},
	}, "captured"); err != nil {
		t.Fatalf("remember success: %v", err)
	}
	if err := store.RememberSuccess(domain.Task{
		ID:                 "BIG-501-b",
		Title:              "Observability audit",
		Labels:             []string{"ops"},
		RequiredTools:      []string{"browser"},
		AcceptanceCriteria: []string{"audit exported"},
		ValidationPlan:     []string{"go test ./internal/observability"},
	}, "captured"); err != nil {
		t.Fatalf("remember success: %v", err)
	}
	suggestions, err := store.SuggestRules(domain.Task{
		ID:                 "BIG-501-target",
		Title:              "Repo audit follow-up",
		Labels:             []string{"prod", "ops"},
		RequiredTools:      []string{"browser"},
		AcceptanceCriteria: []string{"baseline criterion"},
		ValidationPlan:     []string{"baseline validation"},
	}, 2)
	if err != nil {
		t.Fatalf("suggest rules: %v", err)
	}
	if !reflect.DeepEqual(suggestions.MatchedTaskIDs, []string{"BIG-501-a", "BIG-501-b"}) {
		t.Fatalf("unexpected matched tasks: %+v", suggestions.MatchedTaskIDs)
	}
	if !reflect.DeepEqual(suggestions.AcceptanceCriteria, []string{"baseline criterion", "repo rollout documented", "audit exported"}) {
		t.Fatalf("unexpected acceptance criteria: %+v", suggestions.AcceptanceCriteria)
	}
	if !reflect.DeepEqual(suggestions.ValidationPlan, []string{"baseline validation", "go test ./internal/repo", "go test ./internal/observability"}) {
		t.Fatalf("unexpected validation plan: %+v", suggestions.ValidationPlan)
	}
}
