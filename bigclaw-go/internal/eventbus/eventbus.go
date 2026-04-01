package eventbus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	PullRequestCommentEvent = "pull_request_comment"
	CICompletedEvent        = "ci_completed"
	TaskFailedEvent         = "task_failed"
)

type Audit struct {
	Action  string         `json:"action"`
	Details map[string]any `json:"details,omitempty"`
}

type Run struct {
	RunID   string  `json:"run_id"`
	Status  string  `json:"status"`
	Summary string  `json:"summary"`
	Audits  []Audit `json:"audits,omitempty"`
}

type Ledger struct {
	path string
	runs []Run
}

type BusEvent struct {
	EventType string         `json:"event_type"`
	RunID     string         `json:"run_id"`
	Actor     string         `json:"actor"`
	Details   map[string]any `json:"details,omitempty"`
}

type Subscriber func(BusEvent, Run)

type EventBus struct {
	ledger      *Ledger
	subscribers map[string][]Subscriber
}

func NewLedger(path string) *Ledger {
	return &Ledger{path: path}
}

func (l *Ledger) Append(run Run) error {
	runs, err := l.Load()
	if err != nil {
		return err
	}
	runs = append(runs, run)
	l.runs = runs
	return l.save()
}

func (l *Ledger) Load() ([]Run, error) {
	if len(l.runs) > 0 {
		return append([]Run(nil), l.runs...), nil
	}
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
	if err := json.Unmarshal(body, &l.runs); err != nil {
		return nil, err
	}
	return append([]Run(nil), l.runs...), nil
}

func (l *Ledger) save() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(l.runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, body, 0o644)
}

func New(ledger *Ledger) *EventBus {
	return &EventBus{
		ledger:      ledger,
		subscribers: make(map[string][]Subscriber),
	}
}

func (b *EventBus) Subscribe(eventType string, subscriber Subscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], subscriber)
}

func (b *EventBus) Publish(event BusEvent) (Run, error) {
	runs, err := b.ledger.Load()
	if err != nil {
		return Run{}, err
	}
	for index := range runs {
		if runs[index].RunID != event.RunID {
			continue
		}
		previousStatus := runs[index].Status
		switch event.EventType {
		case PullRequestCommentEvent:
			runs[index].Status = "approved"
			runs[index].Summary, _ = event.Details["body"].(string)
			runs[index].Audits = append(runs[index].Audits,
				Audit{Action: "collaboration.comment", Details: map[string]any{"actor": event.Actor}},
				Audit{Action: "event_bus.transition", Details: map[string]any{"previous_status": previousStatus, "status": runs[index].Status}},
			)
		case CICompletedEvent:
			workflow, _ := event.Details["workflow"].(string)
			conclusion, _ := event.Details["conclusion"].(string)
			runs[index].Status = "completed"
			runs[index].Summary = fmt.Sprintf("CI workflow %s completed with %s", workflow, conclusion)
			runs[index].Audits = append(runs[index].Audits,
				Audit{Action: "event_bus.event", Details: map[string]any{"event_type": event.EventType}},
			)
		case TaskFailedEvent:
			errMsg, _ := event.Details["error"].(string)
			runs[index].Status = "failed"
			runs[index].Summary = errMsg
			runs[index].Audits = append(runs[index].Audits,
				Audit{Action: "event_bus.transition", Details: map[string]any{"previous_status": previousStatus, "status": runs[index].Status}},
			)
		}
		b.ledger.runs = runs
		if err := b.ledger.save(); err != nil {
			return Run{}, err
		}
		updated := runs[index]
		for _, subscriber := range b.subscribers[event.EventType] {
			subscriber(event, updated)
		}
		return updated, nil
	}
	return Run{}, fmt.Errorf("run %s not found", event.RunID)
}
