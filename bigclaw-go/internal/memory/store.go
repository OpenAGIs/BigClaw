package memory

import (
	"encoding/json"
	"os"
	"path/filepath"

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
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
	MatchedTaskIDs     []string `json:"matched_task_ids,omitempty"`
}

type TaskMemoryStore struct {
	StoragePath string
}

func (s TaskMemoryStore) loadPatterns() ([]Pattern, error) {
	body, err := os.ReadFile(s.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var patterns []Pattern
	if err := json.Unmarshal(body, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s TaskMemoryStore) writePatterns(patterns []Pattern) error {
	if err := os.MkdirAll(filepath.Dir(s.StoragePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.StoragePath, body, 0o644)
}

func (s TaskMemoryStore) RememberSuccess(task domain.Task, summary string) error {
	patterns, err := s.loadPatterns()
	if err != nil {
		return err
	}
	filtered := make([]Pattern, 0, len(patterns)+1)
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

func (s TaskMemoryStore) SuggestRules(task domain.Task, limit int) (Suggestion, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return Suggestion{}, err
	}
	type scored struct {
		score   float64
		pattern Pattern
	}
	ranked := []scored{}
	for _, pattern := range patterns {
		score := score(task, pattern)
		if score > 0 {
			ranked = append(ranked, scored{score: score, pattern: pattern})
		}
	}
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	if limit < 1 {
		limit = 1
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	acceptance := append([]string(nil), task.AcceptanceCriteria...)
	validation := append([]string(nil), task.ValidationPlan...)
	matched := make([]string, 0, len(ranked))
	for _, item := range ranked {
		matched = append(matched, item.pattern.TaskID)
		for _, criterion := range item.pattern.AcceptanceCriteria {
			if !contains(acceptance, criterion) {
				acceptance = append(acceptance, criterion)
			}
		}
		for _, step := range item.pattern.ValidationPlan {
			if !contains(validation, step) {
				validation = append(validation, step)
			}
		}
	}
	return Suggestion{
		AcceptanceCriteria: acceptance,
		ValidationPlan:     validation,
		MatchedTaskIDs:     matched,
	}, nil
}

func score(task domain.Task, pattern Pattern) float64 {
	labelOverlap := 0
	for _, label := range task.Labels {
		if contains(pattern.Labels, label) {
			labelOverlap++
		}
	}
	toolOverlap := 0
	for _, tool := range task.RequiredTools {
		if contains(pattern.RequiredTools, tool) {
			toolOverlap++
		}
	}
	return float64(labelOverlap*2 + toolOverlap)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
