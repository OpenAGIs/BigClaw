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
	storagePath string
}

func New(storagePath string) *Store {
	return &Store{storagePath: storagePath}
}

func (s *Store) RememberSuccess(task domain.Task, summary string) error {
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

func (s *Store) SuggestRules(task domain.Task, limit int) (Suggestion, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return Suggestion{}, err
	}
	type scoredPattern struct {
		score   float64
		pattern Pattern
	}
	ranked := make([]scoredPattern, 0, len(patterns))
	for _, pattern := range patterns {
		score := scorePattern(task, pattern)
		if score <= 0 {
			continue
		}
		ranked = append(ranked, scoredPattern{score: score, pattern: pattern})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].pattern.TaskID < ranked[j].pattern.TaskID
		}
		return ranked[i].score > ranked[j].score
	})
	maxPatterns := limit
	if maxPatterns < 1 {
		maxPatterns = 1
	}
	if maxPatterns > len(ranked) {
		maxPatterns = len(ranked)
	}

	acceptance := append([]string(nil), task.AcceptanceCriteria...)
	validation := append([]string(nil), task.ValidationPlan...)
	matched := make([]string, 0, maxPatterns)
	for _, item := range ranked[:maxPatterns] {
		matched = append(matched, item.pattern.TaskID)
		acceptance = appendMissing(acceptance, item.pattern.AcceptanceCriteria)
		validation = appendMissing(validation, item.pattern.ValidationPlan)
	}

	return Suggestion{
		AcceptanceCriteria: acceptance,
		ValidationPlan:     validation,
		MatchedTaskIDs:     matched,
	}, nil
}

func (s *Store) loadPatterns() ([]Pattern, error) {
	payload, err := os.ReadFile(s.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var patterns []Pattern
	if err := json.Unmarshal(payload, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s *Store) writePatterns(patterns []Pattern) error {
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(s.storagePath, payload, 0o644)
}

func scorePattern(task domain.Task, pattern Pattern) float64 {
	return float64(overlap(task.Labels, pattern.Labels)*2 + overlap(task.RequiredTools, pattern.RequiredTools))
}

func overlap(left []string, right []string) int {
	index := make(map[string]struct{}, len(left))
	for _, item := range left {
		index[item] = struct{}{}
	}
	count := 0
	for _, item := range right {
		if _, ok := index[item]; ok {
			count++
		}
	}
	return count
}

func appendMissing(existing []string, candidates []string) []string {
	present := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		present[item] = struct{}{}
	}
	for _, item := range candidates {
		if _, ok := present[item]; ok {
			continue
		}
		existing = append(existing, item)
		present[item] = struct{}{}
	}
	return existing
}
