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
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
	MatchedTaskIDs     []string `json:"matched_task_ids,omitempty"`
}

type Store struct {
	StoragePath string
}

type scoredPattern struct {
	score   float64
	pattern Pattern
}

func (s Store) loadPatterns() ([]Pattern, error) {
	if _, err := os.Stat(s.StoragePath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	body, err := os.ReadFile(s.StoragePath)
	if err != nil {
		return nil, err
	}
	var patterns []Pattern
	if err := json.Unmarshal(body, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s Store) writePatterns(patterns []Pattern) error {
	if err := os.MkdirAll(filepath.Dir(s.StoragePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.StoragePath, body, 0o644)
}

func (s Store) RememberSuccess(task domain.Task, summary string) error {
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

func (s Store) SuggestRules(task domain.Task, limit int) (Suggestion, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return Suggestion{}, err
	}

	ranked := make([]scoredPattern, 0, len(patterns))
	for _, pattern := range patterns {
		score := scorePattern(task, pattern)
		if score > 0 {
			ranked = append(ranked, scoredPattern{score: score, pattern: pattern})
		}
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

func scorePattern(task domain.Task, pattern Pattern) float64 {
	return float64(overlap(task.Labels, pattern.Labels)*2 + overlap(task.RequiredTools, pattern.RequiredTools))
}

func overlap(left, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(left))
	for _, item := range left {
		seen[item] = struct{}{}
	}
	total := 0
	for _, item := range right {
		if _, ok := seen[item]; ok {
			total++
		}
	}
	return total
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
