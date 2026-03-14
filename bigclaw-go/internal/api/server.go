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
	Recorder  *observability.Recorder
	Queue     queue.Queue
	Executors []domain.ExecutorKind
	Bus       *events.Bus
	Now       func() time.Time
	Worker    WorkerStatusProvider
	Control   *control.Controller
	FlowStore *flow.Store
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
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"queue_size":           s.Queue.Size(context.Background()),
			"events":               s.Recorder.Snapshot(),
			"trace_count":          len(s.Recorder.TraceSummaries(0)),
			"registered_executors": s.executorNames(),
		})
	})
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		taskID := r.URL.Query().Get("task_id")
		traceID := r.URL.Query().Get("trace_id")
		switch {
		case taskID != "":
			writeJSON(w, http.StatusOK, map[string]any{"events": s.Recorder.EventsByTask(taskID, limit)})
		case traceID != "":
			writeJSON(w, http.StatusOK, map[string]any{"events": s.Recorder.EventsByTrace(traceID, limit)})
		default:
			writeJSON(w, http.StatusOK, map[string]any{"events": s.Recorder.EventsByTask("", limit)})
		}
	})
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

func (s *Server) handleStreamEvents(w http.ResponseWriter, r *http.Request) {
	if s.Bus == nil {
		http.Error(w, "event bus unavailable", http.StatusServiceUnavailable)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	replay := r.URL.Query().Get("replay") == "1"
	taskID := r.URL.Query().Get("task_id")
	traceID := r.URL.Query().Get("trace_id")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	var ch <-chan domain.Event
	var cancel func()
	if replay {
		ch, cancel = s.Bus.SubscribeReplay(128, limit)
	} else {
		ch, cancel = s.Bus.Subscribe(128)
	}
	defer cancel()
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if !matchesEventStreamFilter(event, taskID, traceID) {
				continue
			}
			payload, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

func matchesEventStreamFilter(event domain.Event, taskID string, traceID string) bool {
	if taskID != "" && event.TaskID != taskID {
		return false
	}
	if traceID != "" && event.TraceID != traceID {
		return false
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
