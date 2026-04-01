package runevents

import (
	"encoding/json"
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
	Action    string            `json:"action"`
	Actor     string            `json:"actor"`
	Outcome   string            `json:"outcome"`
	Timestamp string            `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

type RunRecord struct {
	RunID   string       `json:"run_id"`
	Status  string       `json:"status"`
	Summary string       `json:"summary"`
	Audits  []AuditEntry `json:"audits,omitempty"`
}

func (r *RunRecord) Finalize(status string, summary string) {
	r.Status = status
	r.Summary = summary
}

func (r *RunRecord) Audit(action string, actor string, outcome string, details map[string]string) {
	r.Audits = append(r.Audits, AuditEntry{
		Action:    action,
		Actor:     actor,
		Outcome:   outcome,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Details:   cloneMap(details),
	})
}

type BusEvent struct {
	EventType string            `json:"event_type"`
	RunID     string            `json:"run_id"`
	Actor     string            `json:"actor"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp string            `json:"timestamp"`
}

func NewBusEvent(eventType string, runID string, actor string, details map[string]string) BusEvent {
	return BusEvent{
		EventType: eventType,
		RunID:     runID,
		Actor:     actor,
		Details:   cloneMap(details),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

type Subscriber func(BusEvent, RunRecord)

type Ledger struct {
	storagePath string
}

func NewLedger(storagePath string) Ledger {
	return Ledger{storagePath: storagePath}
}

func (l Ledger) Upsert(run RunRecord) error {
	records, err := l.Load()
	if err != nil {
		return err
	}
	replaced := false
	for i := range records {
		if records[i].RunID == run.RunID {
			records[i] = run
			replaced = true
			break
		}
	}
	if !replaced {
		records = append(records, run)
	}
	if err := os.MkdirAll(filepath.Dir(l.storagePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.storagePath, body, 0o644)
}

func (l Ledger) Load() ([]RunRecord, error) {
	body, err := os.ReadFile(l.storagePath)
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

type Bus struct {
	ledger      *Ledger
	runs        map[string]RunRecord
	subscribers map[string][]Subscriber
}

func NewBus(ledger *Ledger) *Bus {
	return &Bus{
		ledger:      ledger,
		runs:        make(map[string]RunRecord),
		subscribers: make(map[string][]Subscriber),
	}
}

func (b *Bus) RegisterRun(run RunRecord) {
	b.runs[run.RunID] = run
}

func (b *Bus) Subscribe(eventType string, handler Subscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *Bus) Publish(event BusEvent) (RunRecord, error) {
	run, err := b.resolveRun(event.RunID)
	if err != nil {
		return RunRecord{}, err
	}
	previousStatus := run.Status
	recordEvent(&run, event)

	status, summary := resolveTransition(run, event)
	if status != "" {
		run.Finalize(status, summary)
		run.Audit("event_bus.transition", "event-bus", status, map[string]string{
			"event_type":      event.EventType,
			"previous_status": previousStatus,
			"status":          status,
			"summary":         summary,
			"event_timestamp": event.Timestamp,
		})
	}

	b.runs[run.RunID] = run
	for _, handler := range b.subscribers[event.EventType] {
		handler(event, run)
	}
	if b.ledger != nil {
		if err := b.ledger.Upsert(run); err != nil {
			return RunRecord{}, err
		}
	}
	return run, nil
}

func (b *Bus) resolveRun(runID string) (RunRecord, error) {
	if run, ok := b.runs[runID]; ok {
		return run, nil
	}
	if b.ledger != nil {
		records, err := b.ledger.Load()
		if err != nil {
			return RunRecord{}, err
		}
		for _, run := range records {
			if run.RunID == runID {
				b.runs[runID] = run
				return run, nil
			}
		}
	}
	return RunRecord{}, os.ErrNotExist
}

func recordEvent(run *RunRecord, event BusEvent) {
	details := cloneMap(event.Details)
	details["event_type"] = event.EventType
	details["event_timestamp"] = event.Timestamp
	run.Audit("event_bus.event", event.Actor, "received", details)
	if event.EventType != PullRequestCommentEvent {
		return
	}
	body := strings.TrimSpace(event.Details["body"])
	if body == "" {
		return
	}
	run.Audit("collaboration.comment", event.Actor, "recorded", map[string]string{
		"surface":  "pull-request",
		"body":     body,
		"mentions": strings.TrimSpace(event.Details["mentions"]),
	})
}

func resolveTransition(run RunRecord, event BusEvent) (string, string) {
	explicitStatus := strings.TrimSpace(event.Details["target_status"])
	if explicitStatus != "" {
		return explicitStatus, buildSummary(event, explicitStatus)
	}

	switch event.EventType {
	case PullRequestCommentEvent:
		decision := strings.ToLower(strings.TrimSpace(event.Details["decision"]))
		switch decision {
		case "approved", "accept", "accepted", "lgtm":
			return "approved", buildSummary(event, "approved")
		case "blocked", "changes-requested", "rejected":
			return "needs-approval", buildSummary(event, "needs-approval")
		}
	case CICompletedEvent:
		conclusion := strings.ToLower(strings.TrimSpace(event.Details["conclusion"]))
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
	summary := strings.TrimSpace(event.Details["summary"])
	if summary != "" {
		return summary
	}
	switch event.EventType {
	case PullRequestCommentEvent:
		body := strings.TrimSpace(event.Details["body"])
		if body != "" {
			return body
		}
		return "pull request comment set run to " + status
	case CICompletedEvent:
		workflow := strings.TrimSpace(event.Details["workflow"])
		conclusion := strings.TrimSpace(event.Details["conclusion"])
		if conclusion == "" {
			conclusion = status
		}
		if workflow != "" {
			return "CI workflow " + workflow + " completed with " + conclusion
		}
		return "CI completed with " + conclusion
	case TaskFailedEvent:
		reason := strings.TrimSpace(event.Details["error"])
		if reason == "" {
			reason = strings.TrimSpace(event.Details["reason"])
		}
		if reason != "" {
			return reason
		}
		return "task failed"
	default:
		return status
	}
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
