package refill

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrLocalIssueNotFound = errors.New("local issue not found")

type LocalIssueStore struct {
	path     string
	issueMap []map[string]any
}

func LoadLocalIssueStore(path string) (*LocalIssueStore, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &LocalIssueStore{path: absolute}, nil
		}
		return nil, err
	}
	issues, err := decodeLocalIssueMaps(body)
	if err != nil {
		return nil, err
	}
	return &LocalIssueStore{path: absolute, issueMap: issues}, nil
}

func decodeLocalIssueMaps(body []byte) ([]map[string]any, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	switch value := decoded.(type) {
	case []any:
		return normalizeLocalIssueMaps(value), nil
	case map[string]any:
		rawIssues, ok := value["issues"]
		if !ok {
			return nil, errors.New("invalid local issue store payload")
		}
		items, ok := rawIssues.([]any)
		if !ok {
			return nil, errors.New("invalid local issue list")
		}
		return normalizeLocalIssueMaps(items), nil
	default:
		return nil, errors.New("invalid local issue store payload")
	}
}

func normalizeLocalIssueMaps(items []any) []map[string]any {
	issues := make([]map[string]any, 0, len(items))
	for _, item := range items {
		issue, ok := item.(map[string]any)
		if !ok {
			continue
		}
		issues = append(issues, issue)
	}
	return issues
}

func (s *LocalIssueStore) IssueStates(stateNames []string) []LinearIssue {
	wanted := map[string]struct{}{}
	for _, stateName := range stateNames {
		trimmed := strings.TrimSpace(stateName)
		if trimmed != "" {
			wanted[trimmed] = struct{}{}
		}
	}
	issues := make([]LinearIssue, 0, len(s.issueMap))
	for _, issue := range s.issueMap {
		stateName := mapString(issue, "state")
		if len(wanted) != 0 {
			if _, ok := wanted[stateName]; !ok {
				continue
			}
		}
		issues = append(issues, LinearIssue{
			ID:         mapString(issue, "id"),
			Identifier: mapString(issue, "identifier"),
			StateName:  stateName,
		})
	}
	return issues
}

func (s *LocalIssueStore) UpdateIssueState(ref string, stateName string, now time.Time) (string, error) {
	for _, issue := range s.issueMap {
		if !issueMatchesRef(issue, ref) {
			continue
		}
		issue["state"] = stateName
		issue["updated_at"] = now.UTC().Truncate(time.Second).Format(time.RFC3339)
		if err := s.Save(); err != nil {
			return "", err
		}
		return mapString(issue, "state"), nil
	}
	return "", ErrLocalIssueNotFound
}

func (s *LocalIssueStore) Save() error {
	issues := s.issueMap
	if issues == nil {
		issues = []map[string]any{}
	}
	payload := map[string]any{"issues": issues}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path, append(body, '\n'), 0o644)
}

func issueMatchesRef(issue map[string]any, ref string) bool {
	return mapString(issue, "id") == ref || mapString(issue, "identifier") == ref
}

func mapString(issue map[string]any, key string) string {
	value, ok := issue[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
