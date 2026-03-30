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

type TaskMemoryStore struct {
	StoragePath string
}

func (s TaskMemoryStore) loadPatterns() ([]Pattern, error) {
	if s.StoragePath == "" {
		return nil, nil
	}
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
	type rankedPattern struct {
		score   float64
		pattern Pattern
	}
	ranked := make([]rankedPattern, 0, len(patterns))
	for _, pattern := range patterns {
		score := scoreTask(task, pattern)
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
		limit = 3
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

func scoreTask(task domain.Task, pattern Pattern) float64 {
	labelOverlap := overlap(task.Labels, pattern.Labels)
	toolOverlap := overlap(task.RequiredTools, pattern.RequiredTools)
	return float64(labelOverlap*2 + toolOverlap)
}

func overlap(left []string, right []string) int {
	seen := make(map[string]struct{}, len(right))
	for _, item := range right {
		seen[item] = struct{}{}
	}
	total := 0
	for _, item := range left {
		if _, ok := seen[item]; ok {
			total++
		}
	}
	return total
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
