package runbus

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

type BusEvent struct {
	EventType string         `json:"event_type"`
	RunID     string         `json:"run_id"`
	Actor     string         `json:"actor"`
	Details   map[string]any `json:"details,omitempty"`
	Timestamp string         `json:"timestamp"`
}

type Audit struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type Comment struct {
	Author   string   `json:"author"`
	Body     string   `json:"body"`
	Mentions []string `json:"mentions,omitempty"`
	Anchor   string   `json:"anchor,omitempty"`
	Surface  string   `json:"surface,omitempty"`
}

type Run struct {
	RunID    string    `json:"run_id"`
	TaskID   string    `json:"task_id,omitempty"`
	Status   string    `json:"status"`
	Summary  string    `json:"summary"`
	Audits   []Audit   `json:"audits,omitempty"`
	Comments []Comment `json:"comments,omitempty"`
}

func (r *Run) Finalize(status, summary string) {
	r.Status = strings.TrimSpace(status)
	r.Summary = strings.TrimSpace(summary)
}

func (r *Run) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, Audit{
		Action:  action,
		Actor:   actor,
		Outcome: outcome,
		Details: cloneMap(details),
	})
}

func (r *Run) AddComment(author, body string, mentions []string) {
	r.Comments = append(r.Comments, Comment{
		Author:   author,
		Body:     body,
		Mentions: append([]string(nil), mentions...),
		Anchor:   "pull-request",
		Surface:  "pull-request",
	})
	r.Audit("collaboration.comment", author, "recorded", map[string]any{
		"body":     body,
		"mentions": append([]string(nil), mentions...),
	})
}

type Ledger struct {
	Path string
}

func (l Ledger) Load() ([]Run, error) {
	if _, err := os.Stat(l.Path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	body, err := os.ReadFile(l.Path)
	if err != nil {
		return nil, err
	}
	var runs []Run
	if err := json.Unmarshal(body, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (l Ledger) Upsert(run Run) error {
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
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.Path, body, 0o644)
}

type Subscriber func(BusEvent, *Run)

type Bus struct {
	ledger      *Ledger
	runs        map[string]*Run
	subscribers map[string][]Subscriber
}

func NewBus(ledger *Ledger) *Bus {
	return &Bus{
		ledger:      ledger,
		runs:        map[string]*Run{},
		subscribers: map[string][]Subscriber{},
	}
}

func (b *Bus) RegisterRun(run Run) {
	copy := run
	b.runs[run.RunID] = &copy
}

func (b *Bus) Subscribe(eventType string, handler Subscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *Bus) Publish(event BusEvent) (*Run, error) {
	run, err := b.resolveRun(event.RunID)
	if err != nil {
		return nil, err
	}

	previousStatus := run.Status
	run.Audit("event_bus.event", event.Actor, "received", mergeMaps(map[string]any{
		"event_type":      event.EventType,
		"event_timestamp": event.Timestamp,
	}, event.Details))

	if event.EventType == PullRequestCommentEvent {
		if body := strings.TrimSpace(toString(event.Details["body"])); body != "" {
			run.AddComment(event.Actor, body, toStringSlice(event.Details["mentions"]))
		}
	}

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

	for _, subscriber := range b.subscribers[event.EventType] {
		subscriber(event, run)
	}

	if b.ledger != nil {
		if err := b.ledger.Upsert(*run); err != nil {
			return nil, err
		}
	}
	return run, nil
}

func (b *Bus) resolveRun(runID string) (*Run, error) {
	if run, ok := b.runs[runID]; ok {
		return run, nil
	}
	if b.ledger != nil {
		runs, err := b.ledger.Load()
		if err != nil {
			return nil, err
		}
		for _, run := range runs {
			if run.RunID == runID {
				copy := run
				b.runs[runID] = &copy
				return &copy, nil
			}
		}
	}
	return nil, fmt.Errorf("run %q is not registered with the event bus", runID)
}

func resolveTransition(run Run, event BusEvent) (string, string) {
	if explicit := strings.TrimSpace(toString(event.Details["target_status"])); explicit != "" {
		return explicit, buildSummary(event, explicit)
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		switch strings.ToLower(strings.TrimSpace(toString(event.Details["decision"]))) {
		case "approved", "accept", "accepted", "lgtm":
			return "approved", buildSummary(event, "approved")
		case "blocked", "changes-requested", "rejected":
			return "needs-approval", buildSummary(event, "needs-approval")
		}
	case CICompletedEvent:
		switch strings.ToLower(strings.TrimSpace(toString(event.Details["conclusion"]))) {
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
	if summary := strings.TrimSpace(toString(event.Details["summary"])); summary != "" {
		return summary
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		if body := strings.TrimSpace(toString(event.Details["body"])); body != "" {
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
			return "CI workflow " + workflow + " completed with " + conclusion
		}
		return "CI completed with " + conclusion
	case TaskFailedEvent:
		if reason := strings.TrimSpace(toString(event.Details["error"])); reason != "" {
			return reason
		}
		if reason := strings.TrimSpace(toString(event.Details["reason"])); reason != "" {
			return reason
		}
		return "task failed"
	default:
		return status
	}
}

func mergeMaps(left, right map[string]any) map[string]any {
	merged := cloneMap(left)
	for key, value := range right {
		merged[key] = value
	}
	return merged
}

func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func toStringSlice(value any) []string {
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
		if text := strings.TrimSpace(toString(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func NowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
