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
	Labels             []string `json:"labels"`
	RequiredTools      []string `json:"required_tools"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
	Summary            string   `json:"summary"`
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

func (s *Store) SuggestRules(task domain.Task, limit int) (Suggestion, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return Suggestion{}, err
	}

	type rankedPattern struct {
		score   int
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

func (s *Store) loadPatterns() ([]Pattern, error) {
	body, err := os.ReadFile(s.path)
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

func (s *Store) writePatterns(patterns []Pattern) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, body, 0o644)
}

func scorePattern(task domain.Task, pattern Pattern) int {
	score := 0
	for _, label := range task.Labels {
		if contains(pattern.Labels, label) {
			score += 2
		}
	}
	for _, tool := range task.RequiredTools {
		if contains(pattern.RequiredTools, tool) {
			score++
		}
	}
	return score
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
