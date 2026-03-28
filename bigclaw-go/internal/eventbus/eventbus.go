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

func NewTaskRun(task domain.Task, runID, medium string) *TaskRun {
	return &TaskRun{
		RunID:  strings.TrimSpace(runID),
		TaskID: strings.TrimSpace(task.ID),
		Source: strings.TrimSpace(task.Source),
		Title:  strings.TrimSpace(task.Title),
		Medium: strings.TrimSpace(medium),
		Status: "running",
	}
}

func (r *TaskRun) Finalize(status, summary string) {
	r.Status = strings.TrimSpace(status)
	r.Summary = strings.TrimSpace(summary)
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{
		Action:  strings.TrimSpace(action),
		Actor:   strings.TrimSpace(actor),
		Outcome: strings.TrimSpace(outcome),
		Details: cloneMap(details),
	})
}

type Ledger struct {
	path string
}

func NewLedger(path string) *Ledger {
	return &Ledger{path: path}
}

func (l *Ledger) Load() ([]map[string]any, error) {
	if _, err := os.Stat(l.path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	body, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}
	var entries []map[string]any
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (l *Ledger) LoadRuns() ([]*TaskRun, error) {
	entries, err := l.Load()
	if err != nil {
		return nil, err
	}
	runs := make([]*TaskRun, 0, len(entries))
	for _, entry := range entries {
		body, err := json.Marshal(entry)
		if err != nil {
			return nil, err
		}
		var run TaskRun
		if err := json.Unmarshal(body, &run); err != nil {
			return nil, err
		}
		runs = append(runs, &run)
	}
	return runs, nil
}

func (l *Ledger) Upsert(run *TaskRun) error {
	entries, err := l.Load()
	if err != nil {
		return err
	}
	serialized := map[string]any{
		"run_id":  run.RunID,
		"task_id": run.TaskID,
		"source":  run.Source,
		"title":   run.Title,
		"medium":  run.Medium,
		"status":  run.Status,
		"summary": run.Summary,
		"audits":  run.Audits,
	}
	for i, entry := range entries {
		if fmt.Sprint(entry["run_id"]) == run.RunID {
			entries[i] = serialized
			return l.write(entries)
		}
	}
	entries = append(entries, serialized)
	return l.write(entries)
}

func (l *Ledger) write(entries []map[string]any) error {
	body, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
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

type Subscriber func(BusEvent, *TaskRun)

type EventBus struct {
	ledger      *Ledger
	runs        map[string]*TaskRun
	subscribers map[string][]Subscriber
}

func NewEventBus(ledger *Ledger) *EventBus {
	return &EventBus{
		ledger:      ledger,
		runs:        make(map[string]*TaskRun),
		subscribers: make(map[string][]Subscriber),
	}
}

func (b *EventBus) RegisterRun(run *TaskRun) {
	b.runs[run.RunID] = run
}

func (b *EventBus) Subscribe(eventType string, handler Subscriber) {
	b.subscribers[strings.TrimSpace(eventType)] = append(b.subscribers[strings.TrimSpace(eventType)], handler)
}

func (b *EventBus) Publish(event BusEvent) (*TaskRun, error) {
	run, err := b.resolveRun(event.RunID)
	if err != nil {
		return nil, err
	}

	previousStatus := run.Status
	run.Audit("event_bus.event", event.Actor, "received", withBaseEventDetails(event))
	if event.EventType == PullRequestCommentEvent {
		body := strings.TrimSpace(fmt.Sprint(event.Details["body"]))
		if body != "" {
			run.Audit("collaboration.comment", event.Actor, "recorded", map[string]any{
				"surface":  "pull-request",
				"body":     body,
				"mentions": stringSlice(event.Details["mentions"]),
				"anchor":   "pull-request",
				"status":   "open",
			})
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
			"event_timestamp": event.Timestamp,
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
		for _, run := range runs {
			if run.RunID == runID {
				b.runs[runID] = run
				return run, nil
			}
		}
	}
	return nil, fmt.Errorf("run %q is not registered with the event bus", runID)
}

func resolveTransition(run *TaskRun, event BusEvent) (string, string) {
	explicitStatus := detailString(event.Details, "target_status")
	if explicitStatus != "" {
		return explicitStatus, buildSummary(event, explicitStatus)
	}

	switch event.EventType {
	case PullRequestCommentEvent:
		decision := strings.ToLower(detailString(event.Details, "decision"))
		switch decision {
		case "approved", "accept", "accepted", "lgtm":
			return "approved", buildSummary(event, "approved")
		case "blocked", "changes-requested", "rejected":
			return "needs-approval", buildSummary(event, "needs-approval")
		}
	case CICompletedEvent:
		conclusion := strings.ToLower(detailString(event.Details, "conclusion"))
		switch conclusion {
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
	if summary := detailString(event.Details, "summary"); summary != "" {
		return summary
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		if body := detailString(event.Details, "body"); body != "" {
			return body
		}
		return fmt.Sprintf("pull request comment set run to %s", status)
	case CICompletedEvent:
		workflow := detailString(event.Details, "workflow")
		conclusion := detailString(event.Details, "conclusion")
		if conclusion == "" {
			conclusion = status
		}
		if workflow != "" {
			return fmt.Sprintf("CI workflow %s completed with %s", workflow, conclusion)
		}
		return fmt.Sprintf("CI completed with %s", conclusion)
	case TaskFailedEvent:
		if reason := detailString(event.Details, "error"); reason != "" {
			return reason
		}
		if reason := detailString(event.Details, "reason"); reason != "" {
			return reason
		}
		return "task failed"
	default:
		return status
	}
}

func withBaseEventDetails(event BusEvent) map[string]any {
	details := cloneMap(event.Details)
	details["event_type"] = event.EventType
	details["event_timestamp"] = event.Timestamp
	return details
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func detailString(details map[string]any, key string) string {
	if len(details) == 0 {
		return ""
	}
	value, ok := details[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func stringSlice(value any) []string {
	items, ok := value.([]string)
	if ok {
		return append([]string(nil), items...)
	}
	generic, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(generic))
	for _, item := range generic {
		out = append(out, strings.TrimSpace(fmt.Sprint(item)))
	}
	return out
}
