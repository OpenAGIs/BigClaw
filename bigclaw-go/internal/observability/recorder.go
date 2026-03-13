package observability

import (
	"sort"
	"sync"
	"time"

	"bigclaw-go/internal/domain"
)

type TraceSummary struct {
	TraceID         string             `json:"trace_id"`
	TaskIDs         []string           `json:"task_ids"`
	EventCount      int                `json:"event_count"`
	EventTypes      []domain.EventType `json:"event_types"`
	FirstSeenAt     time.Time          `json:"first_seen_at"`
	LastSeenAt      time.Time          `json:"last_seen_at"`
	DurationSeconds float64            `json:"duration_seconds"`
	LatestEventType domain.EventType   `json:"latest_event_type"`
}

type Recorder struct {
	mu       sync.Mutex
	counters map[domain.EventType]int
	logs     []domain.Event
	latest   map[string]domain.Event
	tasks    map[string]domain.Task
	sinks    []Sink
}

func NewRecorder() *Recorder {
	return NewRecorderWithSinks()
}

func NewRecorderWithSinks(sinks ...Sink) *Recorder {
	return &Recorder{
		counters: make(map[domain.EventType]int),
		latest:   make(map[string]domain.Event),
		tasks:    make(map[string]domain.Task),
		sinks:    sinks,
	}
}

func (r *Recorder) Record(event domain.Event) {
	r.mu.Lock()
	r.counters[event.Type]++
	r.logs = append(r.logs, event)
	if event.TaskID != "" {
		r.latest[event.TaskID] = event
		r.applyEventToTaskLocked(event)
	}
	sinks := append([]Sink(nil), r.sinks...)
	r.mu.Unlock()
	for _, sink := range sinks {
		_ = sink.Write(event)
	}
}

func (r *Recorder) StoreTask(task domain.Task) {
	if task.ID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.tasks[task.ID]; ok {
		if task.TraceID == "" {
			task.TraceID = existing.TraceID
		}
		if task.State == "" {
			task.State = existing.State
		}
		if task.CreatedAt.IsZero() {
			task.CreatedAt = existing.CreatedAt
		}
		if task.UpdatedAt.IsZero() {
			task.UpdatedAt = existing.UpdatedAt
		}
	}
	if task.TraceID == "" {
		task.TraceID = task.ID
	}
	r.tasks[task.ID] = cloneTask(task)
}

func (r *Recorder) Task(taskID string) (domain.Task, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[taskID]
	if !ok {
		return domain.Task{}, false
	}
	return cloneTask(task), true
}

func (r *Recorder) Tasks(limit int) []domain.Task {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		out = append(out, cloneTask(task))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			if out[i].CreatedAt.Equal(out[j].CreatedAt) {
				return out[i].ID < out[j].ID
			}
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (r *Recorder) Snapshot() map[domain.EventType]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[domain.EventType]int, len(r.counters))
	for key, value := range r.counters {
		out[key] = value
	}
	return out
}

func (r *Recorder) Logs() []domain.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Event, len(r.logs))
	copy(out, r.logs)
	return out
}

func (r *Recorder) EventsByTask(taskID string, limit int) []domain.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	if taskID == "" {
		return r.limitEvents(r.logs, limit)
	}
	filtered := make([]domain.Event, 0)
	for _, event := range r.logs {
		if event.TaskID == taskID {
			filtered = append(filtered, event)
		}
	}
	return r.limitEvents(filtered, limit)
}

func (r *Recorder) EventsByTrace(traceID string, limit int) []domain.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	if traceID == "" {
		return r.limitEvents(r.logs, limit)
	}
	filtered := make([]domain.Event, 0)
	for _, event := range r.logs {
		if event.TraceID == traceID {
			filtered = append(filtered, event)
		}
	}
	return r.limitEvents(filtered, limit)
}

func (r *Recorder) LatestByTask(taskID string) (domain.Event, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	event, ok := r.latest[taskID]
	return event, ok
}

func (r *Recorder) TraceSummary(traceID string) (TraceSummary, bool) {
	logs := r.Logs()
	filtered := filterEventsByTrace(logs, traceID)
	if len(filtered) == 0 {
		return TraceSummary{}, false
	}
	return summarizeTrace(traceID, filtered), true
}

