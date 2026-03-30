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

type AuditEntry struct {
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
	RunID   string       `json:"run_id"`
	Status  string       `json:"status"`
	Summary string       `json:"summary"`
	Audits  []AuditEntry `json:"audits,omitempty"`
	Comments []Comment   `json:"comments,omitempty"`
}

type Ledger struct {
	Path string
}

func (l Ledger) LoadRuns() ([]Run, error) {
	if strings.TrimSpace(l.Path) == "" {
		return nil, nil
	}
	body, err := os.ReadFile(l.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var runs []Run
	if err := json.Unmarshal(body, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (l Ledger) Upsert(run Run) error {
	runs, err := l.LoadRuns()
	if err != nil {
		return err
	}
	replaced := false
	for index := range runs {
		if runs[index].RunID == run.RunID {
			runs[index] = run
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

type BusEvent struct {
	EventType string         `json:"event_type"`
	RunID     string         `json:"run_id"`
	Actor     string         `json:"actor"`
	Details   map[string]any `json:"details,omitempty"`
	Timestamp string         `json:"timestamp,omitempty"`
}

type Subscriber func(BusEvent, Run)

type EventBus struct {
	ledger      Ledger
	runs        map[string]Run
	subscribers map[string][]Subscriber
	now         func() string
}

func New(ledger Ledger) *EventBus {
	return &EventBus{
		ledger:      ledger,
		runs:        make(map[string]Run),
		subscribers: make(map[string][]Subscriber),
		now: func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
	}
}

func (b *EventBus) RegisterRun(run Run) {
	b.runs[run.RunID] = run
}

func (b *EventBus) Subscribe(eventType string, handler Subscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *EventBus) Publish(event BusEvent) (Run, error) {
	if strings.TrimSpace(event.Timestamp) == "" {
		event.Timestamp = b.now()
	}
	run, err := b.resolveRun(event.RunID)
	if err != nil {
		return Run{}, err
	}
	previousStatus := run.Status
	run = recordEvent(run, event)

	nextStatus, summary := resolveTransition(run, event)
	if strings.TrimSpace(nextStatus) != "" {
		run.Status = nextStatus
		run.Summary = summary
		run.Audits = append(run.Audits, AuditEntry{
			Action:  "event_bus.transition",
			Actor:   "event-bus",
			Outcome: nextStatus,
			Details: map[string]any{
				"event_type":       event.EventType,
				"previous_status":  previousStatus,
				"status":           nextStatus,
				"summary":          summary,
				"event_timestamp":  event.Timestamp,
			},
		})
	}

	b.runs[run.RunID] = run
	if err := b.ledger.Upsert(run); err != nil {
		return Run{}, err
	}
	for _, handler := range b.subscribers[event.EventType] {
		handler(event, run)
	}
	return run, nil
}

func (b *EventBus) resolveRun(runID string) (Run, error) {
	if run, ok := b.runs[runID]; ok {
		return run, nil
	}
	runs, err := b.ledger.LoadRuns()
	if err != nil {
		return Run{}, err
	}
	for _, run := range runs {
		if run.RunID == runID {
			b.runs[runID] = run
			return run, nil
		}
	}
	return Run{}, fmt.Errorf("run %q is not registered with the event bus", runID)
}

func recordEvent(run Run, event BusEvent) Run {
	run.Audits = append(run.Audits, AuditEntry{
		Action:  "event_bus.event",
		Actor:   event.Actor,
		Outcome: "received",
		Details: mergeDetails(map[string]any{
			"event_type":      event.EventType,
			"event_timestamp": event.Timestamp,
		}, event.Details),
	})
	if event.EventType != PullRequestCommentEvent {
		return run
	}
	body := strings.TrimSpace(toString(event.Details["body"]))
	if body == "" {
		return run
	}
	mentions := stringSlice(event.Details["mentions"])
	run.Comments = append(run.Comments, Comment{
		Author:   event.Actor,
		Body:     body,
		Mentions: mentions,
		Anchor:   "pull-request",
		Surface:  "pull-request",
	})
	run.Audits = append(run.Audits, AuditEntry{
		Action:  "collaboration.comment",
		Actor:   event.Actor,
		Outcome: "recorded",
		Details: map[string]any{
			"body":     body,
			"mentions": mentions,
			"anchor":   "pull-request",
			"surface":  "pull-request",
		},
	})
	return run
}

func resolveTransition(run Run, event BusEvent) (string, string) {
	if explicit := strings.TrimSpace(toString(event.Details["target_status"])); explicit != "" {
		return explicit, buildSummary(event, explicit)
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		decision := strings.ToLower(strings.TrimSpace(toString(event.Details["decision"])))
		if containsString([]string{"approved", "accept", "accepted", "lgtm"}, decision) {
			return "approved", buildSummary(event, "approved")
		}
		if containsString([]string{"blocked", "changes-requested", "rejected"}, decision) {
			return "needs-approval", buildSummary(event, "needs-approval")
		}
	case CICompletedEvent:
		conclusion := strings.ToLower(strings.TrimSpace(toString(event.Details["conclusion"])))
		if containsString([]string{"success", "passed", "green"}, conclusion) {
			return "completed", buildSummary(event, "completed")
		}
		if containsString([]string{"cancelled", "canceled", "error", "failed", "failure", "timed_out"}, conclusion) {
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

func mergeDetails(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringSlice(value any) []string {
	items, ok := value.([]any)
	if ok {
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, toString(item))
		}
		return out
	}
	if typed, ok := value.([]string); ok {
		return append([]string(nil), typed...)
	}
	return nil
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return fmt.Sprint(value)
}
