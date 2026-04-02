package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

type TaskMemoryStore struct {
	path string
}

type Suggestion struct {
	MatchedTaskIDs     []string `json:"matched_task_ids"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
}

type successPattern struct {
	TaskID             string   `json:"task_id"`
	Summary            string   `json:"summary,omitempty"`
	Labels             []string `json:"labels,omitempty"`
	RequiredTools      []string `json:"required_tools,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
}

func NewTaskMemoryStore(path string) *TaskMemoryStore {
	return &TaskMemoryStore{path: path}
}

func (s *TaskMemoryStore) RememberSuccess(task domain.Task, summary string) error {
	patterns, err := s.load()
	if err != nil {
		return err
	}

	next := successPattern{
		TaskID:             task.ID,
		Summary:            summary,
		Labels:             append([]string(nil), task.Labels...),
		RequiredTools:      append([]string(nil), task.RequiredTools...),
		AcceptanceCriteria: append([]string(nil), task.AcceptanceCriteria...),
		ValidationPlan:     append([]string(nil), task.ValidationPlan...),
	}

	replaced := false
	for i := range patterns {
		if patterns[i].TaskID == task.ID {
			patterns[i] = next
			replaced = true
			break
		}
	}
	if !replaced {
		patterns = append(patterns, next)
	}

	return s.save(patterns)
}

func (s *TaskMemoryStore) SuggestRules(task domain.Task) (Suggestion, error) {
	patterns, err := s.load()
	if err != nil {
		return Suggestion{}, err
	}

	type match struct {
		pattern successPattern
		score   int
	}
	matches := make([]match, 0)
	for _, pattern := range patterns {
		score := overlapCount(task.Labels, pattern.Labels) + overlapCount(task.RequiredTools, pattern.RequiredTools)
		if score == 0 {
			continue
		}
		matches = append(matches, match{pattern: pattern, score: score})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].pattern.TaskID < matches[j].pattern.TaskID
		}
		return matches[i].score > matches[j].score
	})

	suggestion := Suggestion{
		AcceptanceCriteria: append([]string(nil), task.AcceptanceCriteria...),
		ValidationPlan:     append([]string(nil), task.ValidationPlan...),
	}
	seenAcceptance := make(map[string]struct{}, len(suggestion.AcceptanceCriteria))
	for _, item := range suggestion.AcceptanceCriteria {
		seenAcceptance[strings.TrimSpace(item)] = struct{}{}
	}
	seenValidation := make(map[string]struct{}, len(suggestion.ValidationPlan))
	for _, item := range suggestion.ValidationPlan {
		seenValidation[strings.TrimSpace(item)] = struct{}{}
	}

	for _, matched := range matches {
		suggestion.MatchedTaskIDs = append(suggestion.MatchedTaskIDs, matched.pattern.TaskID)
		for _, item := range matched.pattern.AcceptanceCriteria {
			key := strings.TrimSpace(item)
			if key == "" {
				continue
			}
			if _, ok := seenAcceptance[key]; ok {
				continue
			}
			seenAcceptance[key] = struct{}{}
			suggestion.AcceptanceCriteria = append(suggestion.AcceptanceCriteria, item)
		}
		for _, item := range matched.pattern.ValidationPlan {
			key := strings.TrimSpace(item)
			if key == "" {
				continue
			}
			if _, ok := seenValidation[key]; ok {
				continue
			}
			seenValidation[key] = struct{}{}
			suggestion.ValidationPlan = append(suggestion.ValidationPlan, item)
		}
	}

	return suggestion, nil
}

func (s *TaskMemoryStore) load() ([]successPattern, error) {
	if strings.TrimSpace(s.path) == "" {
		return nil, nil
	}
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}
	var patterns []successPattern
	if err := json.Unmarshal(body, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s *TaskMemoryStore) save(patterns []successPattern) error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, body, 0o644)
}

func overlapCount(left []string, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	rightSet := make(map[string]struct{}, len(right))
	for _, item := range right {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		rightSet[key] = struct{}{}
	}
	total := 0
	seen := make(map[string]struct{}, len(left))
	for _, item := range left {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := rightSet[key]; ok {
			total++
		}
	}
	return total
}
