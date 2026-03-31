package eventbuscompat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	PullRequestCommentEvent = "pull_request.comment"
	CICompletedEvent        = "ci.completed"
	TaskFailedEvent         = "task.failed"
)

type AuditRecord struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type RunRecord struct {
	RunID   string        `json:"run_id"`
	Status  string        `json:"status"`
	Summary string        `json:"summary"`
	Audits  []AuditRecord `json:"audits,omitempty"`
}

func (r *RunRecord) Finalize(status, summary string) {
	r.Status = strings.TrimSpace(status)
	r.Summary = strings.TrimSpace(summary)
}

func (r *RunRecord) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditRecord{
		Action:  action,
		Actor:   actor,
		Outcome: outcome,
		Details: cloneMap(details),
	})
}

func (r *RunRecord) AddComment(actor, body string, mentions []string) {
	r.Audit("collaboration.comment", actor, "recorded", map[string]any{
		"body":     body,
		"mentions": append([]string(nil), mentions...),
	})
}

type Ledger struct {
	path string
}

func NewLedger(path string) *Ledger {
	return &Ledger{path: path}
}

func (l *Ledger) Append(run RunRecord) error {
	runs, err := l.Load()
	if err != nil {
		return err
	}
	runs = append(runs, run)
	return l.write(runs)
}

func (l *Ledger) Upsert(run RunRecord) error {
	runs, err := l.Load()
	if err != nil {
		return err
	}
	replaced := false
	for i := range runs {
		if runs[i].RunID == run.RunID {
			runs[i] = run
			replaced = true
			break
		}
	}
	if !replaced {
		runs = append(runs, run)
	}
	return l.write(runs)
}

func (l *Ledger) Load() ([]RunRecord, error) {
	if strings.TrimSpace(l.path) == "" {
		return nil, nil
	}
	body, err := os.ReadFile(l.path)
	if os.IsNotExist(err) {
		return []RunRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	var runs []RunRecord
	if err := json.Unmarshal(body, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (l *Ledger) write(runs []RunRecord) error {
	if strings.TrimSpace(l.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, body, 0o644)
}

type BusEvent struct {
	EventType string         `json:"event_type"`
	RunID     string         `json:"run_id"`
	Actor     string         `json:"actor"`
	Details   map[string]any `json:"details,omitempty"`
	Timestamp string         `json:"timestamp"`
}

func NewBusEvent(eventType, runID, actor string, details map[string]any) BusEvent {
	return BusEvent{
		EventType: strings.TrimSpace(eventType),
		RunID:     strings.TrimSpace(runID),
		Actor:     strings.TrimSpace(actor),
		Details:   cloneMap(details),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

type Subscriber func(BusEvent, *RunRecord)

type EventBus struct {
	ledger      *Ledger
	runs        map[string]*RunRecord
	subscribers map[string][]Subscriber
}

func NewEventBus(ledger *Ledger) *EventBus {
	return &EventBus{
		ledger:      ledger,
		runs:        make(map[string]*RunRecord),
		subscribers: make(map[string][]Subscriber),
	}
}

func (b *EventBus) RegisterRun(run *RunRecord) {
	if run != nil {
		b.runs[run.RunID] = run
	}
}

func (b *EventBus) Subscribe(eventType string, handler Subscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *EventBus) Publish(event BusEvent) (*RunRecord, error) {
	run, err := b.resolveRun(event.RunID)
	if err != nil {
		return nil, err
	}
	previousStatus := run.Status
	b.recordEvent(run, event)

	nextStatus, summary := resolveTransition(*run, event)
	if nextStatus != "" {
		run.Finalize(nextStatus, summary)
		run.Audit("event_bus.transition", "event-bus", nextStatus, map[string]any{
			"event_type":      event.EventType,
			"previous_status": previousStatus,
			"status":          nextStatus,
			"summary":         summary,
			"event_timestamp": event.Timestamp,
		})
	}
	for _, handler := range b.subscribers[event.EventType] {
		handler(event, run)
	}
	if b.ledger != nil {
		if err := b.ledger.Upsert(*run); err != nil {
			return nil, err
		}
	}
	return run, nil
}

func (b *EventBus) resolveRun(runID string) (*RunRecord, error) {
	if run := b.runs[runID]; run != nil {
		return run, nil
	}
	if b.ledger != nil {
		runs, err := b.ledger.Load()
		if err != nil {
			return nil, err
		}
		for i := range runs {
			if runs[i].RunID == runID {
				run := runs[i]
				b.runs[runID] = &run
				return &run, nil
			}
		}
	}
	return nil, fmt.Errorf("run %q is not registered with the event bus", runID)
}

func (b *EventBus) recordEvent(run *RunRecord, event BusEvent) {
	details := cloneMap(event.Details)
	details["event_type"] = event.EventType
	details["event_timestamp"] = event.Timestamp
	run.Audit("event_bus.event", event.Actor, "received", details)
	if event.EventType != PullRequestCommentEvent {
		return
	}
	body := strings.TrimSpace(toString(event.Details["body"]))
	if body == "" {
		return
	}
	mentions := stringsToSlice(event.Details["mentions"])
	run.AddComment(event.Actor, body, mentions)
}

func resolveTransition(run RunRecord, event BusEvent) (string, string) {
	explicitStatus := strings.TrimSpace(toString(event.Details["target_status"]))
	if explicitStatus != "" {
		return explicitStatus, buildSummary(event, explicitStatus)
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		decision := strings.ToLower(strings.TrimSpace(toString(event.Details["decision"])))
		if decision == "approved" || decision == "accept" || decision == "accepted" || decision == "lgtm" {
			return "approved", buildSummary(event, "approved")
		}
		if decision == "blocked" || decision == "changes-requested" || decision == "rejected" {
			return "needs-approval", buildSummary(event, "needs-approval")
		}
	case CICompletedEvent:
		conclusion := strings.ToLower(strings.TrimSpace(toString(event.Details["conclusion"])))
		if conclusion == "success" || conclusion == "passed" || conclusion == "green" {
			return "completed", buildSummary(event, "completed")
		}
		if conclusion == "cancelled" || conclusion == "canceled" || conclusion == "error" || conclusion == "failed" || conclusion == "failure" || conclusion == "timed_out" {
			return "failed", buildSummary(event, "failed")
		}
	case TaskFailedEvent:
		return "failed", buildSummary(event, "failed")
	}
	return "", run.Summary
}

func buildSummary(event BusEvent, status string) string {
	summary := strings.TrimSpace(toString(event.Details["summary"]))
	if summary != "" {
		return summary
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		body := strings.TrimSpace(toString(event.Details["body"]))
		if body != "" {
			return body
		}
		return "pull request comment set run to " + status
	case CICompletedEvent:
		workflow := strings.TrimSpace(toString(event.Details["workflow"]))
		conclusion := strings.TrimSpace(toString(event.Details["conclusion"]))
		if conclusion == "" {
			conclusion = status
		}
		if workflow != "" {
			return fmt.Sprintf("CI workflow %s completed with %s", workflow, conclusion)
		}
		return fmt.Sprintf("CI completed with %s", conclusion)
	case TaskFailedEvent:
		reason := strings.TrimSpace(toString(event.Details["error"]))
		if reason == "" {
			reason = strings.TrimSpace(toString(event.Details["reason"]))
		}
		if reason != "" {
			return reason
		}
		return "task failed"
	default:
		return status
	}
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func stringsToSlice(value any) []string {
	items, ok := value.([]string)
	if ok {
		return append([]string(nil), items...)
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text := strings.TrimSpace(toString(item))
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}
