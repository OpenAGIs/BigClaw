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

type TaskStore struct {
	storagePath string
}

func NewTaskStore(storagePath string) TaskStore {
	return TaskStore{storagePath: storagePath}
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

func (s TaskStore) SuggestRules(task domain.Task, limit int) (Suggestion, error) {
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
		score := scoreTask(task, pattern)
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
	matchedIDs := make([]string, 0, len(ranked))
	for _, item := range ranked {
		matchedIDs = append(matchedIDs, item.pattern.TaskID)
		appendUnique(&acceptance, item.pattern.AcceptanceCriteria...)
		appendUnique(&validation, item.pattern.ValidationPlan...)
	}
	return Suggestion{
		AcceptanceCriteria: acceptance,
		ValidationPlan:     validation,
		MatchedTaskIDs:     matchedIDs,
	}, nil
}

func (s TaskStore) loadPatterns() ([]Pattern, error) {
	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var patterns []Pattern
	if err := json.Unmarshal(data, &patterns); err != nil {
		return nil, err
	}
	for i := range patterns {
		patterns[i].Labels = sliceOrEmpty(patterns[i].Labels)
		patterns[i].RequiredTools = sliceOrEmpty(patterns[i].RequiredTools)
		patterns[i].AcceptanceCriteria = sliceOrEmpty(patterns[i].AcceptanceCriteria)
		patterns[i].ValidationPlan = sliceOrEmpty(patterns[i].ValidationPlan)
	}
	return patterns, nil
}

func (s TaskStore) writePatterns(patterns []Pattern) error {
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storagePath, payload, 0o644)
}

func scoreTask(task domain.Task, pattern Pattern) float64 {
	labelOverlap := overlapCount(task.Labels, pattern.Labels)
	toolOverlap := overlapCount(task.RequiredTools, pattern.RequiredTools)
	return float64(labelOverlap*2 + toolOverlap)
}

func overlapCount(left []string, right []string) int {
	rightSet := make(map[string]struct{}, len(right))
	for _, item := range right {
		rightSet[item] = struct{}{}
	}
	count := 0
	for _, item := range left {
		if _, ok := rightSet[item]; ok {
			count++
		}
	}
	return count
}

func appendUnique(target *[]string, values ...string) {
	seen := make(map[string]struct{}, len(*target))
	for _, item := range *target {
		seen[item] = struct{}{}
	}
	for _, item := range values {
		if _, ok := seen[item]; ok {
			continue
		}
		*target = append(*target, item)
		seen[item] = struct{}{}
	}
}

func sliceOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
