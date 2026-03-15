package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/flow"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

type WorkerStatusProvider interface {
	Snapshot() worker.Status
}

type WorkerPoolStatusProvider interface {
	WorkerStatusProvider
	Snapshots() []worker.Status
}

type Server struct {
	Recorder         *observability.Recorder
	Queue            queue.Queue
	Executors        []domain.ExecutorKind
	Bus              *events.Bus
	EventLog         events.EventLog
	Now              func() time.Time
	Worker           WorkerStatusProvider
	Control          *control.Controller
	FlowStore        *flow.Store
	SchedulerPolicy  *scheduler.PolicyStore
	SchedulerRuntime *scheduler.Scheduler
}

func (s *Server) Handler() http.Handler {
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Control == nil {
		s.Control = control.New()
	}
	if s.FlowStore == nil {
		s.FlowStore = flow.NewStore()
	}
	if s.SchedulerPolicy == nil {
		s.SchedulerPolicy = scheduler.NewDefaultPolicyStore()
	}
	if s.SchedulerRuntime == nil {
		s.SchedulerRuntime = scheduler.NewWithStores(s.SchedulerPolicy, nil)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		taskID := r.URL.Query().Get("task_id")
		traceID := r.URL.Query().Get("trace_id")
		subscriberID := strings.TrimSpace(r.URL.Query().Get("subscriber_id"))
		afterID, err := s.resolveAfterID(subscriberID, strings.TrimSpace(r.URL.Query().Get("after_id")))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		eventTypes := parseEventTypes(r.URL.Query()["event_type"])
		events, backend, err := s.queryEvents(taskID, traceID, afterID, eventTypes, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"events":        events,
			"subscriber_id": subscriberID,
			"after_id":      afterID,
			"next_after_id": nextAfterID(events, afterID),
			"event_types":   eventTypes,
			"backend":       backend,
			"durable":       s.EventLog != nil,
		})
	})
	mux.HandleFunc("/stream/events/checkpoints/", s.handleStreamEventCheckpoint)
	mux.HandleFunc("/stream/events", s.handleStreamEvents)
	mux.HandleFunc("/audit", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		writeJSON(w, http.StatusOK, map[string]any{"audit": s.Recorder.EventsByTask("", limit)})
	})
	mux.HandleFunc("/replay/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		taskID := strings.TrimPrefix(r.URL.Path, "/replay/")
		if taskID == "" {
			http.Error(w, "missing task id", http.StatusBadRequest)
			return
		}
		events := s.Recorder.EventsByTask(taskID, 1000)
		writeJSON(w, http.StatusOK, map[string]any{"task_id": taskID, "timeline": events})
	})
	mux.HandleFunc("/deadletters", s.handleDeadLetters)
	mux.HandleFunc("/deadletters/", s.handleDeadLetterAction)
	mux.HandleFunc("/debug/traces", s.handleDebugTraces)
	mux.HandleFunc("/debug/traces/", s.handleDebugTrace)
	mux.HandleFunc("/debug/status", func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]any{
			"queue_size":   s.Queue.Size(context.Background()),
			"audit_events": len(s.Recorder.Logs()),
			"executors":    s.executorNames(),
		}
		if s.Worker != nil {
			payload["worker"] = s.Worker.Snapshot()
		}
		if pool := s.workerPoolSummary(); pool != nil {
			payload["worker_pool"] = pool
		}
		if s.Control != nil {
			payload["control"] = s.Control.Snapshot()
		}
		writeJSON(w, http.StatusOK, payload)
	})
	mux.HandleFunc("/v2/dashboard/engineering", s.handleV2EngineeringDashboard)
	mux.HandleFunc("/v2/dashboard/operations", s.handleV2OperationsDashboard)
	mux.HandleFunc("/v2/triage/center", s.handleV2TriageCenter)
	mux.HandleFunc("/v2/regression/center", s.handleV2RegressionCenter)
	mux.HandleFunc("/v2/control-center", s.handleV2ControlCenter)
	mux.HandleFunc("/v2/control-center/audit", s.handleV2ControlCenterAudit)
	mux.HandleFunc("/v2/control-center/actions", s.handleV2ControlCenterAction)
	mux.HandleFunc("/v2/control-center/policy", s.handleV2ControlCenterPolicy)
	mux.HandleFunc("/v2/control-center/policy/reload", s.handleV2ControlCenterPolicyReload)
	mux.Handle("/internal/scheduler/fairness/", http.StripPrefix("/internal/scheduler/fairness", s.schedulerRuntime().FairnessServiceHandler()))
	mux.HandleFunc("/v2/reports/weekly", s.handleV2WeeklyReport)
	mux.HandleFunc("/v2/reports/weekly/export", s.handleV2WeeklyReportExport)
	mux.HandleFunc("/v2/reports/distributed", s.handleV2DistributedReport)
	mux.HandleFunc("/v2/reports/distributed/export", s.handleV2DistributedReportExport)
	mux.HandleFunc("/v2/flows/templates", s.handleV2FlowTemplates)
	mux.HandleFunc("/v2/flows/templates/", s.handleV2FlowTemplateAction)
	mux.HandleFunc("/v2/flows/overview", s.handleV2FlowOverview)
	mux.HandleFunc("/v2/prd/intake", s.handleV2PRDIntake)
	mux.HandleFunc("/v2/launch/checklist", s.handleV2LaunchChecklist)
	mux.HandleFunc("/v2/support/handoff", s.handleV2SupportHandoff)
	mux.HandleFunc("/v2/navigation", s.handleV2Navigation)
	mux.HandleFunc("/v2/home", s.handleV2Home)
	mux.HandleFunc("/v2/design-system", s.handleV2DesignSystem)
	mux.HandleFunc("/v2/billing/usage", s.handleV2BillingUsage)
	mux.HandleFunc("/v2/billing/entitlements", s.handleV2BillingEntitlements)
	mux.HandleFunc("/v2/runs/", s.handleV2RunDetail)
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateTask(w, r)
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"message": "use POST /tasks or GET /tasks/{id}"})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/tasks/", s.handleTaskStatus)
	return mux
}

