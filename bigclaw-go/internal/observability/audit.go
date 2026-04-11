package observability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"bigclaw-go/internal/domain"
)

type Sink interface {
	Write(domain.Event) error
}

type JSONLAuditSink struct {
	mu   sync.Mutex
	path string
}

func NewJSONLAuditSink(path string) (*JSONLAuditSink, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &JSONLAuditSink{path: path}, nil
}

func (s *JSONLAuditSink) Write(event domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ValidateEvent(event); err != nil {
		return err
	}
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	contents, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = file.Write(append(contents, '\n'))
	return err
}

func ValidateEvent(event domain.Event) error {
	missing := MissingRequiredFieldsForEvent(event)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("audit event %s missing required fields: %s", event.Type, joinFields(missing))
}

func MissingRequiredFieldsForEvent(event domain.Event) []string {
	spec, ok := GetAuditEventSpec(string(event.Type))
	if !ok {
		return nil
	}
	details := make(map[string]any, len(event.Payload)+2)
	for key, value := range event.Payload {
		details[key] = value
	}
	if event.TaskID != "" {
		details["task_id"] = event.TaskID
	}
	if event.RunID != "" {
		details["run_id"] = event.RunID
	}
	missing := make([]string, 0)
	for _, field := range spec.RequiredFields {
		if _, ok := details[field]; !ok {
			missing = append(missing, field)
		}
	}
	return missing
}

func joinFields(fields []string) string {
	if len(fields) == 0 {
		return ""
	}
	out := fields[0]
	for _, field := range fields[1:] {
		out += ", " + field
	}
	return out
}
