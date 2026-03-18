package domain

import (
	"encoding/json"
	"math"
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
	ID                        string            `json:"id"`
	TraceID                   string            `json:"trace_id,omitempty"`
	Source                    string            `json:"source,omitempty"`
	Title                     string            `json:"title"`
	Description               string            `json:"description,omitempty"`
	Labels                    []string          `json:"labels,omitempty"`
	Priority                  int               `json:"priority,omitempty"`
	State                     TaskState         `json:"state,omitempty"`
	RiskLevel                 RiskLevel         `json:"risk_level,omitempty"`
	BudgetCents               int64             `json:"budget_cents,omitempty"`
	BudgetOverrideActor       string            `json:"budget_override_actor,omitempty"`
	BudgetOverrideReason      string            `json:"budget_override_reason,omitempty"`
	BudgetOverrideAmountCents int64             `json:"budget_override_amount_cents,omitempty"`
	RequiredTools             []string          `json:"required_tools,omitempty"`
	AcceptanceCriteria        []string          `json:"acceptance_criteria,omitempty"`
	ValidationPlan            []string          `json:"validation_plan,omitempty"`
	RequiredExecutor          ExecutorKind      `json:"required_executor,omitempty"`
	IdempotencyKey            string            `json:"idempotency_key,omitempty"`
	TenantID                  string            `json:"tenant_id,omitempty"`
	ContainerImage            string            `json:"container_image,omitempty"`
	Entrypoint                string            `json:"entrypoint,omitempty"`
	Command                   []string          `json:"command,omitempty"`
	Args                      []string          `json:"args,omitempty"`
	Environment               map[string]string `json:"environment,omitempty"`
	RuntimeEnv                map[string]any    `json:"runtime_env,omitempty"`
	Metadata                  map[string]string `json:"metadata,omitempty"`
	WorkingDir                string            `json:"working_dir,omitempty"`
	ExecutionTimeoutSeconds   int64             `json:"execution_timeout_seconds,omitempty"`
	CreatedAt                 time.Time         `json:"created_at,omitempty"`
	UpdatedAt                 time.Time         `json:"updated_at,omitempty"`
}

func (task Task) MarshalJSON() ([]byte, error) {
	type alias Task
	payload := struct {
		alias
		TaskID               string  `json:"task_id,omitempty"`
		Budget               float64 `json:"budget,omitempty"`
		BudgetOverrideAmount float64 `json:"budget_override_amount,omitempty"`
	}{
		alias: alias(task),
	}
	if task.ID != "" {
		payload.TaskID = task.ID
	}
	if task.BudgetCents != 0 {
		payload.Budget = centsToUSD(task.BudgetCents)
	}
	if task.BudgetOverrideAmountCents != 0 {
		payload.BudgetOverrideAmount = centsToUSD(task.BudgetOverrideAmountCents)
	}
	return json.Marshal(payload)
}

func (task *Task) UnmarshalJSON(data []byte) error {
	type alias Task
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var compat struct {
		TaskID                    string   `json:"task_id"`
		Budget                    *float64 `json:"budget"`
		BudgetCents               *int64   `json:"budget_cents"`
		BudgetOverrideAmount      *float64 `json:"budget_override_amount"`
		BudgetOverrideAmountCents *int64   `json:"budget_override_amount_cents"`
	}
	if err := json.Unmarshal(data, &compat); err != nil {
		return err
	}
	*task = Task(decoded)
	if task.ID == "" {
		task.ID = compat.TaskID
	}
	if compat.BudgetCents == nil && compat.Budget != nil {
		task.BudgetCents = usdToCents(*compat.Budget)
	}
	if compat.BudgetOverrideAmountCents == nil && compat.BudgetOverrideAmount != nil {
		task.BudgetOverrideAmountCents = usdToCents(*compat.BudgetOverrideAmount)
	}
	return nil
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

func usdToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

func centsToUSD(amount int64) float64 {
	return float64(amount) / 100
}
