package refill

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrLocalIssueNotFound = errors.New("local issue not found")
var ErrLocalIssueAlreadyExists = errors.New("local issue already exists")

type LocalIssueStore struct {
	path   string
	issues []localIssue
}

type LocalIssueRecord struct {
	ID         string   `json:"id"`
	Identifier string   `json:"identifier"`
	Title      string   `json:"title"`
	State      string   `json:"state"`
	Priority   any      `json:"priority,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	CreatedAt  string   `json:"created_at,omitempty"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
}

type localIssueComment struct {
	Body      string
	CreatedAt string
	extra     map[string]json.RawMessage
}

type localIssue struct {
	ID               string
	Identifier       string
	Title            string
	Description      string
	Priority         json.RawMessage
	State            string
	Labels           json.RawMessage
	AssignedToWorker json.RawMessage
	Comments         []localIssueComment
	hasComments      bool
	CreatedAt        string
	UpdatedAt        string
	extra            map[string]json.RawMessage
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
	issues, err := decodeLocalIssues(body)
	if err != nil {
		return nil, err
	}
	return &LocalIssueStore{path: absolute, issues: issues}, nil
}

func decodeLocalIssues(body []byte) ([]localIssue, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, nil
	}
	store := struct {
		Issues []localIssue `json:"issues"`
	}{}
	if err := json.Unmarshal(body, &store); err == nil && store.Issues != nil {
		return store.Issues, nil
	}
	issues := []localIssue{}
	if err := json.Unmarshal(body, &issues); err == nil {
		return issues, nil
	}
	return nil, errors.New("invalid local issue store payload")
}

func (s *LocalIssueStore) IssueStates(stateNames []string) []LinearIssue {
	wanted := map[string]struct{}{}
	for _, stateName := range stateNames {
		trimmed := strings.TrimSpace(stateName)
		if trimmed != "" {
			wanted[trimmed] = struct{}{}
		}
	}
	issues := make([]LinearIssue, 0, len(s.issues))
	for _, issue := range s.issues {
		stateName := strings.TrimSpace(issue.State)
		if len(wanted) != 0 {
			if _, ok := wanted[stateName]; !ok {
				continue
			}
		}
		issues = append(issues, LinearIssue{
			ID:         strings.TrimSpace(issue.ID),
			Identifier: strings.TrimSpace(issue.Identifier),
			StateName:  stateName,
		})
	}
	return issues
}

func (s *LocalIssueStore) ListIssues(stateNames []string) []LocalIssueRecord {
	wanted := map[string]struct{}{}
	for _, stateName := range stateNames {
		trimmed := strings.TrimSpace(stateName)
		if trimmed != "" {
			wanted[trimmed] = struct{}{}
		}
	}
	issues := make([]LocalIssueRecord, 0, len(s.issues))
	for _, issue := range s.issues {
		stateName := strings.TrimSpace(issue.State)
		if len(wanted) != 0 {
			if _, ok := wanted[stateName]; !ok {
				continue
			}
		}
		record := LocalIssueRecord{
			ID:         strings.TrimSpace(issue.ID),
			Identifier: strings.TrimSpace(issue.Identifier),
			Title:      strings.TrimSpace(issue.Title),
			State:      stateName,
			Priority:   decodeJSONScalar(issue.Priority),
			Labels:     decodeJSONStringSlice(issue.Labels),
			CreatedAt:  strings.TrimSpace(issue.CreatedAt),
			UpdatedAt:  strings.TrimSpace(issue.UpdatedAt),
		}
		issues = append(issues, record)
	}
	return issues
}

func (s *LocalIssueStore) UpdateIssueState(ref string, stateName string, now time.Time) (string, error) {
	for i := range s.issues {
		issue := &s.issues[i]
		if !issueMatchesRef(issue, ref) {
			continue
		}
		issue.State = stateName
		issue.UpdatedAt = now.UTC().Truncate(time.Second).Format(time.RFC3339)
		if err := s.Save(); err != nil {
			return "", err
		}
		return strings.TrimSpace(issue.State), nil
	}
	return "", ErrLocalIssueNotFound
}

type LocalIssueCreateInput struct {
	ID               string
	Identifier       string
	Title            string
	Description      string
	Priority         any
	State            string
	Labels           []string
	AssignedToWorker bool
}

func (s *LocalIssueStore) CreateIssue(input LocalIssueCreateInput, now time.Time) (*LinearIssue, error) {
	identifier := strings.TrimSpace(input.Identifier)
	if identifier == "" {
		return nil, errors.New("issue identifier is required")
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, errors.New("issue title is required")
	}
	issueID := strings.TrimSpace(input.ID)
	if issueID == "" {
		issueID = strings.ToLower(identifier)
	}
	for i := range s.issues {
		issue := &s.issues[i]
		if issueMatchesRef(issue, issueID) || issueMatchesRef(issue, identifier) {
			return nil, ErrLocalIssueAlreadyExists
		}
	}
	stateName := strings.TrimSpace(input.State)
	if stateName == "" {
		stateName = "Backlog"
	}
	timestamp := now.UTC().Truncate(time.Second).Format(time.RFC3339)
	created := localIssue{
		ID:               issueID,
		Identifier:       identifier,
		Title:            title,
		Description:      strings.TrimSpace(input.Description),
		Priority:         mustMarshalJSON(input.Priority),
		State:            stateName,
		Labels:           mustMarshalJSON(input.Labels),
		AssignedToWorker: mustMarshalJSON(input.AssignedToWorker),
		CreatedAt:        timestamp,
		UpdatedAt:        timestamp,
	}
	s.issues = append(s.issues, created)
	if err := s.Save(); err != nil {
		return nil, err
	}
	return &LinearIssue{
		ID:         issueID,
		Identifier: identifier,
		StateName:  stateName,
	}, nil
}