func (r *Recorder) TraceSummaries(limit int) []TraceSummary {
	logs := r.Logs()
	traceOrder := make([]string, 0)
	seen := make(map[string]struct{})
	for index := len(logs) - 1; index >= 0; index-- {
		traceID := logs[index].TraceID
		if traceID == "" {
			continue
		}
		if _, ok := seen[traceID]; ok {
			continue
		}
		seen[traceID] = struct{}{}
		traceOrder = append(traceOrder, traceID)
		if limit > 0 && len(traceOrder) >= limit {
			break
		}
	}
	out := make([]TraceSummary, 0, len(traceOrder))
	for _, traceID := range traceOrder {
		filtered := filterEventsByTrace(logs, traceID)
		if len(filtered) == 0 {
			continue
		}
		out = append(out, summarizeTrace(traceID, filtered))
	}
	return out
}

func (r *Recorder) applyEventToTaskLocked(event domain.Event) {
	task, ok := r.tasks[event.TaskID]
	if !ok {
		task = domain.Task{ID: event.TaskID}
	}
	if task.TraceID == "" {
		task.TraceID = event.TraceID
	}
	if !event.Timestamp.IsZero() {
		task.UpdatedAt = event.Timestamp
	}
	if state, ok := domain.TaskStateFromEventType(event.Type); ok {
		task.State = state
	}
	r.tasks[event.TaskID] = cloneTask(task)
}

func (r *Recorder) limitEvents(events []domain.Event, limit int) []domain.Event {
	if limit <= 0 || len(events) <= limit {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	out := make([]domain.Event, limit)
	copy(out, events[len(events)-limit:])
	return out
}

func filterEventsByTrace(events []domain.Event, traceID string) []domain.Event {
	if traceID == "" {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	filtered := make([]domain.Event, 0)
	for _, event := range events {
		if event.TraceID == traceID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func summarizeTrace(traceID string, events []domain.Event) TraceSummary {
	summary := TraceSummary{
		TraceID:         traceID,
		TaskIDs:         uniqueTaskIDs(events),
		EventCount:      len(events),
		EventTypes:      eventTypes(events),
		FirstSeenAt:     events[0].Timestamp,
		LastSeenAt:      events[len(events)-1].Timestamp,
		LatestEventType: events[len(events)-1].Type,
	}
	summary.DurationSeconds = summary.LastSeenAt.Sub(summary.FirstSeenAt).Seconds()
	if summary.DurationSeconds < 0 {
		summary.DurationSeconds = 0
	}
	return summary
}

func uniqueTaskIDs(events []domain.Event) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, event := range events {
		if event.TaskID == "" {
			continue
		}
		if _, ok := seen[event.TaskID]; ok {
			continue
		}
		seen[event.TaskID] = struct{}{}
		out = append(out, event.TaskID)
	}
	return out
}

func eventTypes(events []domain.Event) []domain.EventType {
	out := make([]domain.EventType, 0, len(events))
	for _, event := range events {
		out = append(out, event.Type)
	}
	return out
}

func cloneTask(task domain.Task) domain.Task {
	clone := task
	if len(task.Labels) > 0 {
		clone.Labels = append([]string(nil), task.Labels...)
	}
	if len(task.RequiredTools) > 0 {
		clone.RequiredTools = append([]string(nil), task.RequiredTools...)
	}
	if len(task.AcceptanceCriteria) > 0 {
		clone.AcceptanceCriteria = append([]string(nil), task.AcceptanceCriteria...)
	}
	if len(task.ValidationPlan) > 0 {
		clone.ValidationPlan = append([]string(nil), task.ValidationPlan...)
	}
	if len(task.Command) > 0 {
		clone.Command = append([]string(nil), task.Command...)
	}
	if len(task.Args) > 0 {
		clone.Args = append([]string(nil), task.Args...)
	}
	if len(task.Environment) > 0 {
		clone.Environment = make(map[string]string, len(task.Environment))
		for key, value := range task.Environment {
			clone.Environment[key] = value
		}
	}
	if len(task.RuntimeEnv) > 0 {
		clone.RuntimeEnv = make(map[string]any, len(task.RuntimeEnv))
		for key, value := range task.RuntimeEnv {
			clone.RuntimeEnv[key] = value
		}
	}
	if len(task.Metadata) > 0 {
		clone.Metadata = make(map[string]string, len(task.Metadata))
		for key, value := range task.Metadata {
			clone.Metadata[key] = value
		}
	}
	return clone
}
