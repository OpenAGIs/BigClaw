package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"bigclaw-go/internal/domain"
)

type Pattern struct {
	TaskID             string   `json:"task_id"`
	Title              string   `json:"title"`
	Labels             []string `json:"labels,omitempty"`
	RequiredTools      []string `json:"required_tools,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
	Summary            string   `json:"summary,omitempty"`
}

type Suggestion struct {
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
	MatchedTaskIDs     []string `json:"matched_task_ids"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) RememberSuccess(task domain.Task, summary string) error {
	patterns, err := s.load()
	if err != nil {
		return err
	}
	filtered := patterns[:0]
	for _, pattern := range patterns {
		if pattern.TaskID != task.ID {
			filtered = append(filtered, pattern)
		}
	}
	filtered = append(filtered, Pattern{
		TaskID:             task.ID,
		Title:              task.Title,
		Labels:             append([]string(nil), task.Labels...),
		RequiredTools:      append([]string(nil), task.RequiredTools...),
		AcceptanceCriteria: append([]string(nil), task.AcceptanceCriteria...),
		ValidationPlan:     append([]string(nil), task.ValidationPlan...),
		Summary:            summary,
	})
	return s.write(filtered)
}

func (s *Store) SuggestRules(task domain.Task, limit int) (Suggestion, error) {
	patterns, err := s.load()
	if err != nil {
		return Suggestion{}, err
	}
	if limit <= 0 {
		limit = 1
	}
	type rankedPattern struct {
		pattern Pattern
		score   int
	}
	ranked := make([]rankedPattern, 0, len(patterns))
	for _, pattern := range patterns {
		score := overlapScore(task, pattern)
		if score <= 0 {
			continue
		}
		ranked = append(ranked, rankedPattern{pattern: pattern, score: score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].pattern.TaskID < ranked[j].pattern.TaskID
		}
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	suggestion := Suggestion{
		AcceptanceCriteria: append([]string(nil), task.AcceptanceCriteria...),
		ValidationPlan:     append([]string(nil), task.ValidationPlan...),
		MatchedTaskIDs:     make([]string, 0, len(ranked)),
	}
	for _, item := range ranked {
		suggestion.MatchedTaskIDs = append(suggestion.MatchedTaskIDs, item.pattern.TaskID)
		for _, acceptance := range item.pattern.AcceptanceCriteria {
			appendUnique(&suggestion.AcceptanceCriteria, acceptance)
		}
		for _, validation := range item.pattern.ValidationPlan {
			appendUnique(&suggestion.ValidationPlan, validation)
		}
	}
	return suggestion, nil
}

func (s *Store) load() ([]Pattern, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var patterns []Pattern
	if err := json.Unmarshal(data, &patterns); err != nil {
		return nil, err
	}
	for i := range patterns {
		patterns[i].Labels = append([]string(nil), patterns[i].Labels...)
		patterns[i].RequiredTools = append([]string(nil), patterns[i].RequiredTools...)
		patterns[i].AcceptanceCriteria = append([]string(nil), patterns[i].AcceptanceCriteria...)
		patterns[i].ValidationPlan = append([]string(nil), patterns[i].ValidationPlan...)
	}
	return patterns, nil
}

func (s *Store) write(patterns []Pattern) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, payload, 0o644)
}

func overlapScore(task domain.Task, pattern Pattern) int {
	return 2*intersectionSize(task.Labels, pattern.Labels) + intersectionSize(task.RequiredTools, pattern.RequiredTools)
}

func intersectionSize(left, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(left))
	for _, item := range left {
		set[item] = struct{}{}
	}
	count := 0
	for _, item := range right {
		if _, ok := set[item]; ok {
			count++
		}
	}
	return count
}

func appendUnique(items *[]string, value string) {
	for _, existing := range *items {
		if existing == value {
			return
		}
	}
	*items = append(*items, value)
}
