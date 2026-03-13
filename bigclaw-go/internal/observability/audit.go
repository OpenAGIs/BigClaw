package observability

import (
	"encoding/json"
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