func (s *Server) handleDebugTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, map[string]any{"traces": s.Recorder.TraceSummaries(limit)})
}

func (s *Server) handleDebugTrace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	traceID := strings.TrimPrefix(r.URL.Path, "/debug/traces/")
	if traceID == "" {
		http.Error(w, "missing trace id", http.StatusBadRequest)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	summary, ok := s.Recorder.TraceSummary(traceID)
	if !ok {
		http.Error(w, "trace not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"trace":  summary,
		"events": s.Recorder.EventsByTrace(traceID, limit),
	})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, fmt.Sprintf("decode task: %v", err), http.StatusBadRequest)
		return
	}
	now := s.Now()
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", now.UnixNano())
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = now
	}
	if task.Title == "" {
		task.Title = task.ID
	}
	if task.TraceID == "" {
		task.TraceID = task.ID
	}
	task.State = domain.TaskQueued
	if err := s.Queue.Enqueue(r.Context(), task); err != nil {
		http.Error(w, fmt.Sprintf("enqueue task: %v", err), http.StatusInternalServerError)
		return
	}
	s.Recorder.StoreTask(task)
	s.publish(domain.Event{ID: task.ID + "-queued", Type: domain.EventTaskQueued, TaskID: task.ID, TraceID: task.TraceID, Timestamp: now, Payload: map[string]any{"executor": task.RequiredExecutor, "title": task.Title}})
	writeJSON(w, http.StatusAccepted, map[string]any{"task": task, "state": string(task.State)})
}

func (s *Server) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	taskID := strings.TrimPrefix(r.URL.Path, "/tasks/")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	events := s.Recorder.EventsByTask(taskID, 100)
	task, hasTask := s.Recorder.Task(taskID)
	if len(events) == 0 && !hasTask {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	latest, hasLatest := s.Recorder.LatestByTask(taskID)
	traceID := task.TraceID
	state := string(task.State)
	if traceID == "" && hasLatest {
		traceID = latest.TraceID
	}
	if state == "" && hasLatest {
		state = eventState(latest.Type)
	}
	if state == "" {
		state = "unknown"
	}
	payload := map[string]any{
		"task_id":  taskID,
		"trace_id": traceID,
		"state":    state,
		"events":   events,
	}
	if hasLatest {
		payload["latest_event"] = latest
	}
	if hasTask {
		payload["task"] = task
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleDeadLetters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.Queue.ListDeadLetters(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("list dead letters: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dead_letters": items})
}

func (s *Server) handleDeadLetterAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/deadletters/")
	if !strings.HasSuffix(path, "/replay") {
		http.Error(w, "unsupported action", http.StatusNotFound)
		return
	}
	taskID := strings.TrimSuffix(path, "/replay")
	taskID = strings.TrimSuffix(taskID, "/")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	if err := s.Queue.ReplayDeadLetter(r.Context(), taskID); err != nil {
		switch {
		case strings.Contains(err.Error(), "not found"):
			http.Error(w, err.Error(), http.StatusNotFound)
		case strings.Contains(err.Error(), "not dead-lettered"):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, fmt.Sprintf("replay dead letter: %v", err), http.StatusInternalServerError)
		}
		return
	}
	now := s.Now()
	s.syncTaskState(taskID, domain.TaskQueued, now)
	traceID := s.traceIDForTask(taskID)
	s.publish(domain.Event{ID: taskID + "-replayed", Type: domain.EventTaskQueued, TaskID: taskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"replayed": true}})
	writeJSON(w, http.StatusAccepted, map[string]any{"task_id": taskID, "state": string(domain.TaskQueued), "replayed": true})
}

