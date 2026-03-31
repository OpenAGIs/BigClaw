package eventbus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

const (
	PullRequestCommentEvent = "pull_request.comment"
	CICompletedEvent        = "ci.completed"
	TaskFailedEvent         = "task.failed"
)

type AuditEntry struct {
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Outcome   string         `json:"outcome"`
	Timestamp string         `json:"timestamp"`
	Details   map[string]any `json:"details"`
}

type TaskRun struct {
	RunID   string       `json:"run_id"`
	TaskID  string       `json:"task_id"`
	Source  string       `json:"source"`
	Title   string       `json:"title"`
	Medium  string       `json:"medium"`
	Status  string       `json:"status"`
	Summary string       `json:"summary"`
	Audits  []AuditEntry `json:"audits"`
}

func NewTaskRun(task domain.Task, runID, medium string) TaskRun {
	return TaskRun{
		RunID:   runID,
		TaskID:  task.ID,
		Source:  task.Source,
		Title:   task.Title,
		Medium:  medium,
		Status:  "running",
		Summary: "",
		Audits:  nil,
	}
}

func (r *TaskRun) Finalize(status, summary string) {
	r.Status = status
	r.Summary = summary
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{
		Action:    action,
		Actor:     actor,
		Outcome:   outcome,
		Timestamp: utcNow(),
		Details:   cloneMap(details),
	})
}

func (r *TaskRun) AddComment(author, body string, mentions []string, anchor, surface string) {
	commentCount := 0
	for _, audit := range r.Audits {
		if audit.Action == "collaboration.comment" {
			commentCount++
		}
	}
	r.Audit("collaboration.comment", author, "recorded", map[string]any{
		"surface":    surface,
		"comment_id": fmt.Sprintf("%s-comment-%d", r.RunID, commentCount+1),
		"body":       body,
		"mentions":   append([]string(nil), mentions...),
		"anchor":     anchor,
		"status":     "open",
	})
}

type Ledger struct {
	storagePath string
}

func NewLedger(storagePath string) Ledger {
	return Ledger{storagePath: storagePath}
}

func (l Ledger) Load() ([]map[string]any, error) {
	data, err := os.ReadFile(l.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (l Ledger) loadRuns() ([]TaskRun, error) {
	entries, err := l.Load()
	if err != nil {
		return nil, err
	}
	out := make([]TaskRun, 0, len(entries))
	for _, entry := range entries {
		payload, err := json.Marshal(entry)
		if err != nil {
			return nil, err
		}
		var run TaskRun
		if err := json.Unmarshal(payload, &run); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (l Ledger) Append(run TaskRun) error {
	return l.Upsert(run)
}

func (l Ledger) Upsert(run TaskRun) error {
	entries, err := l.loadRuns()
	if err != nil {
		return err
	}
	replaced := false
	for i := range entries {
		if entries[i].RunID == run.RunID {
			entries[i] = run
			replaced = true
			break
		}
	}
	if !replaced {
		entries = append(entries, run)
	}
	return l.write(entries)
}

func (l Ledger) write(entries []TaskRun) error {
	if err := os.MkdirAll(filepath.Dir(l.storagePath), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.storagePath, payload, 0o644)
}

type BusEvent struct {
	EventType string         `json:"event_type"`
	RunID     string         `json:"run_id"`
	Actor     string         `json:"actor"`
	Details   map[string]any `json:"details"`
	Timestamp string         `json:"timestamp"`
}

func NewBusEvent(eventType, runID, actor string, details map[string]any) BusEvent {
	return BusEvent{
		EventType: eventType,
		RunID:     runID,
		Actor:     actor,
		Details:   cloneMap(details),
		Timestamp: utcNow(),
	}
}

type Subscriber func(BusEvent, *TaskRun)

type EventBus struct {
	ledger      *Ledger
	runs        map[string]*TaskRun
	subscribers map[string][]Subscriber
}

func NewEventBus(ledger *Ledger) *EventBus {
	return &EventBus{
		ledger:      ledger,
		runs:        map[string]*TaskRun{},
		subscribers: map[string][]Subscriber{},
	}
}

func (b *EventBus) RegisterRun(run *TaskRun) {
	b.runs[run.RunID] = run
}

func (b *EventBus) Subscribe(eventType string, handler Subscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *EventBus) Publish(event BusEvent) (*TaskRun, error) {
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

func (b *EventBus) resolveRun(runID string) (*TaskRun, error) {
	if run, ok := b.runs[runID]; ok {
		return run, nil
	}
	if b.ledger != nil {
		runs, err := b.ledger.loadRuns()
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

func (b *EventBus) recordEvent(run *TaskRun, event BusEvent) {
	details := cloneMap(event.Details)
	details["event_type"] = event.EventType
	details["event_timestamp"] = event.Timestamp
	run.Audit("event_bus.event", event.Actor, "received", details)
	if event.EventType != PullRequestCommentEvent {
		return
	}
	body := strings.TrimSpace(stringValue(event.Details["body"]))
	if body == "" {
		return
	}
	run.AddComment(event.Actor, body, stringSlice(event.Details["mentions"]), "pull-request", "pull-request")
}

func resolveTransition(run TaskRun, event BusEvent) (string, string) {
	if explicit := strings.TrimSpace(stringValue(event.Details["target_status"])); explicit != "" {
		return explicit, buildSummary(event, explicit)
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		switch strings.ToLower(strings.TrimSpace(stringValue(event.Details["decision"]))) {
		case "approved", "accept", "accepted", "lgtm":
			return "approved", buildSummary(event, "approved")
		case "blocked", "changes-requested", "rejected":
			return "needs-approval", buildSummary(event, "needs-approval")
		}
	case CICompletedEvent:
		switch strings.ToLower(strings.TrimSpace(stringValue(event.Details["conclusion"]))) {
		case "success", "passed", "green":
			return "completed", buildSummary(event, "completed")
		case "cancelled", "canceled", "error", "failed", "failure", "timed_out":
			return "failed", buildSummary(event, "failed")
		}
	case TaskFailedEvent:
		return "failed", buildSummary(event, "failed")
	}
	return "", run.Summary
}

func buildSummary(event BusEvent, status string) string {
	if summary := strings.TrimSpace(stringValue(event.Details["summary"])); summary != "" {
		return summary
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		if body := strings.TrimSpace(stringValue(event.Details["body"])); body != "" {
			return body
		}
		return fmt.Sprintf("pull request comment set run to %s", status)
	case CICompletedEvent:
		workflow := strings.TrimSpace(stringValue(event.Details["workflow"]))
		conclusion := strings.TrimSpace(stringValue(event.Details["conclusion"]))
		if conclusion == "" {
			conclusion = status
		}
		if workflow != "" {
			return fmt.Sprintf("CI workflow %s completed with %s", workflow, conclusion)
		}
		return fmt.Sprintf("CI completed with %s", conclusion)
	case TaskFailedEvent:
		if reason := strings.TrimSpace(stringValue(event.Details["error"])); reason != "" {
			return reason
		}
		if reason := strings.TrimSpace(stringValue(event.Details["reason"])); reason != "" {
			return reason
		}
		return "task failed"
	default:
		return status
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