func (s *LocalIssueStore) AppendIssueComment(ref string, body string, now time.Time) error {
	trimmedBody := strings.TrimSpace(body)
	if trimmedBody == "" {
		return errors.New("comment body is required")
	}
	for i := range s.issues {
		issue := &s.issues[i]
		if !issueMatchesRef(issue, ref) {
			continue
		}
		issue.Comments = append(issue.Comments, localIssueComment{
			Body:      trimmedBody,
			CreatedAt: now.UTC().Truncate(time.Second).Format(time.RFC3339),
		})
		issue.hasComments = true
		issue.UpdatedAt = now.UTC().Truncate(time.Second).Format(time.RFC3339)
		return s.Save()
	}
	return ErrLocalIssueNotFound
}

func (s *LocalIssueStore) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(struct {
		Issues []localIssue `json:"issues"`
	}{Issues: s.issues}); err != nil {
		return err
	}
	return os.WriteFile(s.path, buffer.Bytes(), 0o644)
}

func issueMatchesRef(issue *localIssue, ref string) bool {
	trimmed := strings.TrimSpace(ref)
	return strings.TrimSpace(issue.ID) == trimmed || strings.TrimSpace(issue.Identifier) == trimmed
}

func (c *localIssueComment) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.extra = map[string]json.RawMessage{}
	for key, value := range raw {
		switch key {
		case "body":
			if err := json.Unmarshal(value, &c.Body); err != nil {
				return err
			}
		case "created_at":
			if err := json.Unmarshal(value, &c.CreatedAt); err != nil {
				return err
			}
		default:
			c.extra[key] = value
		}
	}
	return nil
}

func (c localIssueComment) MarshalJSON() ([]byte, error) {
	fields := []jsonField{
		{name: "body", raw: mustMarshalJSON(c.Body)},
		{name: "created_at", raw: mustMarshalJSON(c.CreatedAt)},
	}
	fields = append(fields, extraFields(c.extra)...)
	return marshalOrderedObject(fields)
}

func (i *localIssue) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	i.extra = map[string]json.RawMessage{}
	for key, value := range raw {
		switch key {
		case "id":
			if err := json.Unmarshal(value, &i.ID); err != nil {
				return err
			}
		case "identifier":
			if err := json.Unmarshal(value, &i.Identifier); err != nil {
				return err
			}
		case "title":
			if err := json.Unmarshal(value, &i.Title); err != nil {
				return err
			}
		case "description":
			if err := json.Unmarshal(value, &i.Description); err != nil {
				return err
			}
		case "priority":
			i.Priority = append([]byte(nil), value...)
		case "state":
			if err := json.Unmarshal(value, &i.State); err != nil {
				return err
			}
		case "labels":
			i.Labels = append([]byte(nil), value...)
		case "assigned_to_worker":
			i.AssignedToWorker = append([]byte(nil), value...)
		case "comments":
			if err := json.Unmarshal(value, &i.Comments); err != nil {
				return err
			}
			i.hasComments = true
		case "created_at":
			if err := json.Unmarshal(value, &i.CreatedAt); err != nil {
				return err
			}
		case "updated_at":
			if err := json.Unmarshal(value, &i.UpdatedAt); err != nil {
				return err
			}
		default:
			i.extra[key] = value
		}
	}
	return nil
}

func (i localIssue) MarshalJSON() ([]byte, error) {
	fields := []jsonField{
		{name: "id", raw: mustMarshalJSON(i.ID)},
		{name: "identifier", raw: mustMarshalJSON(i.Identifier)},
		{name: "title", raw: mustMarshalJSON(i.Title)},
		{name: "description", raw: mustMarshalJSON(i.Description)},
		{name: "priority", raw: jsonOrNull(i.Priority)},
		{name: "state", raw: mustMarshalJSON(i.State)},
		{name: "labels", raw: jsonOrNull(i.Labels)},
		{name: "assigned_to_worker", raw: jsonOrNull(i.AssignedToWorker)},
		{name: "created_at", raw: mustMarshalJSON(i.CreatedAt)},
		{name: "updated_at", raw: mustMarshalJSON(i.UpdatedAt)},
	}
	if i.hasComments || len(i.Comments) != 0 {
		fields = append(fields, jsonField{name: "comments", raw: mustMarshalJSON(i.Comments)})
	}
	fields = append(fields, extraFields(i.extra)...)
	return marshalOrderedObject(fields)
}

type jsonField struct {
	name string
	raw  []byte
}

func extraFields(extra map[string]json.RawMessage) []jsonField {
	if len(extra) == 0 {
		return nil
	}
	keys := make([]string, 0, len(extra))
	for key := range extra {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]jsonField, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, jsonField{name: key, raw: extra[key]})
	}
	return fields
}

func marshalOrderedObject(fields []jsonField) ([]byte, error) {
	buffer := &bytes.Buffer{}
	buffer.WriteByte('{')
	for index, field := range fields {
		if index > 0 {
			buffer.WriteByte(',')
		}
		name, err := json.Marshal(field.name)
		if err != nil {
			return nil, err
		}
		buffer.Write(name)
		buffer.WriteByte(':')
		buffer.Write(field.raw)
	}
	buffer.WriteByte('}')
	return buffer.Bytes(), nil
}

func mustMarshalJSON(value any) []byte {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(value)
	if err != nil {
		panic(fmt.Sprintf("marshal json: %v", err))
	}
	return bytes.TrimSpace(buffer.Bytes())
}

func jsonOrNull(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("null")
	}
	return raw
}

func decodeJSONScalar(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func decodeJSONStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	items := []string{}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	return items
}
