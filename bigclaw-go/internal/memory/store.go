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

type SuggestionSet struct {
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
	MatchedTaskIDs     []string `json:"matched_task_ids,omitempty"`
}

type TaskStore struct {
	path string
}

func NewTaskStore(path string) TaskStore {
	return TaskStore{path: path}
}

func (s TaskStore) RememberSuccess(task domain.Task, summary string) error {
	patterns, err := s.loadPatterns()
	if err != nil {
		return err
	}
	filtered := make([]Pattern, 0, len(patterns))
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
	return s.writePatterns(filtered)
}

func (s TaskStore) SuggestRules(task domain.Task, limit int) (SuggestionSet, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return SuggestionSet{}, err
	}
	type rankedPattern struct {
		score   float64
		pattern Pattern
	}
	ranked := make([]rankedPattern, 0)
	for _, pattern := range patterns {
		score := similarityScore(task, pattern)
		if score <= 0 {
			continue
		}
		ranked = append(ranked, rankedPattern{score: score, pattern: pattern})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].pattern.TaskID < ranked[j].pattern.TaskID
		}
		return ranked[i].score > ranked[j].score
	})
	if limit <= 0 {
		limit = 1
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := SuggestionSet{
		AcceptanceCriteria: append([]string(nil), task.AcceptanceCriteria...),
		ValidationPlan:     append([]string(nil), task.ValidationPlan...),
	}
	for _, item := range ranked {
		out.MatchedTaskIDs = append(out.MatchedTaskIDs, item.pattern.TaskID)
		for _, criterion := range item.pattern.AcceptanceCriteria {
			if !contains(out.AcceptanceCriteria, criterion) {
				out.AcceptanceCriteria = append(out.AcceptanceCriteria, criterion)
			}
		}
		for _, step := range item.pattern.ValidationPlan {
			if !contains(out.ValidationPlan, step) {
				out.ValidationPlan = append(out.ValidationPlan, step)
			}
		}
	}
	return out, nil
}

func (s TaskStore) loadPatterns() ([]Pattern, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var patterns []Pattern
	if len(body) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(body, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s TaskStore) writePatterns(patterns []Pattern) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, body, 0o644)
}

func similarityScore(task domain.Task, pattern Pattern) float64 {
	return float64((overlap(task.Labels, pattern.Labels) * 2) + overlap(task.RequiredTools, pattern.RequiredTools))
}

func overlap(left, right []string) int {
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

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
