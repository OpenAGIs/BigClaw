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

type TaskStore struct {
	storagePath string
}

func NewTaskStore(storagePath string) *TaskStore {
	return &TaskStore{storagePath: storagePath}
}

func (s *TaskStore) RememberSuccess(task domain.Task, summary string) error {
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
		Labels:             copyStrings(task.Labels),
		RequiredTools:      copyStrings(task.RequiredTools),
		AcceptanceCriteria: copyStrings(task.AcceptanceCriteria),
		ValidationPlan:     copyStrings(task.ValidationPlan),
		Summary:            summary,
	})
	return s.writePatterns(filtered)
}

func (s *TaskStore) SuggestRules(task domain.Task, limit int) (Suggestion, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return Suggestion{}, err
	}
	if limit <= 0 {
		limit = 1
	}
	type rankedPattern struct {
		score   float64
		pattern Pattern
	}
	ranked := make([]rankedPattern, 0, len(patterns))
	for _, pattern := range patterns {
		score := scorePattern(task, pattern)
		if score > 0 {
			ranked = append(ranked, rankedPattern{score: score, pattern: pattern})
		}
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

	acceptance := copyStrings(task.AcceptanceCriteria)
	validation := copyStrings(task.ValidationPlan)
	matched := make([]string, 0, len(ranked))
	for _, item := range ranked {
		matched = append(matched, item.pattern.TaskID)
		for _, candidate := range item.pattern.AcceptanceCriteria {
			if !contains(acceptance, candidate) {
				acceptance = append(acceptance, candidate)
			}
		}
		for _, candidate := range item.pattern.ValidationPlan {
			if !contains(validation, candidate) {
				validation = append(validation, candidate)
			}
		}
	}

	return Suggestion{
		AcceptanceCriteria: acceptance,
		ValidationPlan:     validation,
		MatchedTaskIDs:     matched,
	}, nil
}

func (s *TaskStore) loadPatterns() ([]Pattern, error) {
	if s == nil || s.storagePath == "" {
		return nil, nil
	}
	body, err := os.ReadFile(s.storagePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var patterns []Pattern
	if err := json.Unmarshal(body, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s *TaskStore) writePatterns(patterns []Pattern) error {
	if s == nil || s.storagePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storagePath, body, 0o644)
}

func scorePattern(task domain.Task, pattern Pattern) float64 {
	labelOverlap := overlapCount(task.Labels, pattern.Labels)
	toolOverlap := overlapCount(task.RequiredTools, pattern.RequiredTools)
	return float64(labelOverlap*2 + toolOverlap)
}

func overlapCount(left, right []string) int {
	seen := make(map[string]struct{}, len(left))
	for _, value := range left {
		seen[value] = struct{}{}
	}
	count := 0
	for _, value := range right {
		if _, ok := seen[value]; ok {
			count++
		}
	}
	return count
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