func (s *Server) handleStreamEventCheckpoint(w http.ResponseWriter, r *http.Request) {
	store := s.checkpointStore()
	if store == nil {
		http.Error(w, "checkpoint store unavailable", http.StatusServiceUnavailable)
		return
	}
	subscriberID := strings.TrimPrefix(r.URL.Path, "/stream/events/checkpoints/")
	if subscriberID == "" {
		http.Error(w, "missing subscriber id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		checkpoint, err := store.Checkpoint(subscriberID)
		if err != nil {
			if events.IsNoEventLog(err) {
				http.Error(w, "checkpoint not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"checkpoint": checkpoint,
			"backend":    "sqlite",
			"durable":    s.EventLog != nil,
		})
	case http.MethodPost:
		var payload struct {
			EventID string `json:"event_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("decode checkpoint ack: %v", err), http.StatusBadRequest)
			return
		}
		checkpoint, err := store.Acknowledge(subscriberID, strings.TrimSpace(payload.EventID), s.Now())
		if err != nil {
			if events.IsNoEventLog(err) {
				http.Error(w, "event not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"checkpoint": checkpoint,
			"backend":    "sqlite",
			"durable":    s.EventLog != nil,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleStreamEvents(w http.ResponseWriter, r *http.Request) {
	if s.Bus == nil {
		http.Error(w, "event bus unavailable", http.StatusServiceUnavailable)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	taskID := r.URL.Query().Get("task_id")
	traceID := r.URL.Query().Get("trace_id")
	subscriberID := strings.TrimSpace(r.URL.Query().Get("subscriber_id"))
	afterID := strings.TrimSpace(r.URL.Query().Get("after_id"))
	if afterID == "" {
		afterID = strings.TrimSpace(r.Header.Get("Last-Event-ID"))
	}
	afterID, err := s.resolveAfterID(subscriberID, afterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	eventTypes := parseEventTypes(r.URL.Query()["event_type"])
	eventTypeSet := toEventTypeSet(eventTypes)
	replay := r.URL.Query().Get("replay") == "1" || afterID != ""
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := s.Bus.SubscribeTopic(128, events.SubscriptionFilter{TaskID: taskID, TraceID: traceID, EventTypes: eventTypeSet})
	defer cancel()
	delivered := make(map[string]struct{})
	if replay {
		history, _, err := s.queryEvents(taskID, traceID, afterID, eventTypes, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, event := range history {
			if !markStreamEventDelivered(delivered, event) {
				continue
			}
			if err := writeSSEEvent(w, flusher, event); err != nil {
				return
			}
		}
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if !matchesEventStreamFilter(event, taskID, traceID, eventTypeSet) {
				continue
			}
			if !markStreamEventDelivered(delivered, event) {
				continue
			}
			if err := writeSSEEvent(w, flusher, event); err != nil {
				return
			}
		}
	}
}

func (s *Server) queryEvents(taskID, traceID, afterID string, eventTypes []domain.EventType, limit int) ([]domain.Event, string, error) {
	backendLimit := limit
	if len(eventTypes) > 0 {
		backendLimit = 0
	}
	if s.EventLog != nil {
		var history []domain.Event
		var err error
		if afterID != "" {
			switch {
			case taskID != "":
				history, err = s.EventLog.EventsByTaskAfter(taskID, afterID, backendLimit)
			case traceID != "":
				history, err = s.EventLog.EventsByTraceAfter(traceID, afterID, backendLimit)
			default:
				history, err = s.EventLog.ReplayAfter(afterID, backendLimit)
			}
		} else {
			switch {
			case taskID != "":
				history, err = s.EventLog.EventsByTask(taskID, backendLimit)
			case traceID != "":
				history, err = s.EventLog.EventsByTrace(traceID, backendLimit)
			default:
				history, err = s.EventLog.Replay(backendLimit)
			}
		}
		if err != nil {
			return nil, "sqlite", err
		}
		return limitQueriedEvents(filterEventsByType(history, eventTypes), afterID, limit), "sqlite", nil
	}
	history := s.queryMemoryEvents(taskID, traceID, afterID, backendLimit)
	return limitQueriedEvents(filterEventsByType(history, eventTypes), afterID, limit), "memory", nil
}

func (s *Server) queryMemoryEvents(taskID, traceID, afterID string, limit int) []domain.Event {
	if s.Recorder == nil {
		return nil
	}
	logs := s.Recorder.Logs()
	if afterID == "" {
		filtered := make([]domain.Event, 0, len(logs))
		for _, event := range logs {
			if !matchesEventStreamFilter(event, taskID, traceID, nil) {
				continue
			}
			filtered = append(filtered, event)
		}
		if limit > 0 && len(filtered) > limit {
			filtered = filtered[len(filtered)-limit:]
		}
		out := make([]domain.Event, len(filtered))
		copy(out, filtered)
		return out
	}
	start := 0
	for index, event := range logs {
		if event.ID == afterID {
			start = index + 1
		}
	}
	filtered := make([]domain.Event, 0, len(logs)-start)
	for _, event := range logs[start:] {
		if !matchesEventStreamFilter(event, taskID, traceID, nil) {
			continue
		}
		filtered = append(filtered, event)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func (s *Server) checkpointStore() events.CheckpointStore {
	if store, ok := s.EventLog.(events.CheckpointStore); ok {
		return store
	}
	return nil
}

func (s *Server) resolveAfterID(subscriberID string, afterID string) (string, error) {
	if afterID != "" || subscriberID == "" {
		return afterID, nil
	}
	store := s.checkpointStore()
	if store == nil {
		return "", nil
	}
	checkpoint, err := store.Checkpoint(subscriberID)
	if err != nil {
		if events.IsNoEventLog(err) {
			return "", nil
		}
		return "", err
	}
	return checkpoint.EventID, nil
}

func parseEventTypes(values []string) []domain.EventType {
	seen := make(map[domain.EventType]struct{})
	out := make([]domain.EventType, 0)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			eventType := domain.EventType(part)
			if _, ok := seen[eventType]; ok {
				continue
			}
			seen[eventType] = struct{}{}
			out = append(out, eventType)
		}
	}
	return out
}

func toEventTypeSet(eventTypes []domain.EventType) map[domain.EventType]struct{} {
	if len(eventTypes) == 0 {
		return nil
	}
	set := make(map[domain.EventType]struct{}, len(eventTypes))
	for _, eventType := range eventTypes {
		set[eventType] = struct{}{}
	}
	return set
}

func filterEventsByType(events []domain.Event, eventTypes []domain.EventType) []domain.Event {
	if len(eventTypes) == 0 {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	allowed := toEventTypeSet(eventTypes)
	filtered := make([]domain.Event, 0, len(events))
	for _, event := range events {
		if _, ok := allowed[event.Type]; ok {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func limitQueriedEvents(events []domain.Event, afterID string, limit int) []domain.Event {
	if limit <= 0 || len(events) <= limit {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	if afterID != "" {
		out := make([]domain.Event, limit)
		copy(out, events[:limit])
		return out
	}
	out := make([]domain.Event, limit)
	copy(out, events[len(events)-limit:])
	return out
}

func nextAfterID(events []domain.Event, fallback string) string {
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].ID != "" {
			return events[index].ID
		}
	}
	return fallback
}

func markStreamEventDelivered(delivered map[string]struct{}, event domain.Event) bool {
	key := streamEventKey(event)
	if _, ok := delivered[key]; ok {
		return false
	}
	delivered[key] = struct{}{}
	return true
}

func streamEventKey(event domain.Event) string {
	if event.ID != "" {
		return "id:" + event.ID
	}
	return fmt.Sprintf("anon:%s:%s:%s:%s:%d", event.Type, event.TaskID, event.TraceID, event.RunID, event.Timestamp.UnixNano())
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event domain.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n", payload); err != nil {
		return err
	}
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func matchesEventStreamFilter(event domain.Event, taskID string, traceID string, eventTypes map[domain.EventType]struct{}) bool {
	if taskID != "" && event.TaskID != taskID {
		return false
	}
	if traceID != "" && event.TraceID != traceID {
		return false
	}
	if len(eventTypes) > 0 {
		if _, ok := eventTypes[event.Type]; !ok {
			return false
		}
	}
	return true
}

func (s *Server) publish(event domain.Event) {
	if s.Bus != nil {
		s.Bus.Publish(event)
		return
	}
	if s.Recorder != nil {
		s.Recorder.Record(event)
	}
}

func (s *Server) executorNames() []string {
	names := make([]string, 0, len(s.Executors))
	for _, executor := range s.Executors {
		names = append(names, string(executor))
	}
	sort.Strings(names)
	return names
}

func eventState(eventType domain.EventType) string {
	if state, ok := domain.TaskStateFromEventType(eventType); ok {
		return string(state)
	}
	return "unknown"
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
