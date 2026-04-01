package events

import "strings"

const (
	PullRequestCommentEvent = "pull_request.comment"
	CICompletedEvent        = "ci.completed"
	TaskFailedEvent         = "task.failed"
)

type TransitionEvent struct {
	EventType string         `json:"event_type"`
	RunID     string         `json:"run_id"`
	Actor     string         `json:"actor"`
	Details   map[string]any `json:"details,omitempty"`
	Timestamp string         `json:"timestamp,omitempty"`
}

type TransitionComment struct {
	Author   string   `json:"author"`
	Body     string   `json:"body"`
	Mentions []string `json:"mentions,omitempty"`
	Anchor   string   `json:"anchor,omitempty"`
	Surface  string   `json:"surface,omitempty"`
}

type TransitionAudit struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type TransitionRun struct {
	RunID    string              `json:"run_id"`
	Status   string              `json:"status"`
	Summary  string              `json:"summary"`
	Audits   []TransitionAudit   `json:"audits,omitempty"`
	Comments []TransitionComment `json:"comments,omitempty"`
}

type TransitionSubscriber func(TransitionEvent, *TransitionRun)

type TransitionBus struct {
	runs        map[string]*TransitionRun
	subscribers map[string][]TransitionSubscriber
}

func NewTransitionBus() *TransitionBus {
	return &TransitionBus{
		runs:        map[string]*TransitionRun{},
		subscribers: map[string][]TransitionSubscriber{},
	}
}

func (b *TransitionBus) RegisterRun(run *TransitionRun) {
	if run != nil {
		b.runs[run.RunID] = run
	}
}

func (b *TransitionBus) Subscribe(eventType string, handler TransitionSubscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *TransitionBus) Publish(event TransitionEvent) (*TransitionRun, bool) {
	run, ok := b.runs[event.RunID]
	if !ok {
		return nil, false
	}
	if event.Details == nil {
		event.Details = map[string]any{}
	}
	previousStatus := run.Status
	b.recordEvent(run, event)

	if nextStatus, summary := b.resolveTransition(run, event); nextStatus != "" {
		run.Status = nextStatus
		run.Summary = summary
		run.Audits = append(run.Audits, TransitionAudit{
			Action:  "event_bus.transition",
			Actor:   "event-bus",
			Outcome: nextStatus,
			Details: map[string]any{
				"event_type":      event.EventType,
				"previous_status": previousStatus,
				"status":          nextStatus,
				"summary":         summary,
				"event_timestamp": event.Timestamp,
			},
		})
	}

	for _, handler := range b.subscribers[event.EventType] {
		handler(event, run)
	}
	return run, true
}

func (b *TransitionBus) recordEvent(run *TransitionRun, event TransitionEvent) {
	details := cloneTransitionMap(event.Details)
	details["event_type"] = event.EventType
	details["event_timestamp"] = event.Timestamp
	run.Audits = append(run.Audits, TransitionAudit{
		Action:  "event_bus.event",
		Actor:   event.Actor,
		Outcome: "received",
		Details: details,
	})

	if event.EventType != PullRequestCommentEvent {
		return
	}
	body := strings.TrimSpace(stringValue(event.Details["body"]))
	if body == "" {
		return
	}
	run.Comments = append(run.Comments, TransitionComment{
		Author:   event.Actor,
		Body:     body,
		Mentions: stringSliceValue(event.Details["mentions"]),
		Anchor:   "pull-request",
		Surface:  "pull-request",
	})
}

func (b *TransitionBus) resolveTransition(run *TransitionRun, event TransitionEvent) (string, string) {
	if explicitStatus := strings.TrimSpace(stringValue(event.Details["target_status"])); explicitStatus != "" {
		return explicitStatus, buildTransitionSummary(event, explicitStatus)
	}

	switch event.EventType {
	case PullRequestCommentEvent:
		decision := strings.ToLower(strings.TrimSpace(stringValue(event.Details["decision"])))
		switch decision {
		case "approved", "accept", "accepted", "lgtm":
			return "approved", buildTransitionSummary(event, "approved")
		case "blocked", "changes-requested", "rejected":
			return "needs-approval", buildTransitionSummary(event, "needs-approval")
		}
	case CICompletedEvent:
		conclusion := strings.ToLower(strings.TrimSpace(stringValue(event.Details["conclusion"])))
		switch conclusion {
		case "success", "passed", "green":
			return "completed", buildTransitionSummary(event, "completed")
		case "cancelled", "canceled", "error", "failed", "failure", "timed_out":
			return "failed", buildTransitionSummary(event, "failed")
		}
	case TaskFailedEvent:
		return "failed", buildTransitionSummary(event, "failed")
	}

	return "", run.Summary
}

func buildTransitionSummary(event TransitionEvent, status string) string {
	if summary := strings.TrimSpace(stringValue(event.Details["summary"])); summary != "" {
		return summary
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		if body := strings.TrimSpace(stringValue(event.Details["body"])); body != "" {
			return body
		}
		return "pull request comment set run to " + status
	case CICompletedEvent:
		workflow := strings.TrimSpace(stringValue(event.Details["workflow"]))
		conclusion := strings.TrimSpace(stringValue(event.Details["conclusion"]))
		if conclusion == "" {
			conclusion = status
		}
		if workflow != "" {
			return "CI workflow " + workflow + " completed with " + conclusion
		}
		return "CI completed with " + conclusion
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

func cloneTransitionMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func stringSliceValue(value any) []string {
	items, ok := value.([]string)
	if ok {
		return append([]string(nil), items...)
	}
	rawItems, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
