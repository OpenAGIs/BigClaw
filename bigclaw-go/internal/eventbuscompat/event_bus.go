package eventbuscompat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/domain"
)

const (
	PullRequestCommentEvent = "pull_request.comment"
	CICompletedEvent        = "ci.completed"
	TaskFailedEvent         = "task.failed"
)

type AuditEntry struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type TaskRun struct {
	RunID   string       `json:"run_id"`
	TaskID  string       `json:"task_id"`
	Source  string       `json:"source"`
	Title   string       `json:"title"`
	Medium  string       `json:"medium"`
	Status  string       `json:"status"`
	Summary string       `json:"summary"`
	Audits  []AuditEntry `json:"audits,omitempty"`
}

func NewRun(task domain.Task, runID, medium string) *TaskRun {
	return &TaskRun{
		RunID:  runID,
		TaskID: task.ID,
		Source: task.Source,
		Title:  task.Title,
		Medium: medium,
		Status: "running",
	}
}

func (r *TaskRun) Finalize(status, summary string) {
	r.Status = status
	r.Summary = summary
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{
		Action:  action,
		Actor:   actor,
		Outcome: outcome,
		Details: details,
	})
}

func (r *TaskRun) AddComment(author, body string, mentions []string) {
	r.Audit("collaboration.comment", author, "recorded", map[string]any{
		"surface":    "pull-request",
		"body":       body,
		"mentions":   append([]string(nil), mentions...),
		"comment_id": fmt.Sprintf("%s-comment-%d", r.RunID, countComments(r.Audits)+1),
	})
}

func countComments(audits []AuditEntry) int {
	count := 0
	for _, audit := range audits {
		if audit.Action == "collaboration.comment" {
			count++
		}
	}
	return count
}

type Ledger struct {
	storagePath string
}

func NewLedger(path string) *Ledger {
	return &Ledger{storagePath: path}
}

func (l *Ledger) Load() ([]map[string]any, error) {
	if _, err := os.Stat(l.storagePath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	body, err := os.ReadFile(l.storagePath)
	if err != nil {
		return nil, err
	}
	var entries []map[string]any
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (l *Ledger) LoadRuns() ([]TaskRun, error) {
	entries, err := l.Load()
	if err != nil {
		return nil, err
	}
	runs := make([]TaskRun, 0, len(entries))
	for _, entry := range entries {
		body, err := json.Marshal(entry)
		if err != nil {
			return nil, err
		}
		var run TaskRun
		if err := json.Unmarshal(body, &run); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (l *Ledger) Upsert(run *TaskRun) error {
	entries, err := l.Load()
	if err != nil {
		return err
	}
	body, err := json.Marshal(run)
	if err != nil {
		return err
	}
	var serialized map[string]any
	if err := json.Unmarshal(body, &serialized); err != nil {
		return err
	}
	for i, entry := range entries {
		if entry["run_id"] == run.RunID {
			entries[i] = serialized
			return l.writeEntries(entries)
		}
	}
	entries = append(entries, serialized)
	return l.writeEntries(entries)
}

func (l *Ledger) writeEntries(entries []map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(l.storagePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.storagePath, body, 0o644)
}

type BusEvent struct {
	EventType string
	RunID     string
	Actor     string
	Details   map[string]any
}

type EventSubscriber func(BusEvent, *TaskRun)

type EventBus struct {
	ledger      *Ledger
	runs        map[string]*TaskRun
	subscribers map[string][]EventSubscriber
}

func NewEventBus(ledger *Ledger) *EventBus {
	return &EventBus{
		ledger:      ledger,
		runs:        make(map[string]*TaskRun),
		subscribers: make(map[string][]EventSubscriber),
	}
}

func (b *EventBus) RegisterRun(run *TaskRun) {
	b.runs[run.RunID] = run
}

func (b *EventBus) Subscribe(eventType string, handler EventSubscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *EventBus) Publish(event BusEvent) (*TaskRun, error) {
	run, err := b.resolveRun(event.RunID)
	if err != nil {
		return nil, err
	}
	previousStatus := run.Status
	run.Audit("event_bus.event", event.Actor, "received", map[string]any{
		"event_type": event.EventType,
	})
	if event.EventType == PullRequestCommentEvent {
		body := strings.TrimSpace(asString(event.Details["body"]))
		if body != "" {
			run.AddComment(event.Actor, body, stringSlice(event.Details["mentions"]))
		}
	}
	nextStatus, summary := resolveTransition(run, event)
	if nextStatus != "" {
		run.Finalize(nextStatus, summary)
		run.Audit("event_bus.transition", "event-bus", nextStatus, map[string]any{
			"event_type":      event.EventType,
			"previous_status": previousStatus,
			"status":          nextStatus,
			"summary":         summary,
		})
	}
	for _, handler := range b.subscribers[event.EventType] {
		handler(event, run)
	}
	if b.ledger != nil {
		if err := b.ledger.Upsert(run); err != nil {
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
		runs, err := b.ledger.LoadRuns()
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

func resolveTransition(run *TaskRun, event BusEvent) (string, string) {
	if explicit := strings.TrimSpace(asString(event.Details["target_status"])); explicit != "" {
		return explicit, buildSummary(event, explicit)
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		decision := strings.ToLower(strings.TrimSpace(asString(event.Details["decision"])))
		if decision == "approved" || decision == "accept" || decision == "accepted" || decision == "lgtm" {
			return "approved", buildSummary(event, "approved")
		}
		if decision == "blocked" || decision == "changes-requested" || decision == "rejected" {
			return "needs-approval", buildSummary(event, "needs-approval")
		}
	case CICompletedEvent:
		conclusion := strings.ToLower(strings.TrimSpace(asString(event.Details["conclusion"])))
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
	if summary := strings.TrimSpace(asString(event.Details["summary"])); summary != "" {
		return summary
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		if body := strings.TrimSpace(asString(event.Details["body"])); body != "" {
			return body
		}
		return "pull request comment set run to " + status
	case CICompletedEvent:
		workflow := strings.TrimSpace(asString(event.Details["workflow"]))
		conclusion := strings.TrimSpace(asString(event.Details["conclusion"]))
		if conclusion == "" {
			conclusion = status
		}
		if workflow != "" {
			return fmt.Sprintf("CI workflow %s completed with %s", workflow, conclusion)
		}
		return "CI completed with " + conclusion
	case TaskFailedEvent:
		if reason := strings.TrimSpace(asString(event.Details["error"])); reason != "" {
			return reason
		}
		if reason := strings.TrimSpace(asString(event.Details["reason"])); reason != "" {
			return reason
		}
		return "task failed"
	default:
		return status
	}
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func stringSlice(value any) []string {
	if value == nil {
		return nil
	}
	items, ok := value.([]string)
	if ok {
		return append([]string(nil), items...)
	}
	interfaces, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(interfaces))
	for _, item := range interfaces {
		out = append(out, asString(item))
	}
	return out
}
