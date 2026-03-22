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

type LocalIssue struct {
	ID               string
	Identifier       string
	Title            string
	Description      string
	State            string
	Priority         int
	Labels           []string
	AssignedToWorker bool
	CreatedAt        string
	UpdatedAt        string
}

type LocalIssueComment struct {
	Author    string
	CreatedAt time.Time
	Body      string
}

type LocalIssueCreateParams struct {
	ID               string
	Identifier       string
	Title            string
	Description      string
	State            string
	Priority         int
	Labels           []string
	AssignedToWorker bool
	CreatedAt        time.Time
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

func (s *LocalIssueStore) Issues() []LocalIssue {
	issues := make([]LocalIssue, 0, len(s.issueMap))
	for _, issue := range s.issueMap {
		issues = append(issues, LocalIssue{
			ID:               mapString(issue, "id"),
			Identifier:       mapString(issue, "identifier"),
			Title:            mapString(issue, "title"),
			Description:      mapString(issue, "description"),
			State:            mapString(issue, "state"),
			Priority:         mapInt(issue, "priority"),
			Labels:           mapStringSlice(issue, "labels"),
			AssignedToWorker: mapBool(issue, "assigned_to_worker"),
			CreatedAt:        mapString(issue, "created_at"),
			UpdatedAt:        mapString(issue, "updated_at"),
		})
	}
	return issues
}

func (s *LocalIssueStore) FindIssue(ref string) (LocalIssue, bool) {
	for _, issue := range s.issueMap {
		if !issueMatchesRef(issue, ref) {
			continue
		}
		return LocalIssue{
			ID:               mapString(issue, "id"),
			Identifier:       mapString(issue, "identifier"),
			Title:            mapString(issue, "title"),
			Description:      mapString(issue, "description"),
			State:            mapString(issue, "state"),
			Priority:         mapInt(issue, "priority"),
			Labels:           mapStringSlice(issue, "labels"),
			AssignedToWorker: mapBool(issue, "assigned_to_worker"),
			CreatedAt:        mapString(issue, "created_at"),
			UpdatedAt:        mapString(issue, "updated_at"),
		}, true
	}
	return LocalIssue{}, false
}

func (s *LocalIssueStore) CreateIssue(params LocalIssueCreateParams) (LocalIssue, error) {
	identifier := strings.TrimSpace(params.Identifier)
	if identifier == "" {
		return LocalIssue{}, errors.New("identifier is required")
	}
	title := strings.TrimSpace(params.Title)
	if title == "" {
		return LocalIssue{}, errors.New("title is required")
	}
	id := strings.TrimSpace(params.ID)
	if id == "" {
		id = strings.ToLower(strings.ReplaceAll(identifier, "_", "-"))
	}

	for _, issue := range s.issueMap {
		if strings.EqualFold(mapString(issue, "id"), id) || strings.EqualFold(mapString(issue, "identifier"), identifier) {
			return LocalIssue{}, fmt.Errorf("issue %q already exists", identifier)
		}
	}

	createdAt := params.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	createdAt = createdAt.UTC().Truncate(time.Second)
	createdAtString := createdAt.Format(time.RFC3339)

	stateName := strings.TrimSpace(params.State)
	if stateName == "" {
		stateName = "Todo"
	}

	entry := map[string]any{
		"assigned_to_worker": params.AssignedToWorker,
		"created_at":         createdAtString,
		"description":        strings.TrimSpace(params.Description),
		"id":                 id,
		"identifier":         identifier,
		"labels":             params.Labels,
		"priority":           params.Priority,
		"state":              stateName,
		"title":              title,
		"updated_at":         createdAtString,
	}
	s.issueMap = append(s.issueMap, entry)

	if err := s.Save(); err != nil {
		return LocalIssue{}, err
	}
	return LocalIssue{
		ID:               id,
		Identifier:       identifier,
		Title:            title,
		Description:      strings.TrimSpace(params.Description),
		State:            stateName,
		Priority:         params.Priority,
		Labels:           params.Labels,
		AssignedToWorker: params.AssignedToWorker,
		CreatedAt:        createdAtString,
		UpdatedAt:        createdAtString,
	}, nil
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

func issueCommentList(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return []map[string]any{}
	}
	comments := make([]map[string]any, 0, len(items))
	for _, item := range items {
		comment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		comments = append(comments, comment)
	}
	return comments
}

func (s *LocalIssueStore) IssueStates(stateNames []string) []TrackedIssue {
	wanted := map[string]struct{}{}
	for _, stateName := range stateNames {
		trimmed := strings.TrimSpace(stateName)
		if trimmed != "" {
			wanted[trimmed] = struct{}{}
		}
	}
	issues := make([]TrackedIssue, 0, len(s.issueMap))
	for _, issue := range s.issueMap {
		stateName := mapString(issue, "state")
		if len(wanted) != 0 {
			if _, ok := wanted[stateName]; !ok {
				continue
			}
		}
		issues = append(issues, TrackedIssue{
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

func (s *LocalIssueStore) AddComment(ref string, comment LocalIssueComment) error {
	body := strings.TrimSpace(comment.Body)
	if body == "" {
		return errors.New("comment body is required")
	}
	author := strings.TrimSpace(comment.Author)
	if author == "" {
		author = "codex"
	}
	createdAt := comment.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	timestamp := createdAt.UTC().Truncate(time.Second).Format(time.RFC3339)
	entry := map[string]any{
		"author":     author,
		"created_at": timestamp,
		"body":       body,
	}
	for _, issue := range s.issueMap {
		if !issueMatchesRef(issue, ref) {
			continue
		}
		comments := issueCommentList(issue["comments"])
		comments = append(comments, entry)
		issue["comments"] = comments
		issue["updated_at"] = timestamp
		return s.Save()
	}
	return ErrLocalIssueNotFound
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
	return strings.EqualFold(mapString(issue, "id"), ref) || strings.EqualFold(mapString(issue, "identifier"), ref)
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

func mapInt(issue map[string]any, key string) int {
	value, ok := issue[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}

func mapBool(issue map[string]any, key string) bool {
	value, ok := issue[key]
	if !ok || value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	default:
		return false
	}
}

func mapStringSlice(issue map[string]any, key string) []string {
	value, ok := issue[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text == "" {
				continue
			}
			out = append(out, text)
		}
		return out
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(item)
			if text == "" {
				continue
			}
			out = append(out, text)
		}
		return out
	default:
		return nil
	}
}
