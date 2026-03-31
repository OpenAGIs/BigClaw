package product

import (
	"encoding/json"
	"os"
	"path/filepath"

	"bigclaw-go/internal/domain"
)

type MemoryPattern struct {
	TaskID             string   `json:"task_id"`
	Title              string   `json:"title"`
	Labels             []string `json:"labels,omitempty"`
	RequiredTools      []string `json:"required_tools,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
	Summary            string   `json:"summary,omitempty"`
}

type MemorySuggestion struct {
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
	MatchedTaskIDs     []string `json:"matched_task_ids,omitempty"`
}

type TaskMemoryStore struct {
	StoragePath string
}

func (s TaskMemoryStore) loadPatterns() ([]MemoryPattern, error) {
	if _, err := os.Stat(s.StoragePath); os.IsNotExist(err) {
		return nil, nil
	}
	body, err := os.ReadFile(s.StoragePath)
	if err != nil {
		return nil, err
	}
	var patterns []MemoryPattern
	if err := json.Unmarshal(body, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s TaskMemoryStore) writePatterns(patterns []MemoryPattern) error {
	if err := os.MkdirAll(filepath.Dir(s.StoragePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(s.StoragePath, body, 0o644)
}

func (s TaskMemoryStore) RememberSuccess(task domain.Task, summary string) error {
	patterns, err := s.loadPatterns()
	if err != nil {
		return err
	}
	filtered := make([]MemoryPattern, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern.TaskID != task.ID {
			filtered = append(filtered, pattern)
		}
	}
	filtered = append(filtered, MemoryPattern{
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

func (s TaskMemoryStore) SuggestRules(task domain.Task, limit int) (MemorySuggestion, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return MemorySuggestion{}, err
	}
	if limit <= 0 {
		limit = 3
	}

	type scoredPattern struct {
		score   float64
		pattern MemoryPattern
	}
	ranked := make([]scoredPattern, 0, len(patterns))
	for _, pattern := range patterns {
		score := memoryScore(task, pattern)
		if score > 0 {
			ranked = append(ranked, scoredPattern{score: score, pattern: pattern})
		}
	}
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
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
			if !containsString(acceptance, criterion) {
				acceptance = append(acceptance, criterion)
			}
		}
		for _, step := range item.pattern.ValidationPlan {
			if !containsString(validation, step) {
				validation = append(validation, step)
			}
		}
	}

	return MemorySuggestion{
		AcceptanceCriteria: acceptance,
		ValidationPlan:     validation,
		MatchedTaskIDs:     matched,
	}, nil
}

func memoryScore(task domain.Task, pattern MemoryPattern) float64 {
	return float64(overlapCount(task.Labels, pattern.Labels)*2 + overlapCount(task.RequiredTools, pattern.RequiredTools))
}

func overlapCount(left []string, right []string) int {
	count := 0
	for _, item := range left {
		if containsString(right, item) {
			count++
		}
	}
	return count
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
