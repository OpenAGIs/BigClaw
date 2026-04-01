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

type RuleSuggestion struct {
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
	MatchedTaskIDs     []string `json:"matched_task_ids"`
}

type TaskMemoryStore struct {
	storagePath string
}

func NewTaskMemoryStore(storagePath string) TaskMemoryStore {
	return TaskMemoryStore{storagePath: storagePath}
}

func (s TaskMemoryStore) RememberSuccess(task domain.Task, summary string) error {
	patterns, err := s.loadPatterns()
	if err != nil {
		return err
	}
	filtered := make([]Pattern, 0, len(patterns))
	for _, item := range patterns {
		if item.TaskID != task.ID {
			filtered = append(filtered, item)
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

func (s TaskMemoryStore) SuggestRules(task domain.Task, limit int) (RuleSuggestion, error) {
	patterns, err := s.loadPatterns()
	if err != nil {
		return RuleSuggestion{}, err
	}
	type rankedPattern struct {
		score   float64
		pattern Pattern
	}
	ranked := make([]rankedPattern, 0, len(patterns))
	for _, pattern := range patterns {
		score := scoreTask(task, pattern)
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
	if limit < 1 {
		limit = 1
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	acceptance := append([]string(nil), task.AcceptanceCriteria...)
	validation := append([]string(nil), task.ValidationPlan...)
	matchedTaskIDs := make([]string, 0, len(ranked))
	for _, item := range ranked {
		matchedTaskIDs = append(matchedTaskIDs, item.pattern.TaskID)
		acceptance = appendUnique(acceptance, item.pattern.AcceptanceCriteria)
		validation = appendUnique(validation, item.pattern.ValidationPlan)
	}

	return RuleSuggestion{
		AcceptanceCriteria: acceptance,
		ValidationPlan:     validation,
		MatchedTaskIDs:     matchedTaskIDs,
	}, nil
}

func (s TaskMemoryStore) loadPatterns() ([]Pattern, error) {
	body, err := os.ReadFile(s.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}
	var patterns []Pattern
	if err := json.Unmarshal(body, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s TaskMemoryStore) writePatterns(patterns []Pattern) error {
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storagePath, body, 0o644)
}

func scoreTask(task domain.Task, pattern Pattern) float64 {
	labelOverlap := overlapCount(task.Labels, pattern.Labels)
	toolOverlap := overlapCount(task.RequiredTools, pattern.RequiredTools)
	return float64(labelOverlap*2 + toolOverlap)
}

func overlapCount(left []string, right []string) int {
	seen := make(map[string]struct{}, len(left))
	for _, item := range left {
		seen[item] = struct{}{}
	}
	count := 0
	for _, item := range right {
		if _, ok := seen[item]; ok {
			count++
		}
	}
	return count
}

func appendUnique(existing []string, extra []string) []string {
	out := append([]string(nil), existing...)
	seen := make(map[string]struct{}, len(out))
	for _, item := range out {
		seen[item] = struct{}{}
	}
	for _, item := range extra {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
