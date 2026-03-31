package eventbus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/domain"
)

const (
	PullRequestCommentEvent = "pull_request_comment"
	CICompletedEvent        = "ci_completed"
	TaskFailedEvent         = "task_failed"
)

type BusEvent struct {
	EventType string         `json:"event_type"`
	RunID     string         `json:"run_id"`
	Actor     string         `json:"actor"`
	Details   map[string]any `json:"details,omitempty"`
}

type AuditRecord struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type TaskRun struct {
	Task    domain.Task   `json:"task"`
	RunID   string        `json:"run_id"`
	Medium  string        `json:"medium"`
	Status  string        `json:"status"`
	Summary string        `json:"summary"`
	Audits  []AuditRecord `json:"audits,omitempty"`
}

type ObservabilityLedger struct {
	Path string
}

type EventBus struct {
	ledger      *ObservabilityLedger
	subscribers map[string][]func(BusEvent, TaskRun)
}

func TaskRunFromTask(task domain.Task, runID, medium string) TaskRun {
	return TaskRun{
		Task:   task,
		RunID:  strings.TrimSpace(runID),
		Medium: strings.TrimSpace(medium),
		Status: "running",
	}
}

func (run *TaskRun) Finalize(status, summary string) {
	run.Status = strings.TrimSpace(status)
	run.Summary = strings.TrimSpace(summary)
}

func NewEventBus(ledger *ObservabilityLedger) *EventBus {
	return &EventBus{
		ledger:      ledger,
		subscribers: make(map[string][]func(BusEvent, TaskRun)),
	}
}

func (b *EventBus) Subscribe(eventType string, handler func(BusEvent, TaskRun)) {
	if b == nil || handler == nil {
		return
	}
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *EventBus) Publish(event BusEvent) (TaskRun, error) {
	if b == nil || b.ledger == nil {
		return TaskRun{}, fmt.Errorf("event bus requires ledger")
	}
	runs, err := b.ledger.Load()
	if err != nil {
		return TaskRun{}, err
	}
	index := -1
	for i, run := range runs {
		if run.RunID == event.RunID {
			index = i
			break
		}
	}
	if index == -1 {
		return TaskRun{}, fmt.Errorf("run %s not found", event.RunID)
	}
	current := runs[index]
	updated := applyEvent(current, event)
	runs[index] = updated
	if err := b.ledger.writeAll(runs); err != nil {
		return TaskRun{}, err
	}
	for _, handler := range b.subscribers[event.EventType] {
		handler(event, updated)
	}
	return updated, nil
}

func (l *ObservabilityLedger) Append(run TaskRun) error {
	runs, err := l.Load()
	if err != nil {
		return err
	}
	runs = append(runs, run)
	return l.writeAll(runs)
}

func (l *ObservabilityLedger) Load() ([]TaskRun, error) {
	if l == nil || strings.TrimSpace(l.Path) == "" {
		return nil, fmt.Errorf("ledger path required")
	}
	data, err := os.ReadFile(l.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return []TaskRun{}, nil
		}
		return nil, err
	}
	var runs []TaskRun
	if len(data) == 0 {
		return []TaskRun{}, nil
	}
	if err := json.Unmarshal(data, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (l *ObservabilityLedger) writeAll(runs []TaskRun) error {
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.Path, payload, 0o644)
}

func applyEvent(current TaskRun, event BusEvent) TaskRun {
	updated := current
	updated.Audits = append(updated.Audits, AuditRecord{
		Action: "event_bus.event",
		Actor:  event.Actor,
		Details: map[string]any{
			"event_type": event.EventType,
		},
	})

	previousStatus := updated.Status
	switch event.EventType {
	case PullRequestCommentEvent:
		updated.Status = "approved"
		updated.Summary = stringValue(event.Details["body"])
		updated.Audits = append(updated.Audits, AuditRecord{
			Action: "collaboration.comment",
			Actor:  event.Actor,
			Details: map[string]any{
				"decision": stringValue(event.Details["decision"]),
				"body":     updated.Summary,
				"mentions": sliceValue(event.Details["mentions"]),
			},
		})
	case CICompletedEvent:
		workflow := stringValue(event.Details["workflow"])
		conclusion := stringValue(event.Details["conclusion"])
		updated.Status = "completed"
		updated.Summary = fmt.Sprintf("CI workflow %s completed with %s", workflow, conclusion)
	case TaskFailedEvent:
		updated.Status = "failed"
		updated.Summary = stringValue(event.Details["error"])
	}

	if updated.Status != previousStatus {
		updated.Audits = append(updated.Audits, AuditRecord{
			Action: "event_bus.transition",
			Actor:  event.Actor,
			Details: map[string]any{
				"previous_status": previousStatus,
				"status":          updated.Status,
			},
		})
	}
	return updated
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func sliceValue(value any) []string {
	raw, ok := value.([]string)
	if ok {
		return append([]string(nil), raw...)
	}
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := stringValue(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}
