package eventbus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PullRequestCommentEvent = "pull_request_comment"
	CICompletedEvent        = "ci_completed"
	TaskFailedEvent         = "task_failed"
)

type BusEvent struct {
	EventType string         `json:"event_type"`
	RunID     string         `json:"run_id"`
	Actor     string         `json:"actor,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

type AuditRecord struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor,omitempty"`
	Outcome string         `json:"outcome,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type RunRecord struct {
	RunID   string        `json:"run_id"`
	TaskID  string        `json:"task_id"`
	Status  string        `json:"status"`
	Summary string        `json:"summary,omitempty"`
	Audits  []AuditRecord `json:"audits,omitempty"`
}

type Ledger struct {
	path string
}

func NewLedger(path string) *Ledger {
	return &Ledger{path: path}
}

func (l *Ledger) Append(record RunRecord) error {
	records, err := l.Load()
	if err != nil {
		return err
	}
	records = append(records, record)
	return l.save(records)
}

func (l *Ledger) Load() ([]RunRecord, error) {
	body, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}
	var records []RunRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (l *Ledger) save(records []RunRecord) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, body, 0o644)
}

type Subscriber func(BusEvent, RunRecord)

type EventBus struct {
	ledger      *Ledger
	subscribers map[string][]Subscriber
}

func NewEventBus(ledger *Ledger) *EventBus {
	return &EventBus{
		ledger:      ledger,
		subscribers: make(map[string][]Subscriber),
	}
}

func (b *EventBus) Subscribe(eventType string, fn Subscriber) {
	b.subscribers[strings.TrimSpace(eventType)] = append(b.subscribers[strings.TrimSpace(eventType)], fn)
}

func (b *EventBus) Publish(event BusEvent) (RunRecord, error) {
	records, err := b.ledger.Load()
	if err != nil {
		return RunRecord{}, err
	}
	index := -1
	for i := range records {
		if records[i].RunID == event.RunID {
			index = i
			break
		}
	}
	if index == -1 {
		return RunRecord{}, fmt.Errorf("run %s not found", event.RunID)
	}

	record := records[index]
	record.Audits = append(record.Audits, AuditRecord{
		Action:  "event_bus.event",
		Actor:   strings.TrimSpace(event.Actor),
		Outcome: "received",
		Details: map[string]any{"event_type": event.EventType},
	})

	previousStatus := record.Status
	switch strings.TrimSpace(event.EventType) {
	case PullRequestCommentEvent:
		record.Status = "approved"
		record.Summary = stringDetail(event.Details, "body")
		record.Audits = append(record.Audits, AuditRecord{
			Action:  "collaboration.comment",
			Actor:   strings.TrimSpace(event.Actor),
			Outcome: stringDetail(event.Details, "decision"),
			Details: cloneMap(event.Details),
		})
	case CICompletedEvent:
		record.Status = "completed"
		record.Summary = fmt.Sprintf("CI workflow %s completed with %s", stringDetail(event.Details, "workflow"), stringDetail(event.Details, "conclusion"))
	case TaskFailedEvent:
		record.Status = "failed"
		record.Summary = stringDetail(event.Details, "error")
	default:
		return RunRecord{}, fmt.Errorf("unsupported event type %s", event.EventType)
	}

	record.Audits = append(record.Audits, AuditRecord{
		Action:  "event_bus.transition",
		Actor:   strings.TrimSpace(event.Actor),
		Outcome: record.Status,
		Details: map[string]any{
			"event_type":      event.EventType,
			"previous_status": previousStatus,
			"status":          record.Status,
		},
	})

	records[index] = record
	if err := b.ledger.save(records); err != nil {
		return RunRecord{}, err
	}
	for _, fn := range b.subscribers[strings.TrimSpace(event.EventType)] {
		fn(event, record)
	}
	return record, nil
}

func stringDetail(details map[string]any, key string) string {
	if details == nil {
		return ""
	}
	if value, ok := details[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func cloneMap(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}
	out := make(map[string]any, len(details))
	for key, value := range details {
		out[key] = value
	}
	return out
}
