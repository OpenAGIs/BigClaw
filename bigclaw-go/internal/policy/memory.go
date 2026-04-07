package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"bigclaw-go/internal/domain"
)

type MemoryPattern struct {
	TaskID             string   `json:"task_id"`
	Title              string   `json:"title"`
	Labels             []string `json:"labels"`
	RequiredTools      []string `json:"required_tools"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
	Summary            string   `json:"summary"`
}

type TaskMemorySuggestion struct {
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
	MatchedTaskIDs     []string `json:"matched_task_ids"`
}

type TaskMemoryStore struct {
	storagePath string
}

func NewTaskMemoryStore(storagePath string) *TaskMemoryStore {
	return &TaskMemoryStore{storagePath: storagePath}
}

func (s *TaskMemoryStore) loadPatterns() ([]MemoryPattern, error) {
	payload, err := os.ReadFile(s.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var patterns []MemoryPattern
	if err := json.Unmarshal(payload, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s *TaskMemoryStore) writePatterns(patterns []MemoryPattern) error {
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storagePath, payload, 0o644)
}

func (s *TaskMemoryStore) RememberSuccess(task domain.Task, summary string) error {
	patterns, err := s.loadPatterns()
	if err != nil {
		return err
	}

	filtered := make([]MemoryPattern, 0, len(patterns)+1)
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

func (s *TaskMemoryStore) SuggestRules(task domain.Task, limit int) (TaskMemorySuggestion, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return TaskMemorySuggestion{}, err
	}

	type rankedPattern struct {
		score   float64
		pattern MemoryPattern
	}
	ranked := make([]rankedPattern, 0, len(patterns))
	for _, pattern := range patterns {
		score := scoreMemoryPattern(task, pattern)
		if score <= 0 {
			continue
		}
		ranked = append(ranked, rankedPattern{score: score, pattern: pattern})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

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
		for _, candidate := range item.pattern.AcceptanceCriteria {
			if !containsString(acceptance, candidate) {
				acceptance = append(acceptance, candidate)
			}
		}
		for _, candidate := range item.pattern.ValidationPlan {
			if !containsString(validation, candidate) {
				validation = append(validation, candidate)
			}
		}
	}

	return TaskMemorySuggestion{
		AcceptanceCriteria: acceptance,
		ValidationPlan:     validation,
		MatchedTaskIDs:     matched,
	}, nil
}

func scoreMemoryPattern(task domain.Task, pattern MemoryPattern) float64 {
	labelOverlap := overlapCount(task.Labels, pattern.Labels)
	toolOverlap := overlapCount(task.RequiredTools, pattern.RequiredTools)
	return float64(labelOverlap*2 + toolOverlap)
}

func overlapCount(left []string, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	lookup := make(map[string]struct{}, len(left))
	for _, item := range left {
		lookup[item] = struct{}{}
	}
	count := 0
	for _, item := range right {
		if _, ok := lookup[item]; ok {
			count++
		}
	}
	return count
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
