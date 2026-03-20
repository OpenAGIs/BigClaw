package domain

import (
	"encoding/json"
	"math"
	"strings"
	"time"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type TaskState string

const (
	TaskQueued     TaskState = "queued"
	TaskLeased     TaskState = "leased"
	TaskRunning    TaskState = "running"
	TaskBlocked    TaskState = "blocked"
	TaskRetrying   TaskState = "retrying"
	TaskSucceeded  TaskState = "succeeded"
	TaskFailed     TaskState = "failed"
	TaskCancelled  TaskState = "cancelled"
	TaskDeadLetter TaskState = "dead_letter"
)

type ExecutorKind string

const (
	ExecutorLocal      ExecutorKind = "local"
	ExecutorKubernetes ExecutorKind = "kubernetes"
	ExecutorRay        ExecutorKind = "ray"
)

type Task struct {
	ID                      string            `json:"id"`
	TraceID                 string            `json:"trace_id,omitempty"`
	Source                  string            `json:"source,omitempty"`
	Title                   string            `json:"title"`
	Description             string            `json:"description,omitempty"`
	Labels                  []string          `json:"labels,omitempty"`
	Priority                int               `json:"priority,omitempty"`
	State                   TaskState         `json:"state,omitempty"`
	RiskLevel               RiskLevel         `json:"risk_level,omitempty"`
	BudgetCents             int64             `json:"budget_cents,omitempty"`
	BudgetOverrideActor     string            `json:"budget_override_actor,omitempty"`
	BudgetOverrideReason    string            `json:"budget_override_reason,omitempty"`
	BudgetOverrideAmount    float64           `json:"budget_override_amount,omitempty"`
	RequiredTools           []string          `json:"required_tools,omitempty"`
	AcceptanceCriteria      []string          `json:"acceptance_criteria,omitempty"`
	ValidationPlan          []string          `json:"validation_plan,omitempty"`
	RequiredExecutor        ExecutorKind      `json:"required_executor,omitempty"`
	IdempotencyKey          string            `json:"idempotency_key,omitempty"`
	TenantID                string            `json:"tenant_id,omitempty"`
	ContainerImage          string            `json:"container_image,omitempty"`
	Entrypoint              string            `json:"entrypoint,omitempty"`
	Command                 []string          `json:"command,omitempty"`
	Args                    []string          `json:"args,omitempty"`
	Environment             map[string]string `json:"environment,omitempty"`
	RuntimeEnv              map[string]any    `json:"runtime_env,omitempty"`
	Metadata                map[string]string `json:"metadata,omitempty"`
	WorkingDir              string            `json:"working_dir,omitempty"`
	ExecutionTimeoutSeconds int64             `json:"execution_timeout_seconds,omitempty"`
	CreatedAt               time.Time         `json:"created_at,omitempty"`
	UpdatedAt               time.Time         `json:"updated_at,omitempty"`
}

type RunAttempt struct {
	TaskID         string       `json:"task_id"`
	Attempt        int          `json:"attempt"`
	State          TaskState    `json:"state"`
	WorkerID       string       `json:"worker_id"`
	Executor       ExecutorKind `json:"executor"`
	LeaseExpiresAt time.Time    `json:"lease_expires_at"`
	StartedAt      time.Time    `json:"started_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	Error          string       `json:"error,omitempty"`
}

type EventType string

const (
	EventTaskQueued      EventType = "task.queued"
	EventTaskLeased      EventType = "task.leased"
	EventTaskStarted     EventType = "task.started"
	EventTaskCompleted   EventType = "task.completed"
	EventTaskRetried     EventType = "task.retried"
	EventTaskPreempted   EventType = "task.preempted"
	EventTaskCancelled   EventType = "task.cancelled"
	EventTaskDeadLetter  EventType = "task.dead_lettered"
	EventSchedulerRouted EventType = "scheduler.routed"
	EventControlPaused   EventType = "control.paused"
	EventControlResumed  EventType = "control.resumed"
	EventRunTakeover     EventType = "run.takeover"
	EventRunReleased     EventType = "run.released"
	EventRunAnnotated    EventType = "run.annotated"
)

type Event struct {
	ID        string         `json:"id"`
	Type      EventType      `json:"type"`
	TaskID    string         `json:"task_id,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	RunID     string         `json:"run_id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload,omitempty"`
	Delivery  *EventDelivery `json:"delivery,omitempty"`
}

type EventDeliveryMode string

const (
	EventDeliveryModeLive   EventDeliveryMode = "live"
	EventDeliveryModeReplay EventDeliveryMode = "replay"
)

type EventDelivery struct {
	Mode           EventDeliveryMode `json:"mode,omitempty"`
	Replay         bool              `json:"replay,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

func TaskStateFromEventType(eventType EventType) (TaskState, bool) {
	switch eventType {
	case EventTaskQueued:
		return TaskQueued, true
	case EventTaskLeased, EventSchedulerRouted:
		return TaskLeased, true
	case EventTaskStarted:
		return TaskRunning, true
	case EventTaskRetried:
		return TaskRetrying, true
	case EventTaskCompleted:
		return TaskSucceeded, true
	case EventTaskPreempted, EventTaskCancelled:
		return TaskCancelled, true
	case EventTaskDeadLetter:
		return TaskDeadLetter, true
	default:
		return "", false
	}
}

func IsActiveTaskState(state TaskState) bool {
	switch state {
	case TaskQueued, TaskLeased, TaskRunning, TaskBlocked, TaskRetrying:
		return true
	default:
		return false
	}
}

type taskJSONAlias Task

type taskJSONEnvelope struct {
	taskJSONAlias
	TaskID string   `json:"task_id,omitempty"`
	Budget *float64 `json:"budget,omitempty"`
}

func (t Task) MarshalJSON() ([]byte, error) {
	payload := taskJSONEnvelope{taskJSONAlias: taskJSONAlias(t)}
	payload.TaskID = t.ID
	if t.BudgetCents != 0 {
		budget := float64(t.BudgetCents) / 100
		payload.Budget = &budget
	}
	return json.Marshal(payload)
}

func (t *Task) UnmarshalJSON(data []byte) error {
	var payload taskJSONEnvelope
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	task := Task(payload.taskJSONAlias)
	if task.ID == "" {
		task.ID = payload.TaskID
	}
	if payload.Budget != nil && task.BudgetCents == 0 {
		task.BudgetCents = int64(math.Round(*payload.Budget * 100))
	}
	task.State = normalizeTaskState(task.State)
	*t = task
	return nil
}

func normalizeTaskState(state TaskState) TaskState {
	normalized := strings.ToLower(strings.TrimSpace(string(state)))
	switch normalized {
	case "", string(TaskQueued), "todo":
		return TaskQueued
	case string(TaskLeased):
		return TaskLeased
	case string(TaskRunning):
		return TaskRunning
	case string(TaskBlocked):
		return TaskBlocked
	case string(TaskRetrying):
		return TaskRetrying
	case string(TaskSucceeded):
		return TaskSucceeded
	case string(TaskFailed):
		return TaskFailed
	case "cancelled", "canceled":
		return TaskCancelled
	case string(TaskDeadLetter):
		return TaskDeadLetter
	}
	switch {
	case strings.Contains(normalized, "progress"):
		return TaskRunning
	case strings.Contains(normalized, "done"), strings.Contains(normalized, "closed"), strings.Contains(normalized, "resolved"):
		return TaskSucceeded
	case strings.Contains(normalized, "block"):
		return TaskBlocked
	case strings.Contains(normalized, "fail"):
		return TaskFailed
	default:
		return state
	}
}
