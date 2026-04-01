package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"bigclaw-go/internal/domain"
)

type Store struct {
	path string
}

type Suggestion struct {
	MatchedTaskIDs     []string `json:"matched_task_ids"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
}

type rememberedTask struct {
	TaskID             string   `json:"task_id"`
	Labels             []string `json:"labels,omitempty"`
	RequiredTools      []string `json:"required_tools,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ValidationPlan     []string `json:"validation_plan,omitempty"`
	Summary            string   `json:"summary,omitempty"`
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) RememberSuccess(task domain.Task, summary string) error {
	entries, err := s.load()
	if err != nil {
		return err
	}
	entries = append(entries, rememberedTask{
		TaskID:             task.ID,
		Labels:             dedupeStrings(task.Labels),
		RequiredTools:      dedupeStrings(task.RequiredTools),
		AcceptanceCriteria: dedupeStrings(task.AcceptanceCriteria),
		ValidationPlan:     dedupeStrings(task.ValidationPlan),
		Summary:            summary,
	})
	return s.save(entries)
}

func (s *Store) SuggestRules(task domain.Task) (Suggestion, error) {
	entries, err := s.load()
	if err != nil {
		return Suggestion{}, err
	}
	suggestion := Suggestion{
		AcceptanceCriteria: dedupeStrings(task.AcceptanceCriteria),
		ValidationPlan:     dedupeStrings(task.ValidationPlan),
	}
	for _, entry := range entries {
		if !matches(task, entry) {
			continue
		}
		suggestion.MatchedTaskIDs = append(suggestion.MatchedTaskIDs, entry.TaskID)
		suggestion.AcceptanceCriteria = mergeStrings(suggestion.AcceptanceCriteria, entry.AcceptanceCriteria)
		suggestion.ValidationPlan = mergeStrings(suggestion.ValidationPlan, entry.ValidationPlan)
	}
	suggestion.MatchedTaskIDs = dedupeStrings(suggestion.MatchedTaskIDs)
	return suggestion, nil
}

func matches(task domain.Task, entry rememberedTask) bool {
	return overlap(task.Labels, entry.Labels) || overlap(task.RequiredTools, entry.RequiredTools)
}

func overlap(left []string, right []string) bool {
	seen := make(map[string]struct{}, len(left))
	for _, item := range left {
		if item == "" {
			continue
		}
		seen[item] = struct{}{}
	}
	for _, item := range right {
		if _, ok := seen[item]; ok {
			return true
		}
	}
	return false
}

func (s *Store) load() ([]rememberedTask, error) {
	if s == nil || s.path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []rememberedTask
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) save(entries []rememberedTask) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o644)
}

func mergeStrings(current []string, additional []string) []string {
	return dedupeStrings(append(append([]string(nil), current...), additional...))
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
