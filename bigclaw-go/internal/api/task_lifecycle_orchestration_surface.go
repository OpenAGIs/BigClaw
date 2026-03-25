package api

import (
	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
)

type taskLifecycleOrchestrationSurface struct {
	Title              string                            `json:"title"`
	Summary            taskLifecycleOrchestrationSummary `json:"summary"`
	Lifecycle          lifecycleOrchestrationModel       `json:"lifecycle"`
	BatchStartStrategy lifecycleBatchStrategy            `json:"batch_start_strategy"`
	BatchStopStrategy  lifecycleBatchStrategy            `json:"batch_stop_strategy"`
}

type taskLifecycleOrchestrationSummary struct {
	ControllerState     string `json:"controller_state"`
	ControllerPaused    bool   `json:"controller_paused"`
	ActiveTakeovers     int    `json:"active_takeovers"`
	QueueCancelEnabled  bool   `json:"queue_cancel_enabled"`
	DefaultDispatchMode string `json:"default_dispatch_mode"`
	BatchStartAction    string `json:"batch_start_action"`
	BatchStopAction     string `json:"batch_stop_action"`
}

type lifecycleOrchestrationModel struct {
	States               []string              `json:"states"`
	AutomaticTransitions []lifecycleTransition `json:"automatic_transitions"`
	OperatorTransitions  []lifecycleTransition `json:"operator_transitions"`
	InterventionRules    []string              `json:"intervention_rules"`
	TerminalStates       []string              `json:"terminal_states"`
}

type lifecycleTransition struct {
	From   []string `json:"from"`
	Action string   `json:"action"`
	To     string   `json:"to"`
}

type lifecycleBatchStrategy struct {
	Intent          string   `json:"intent"`
	TriggerAction   string   `json:"trigger_action"`
	AppliesToStates []string `json:"applies_to_states"`
	Sequence        []string `json:"sequence"`
	Guardrails      []string `json:"guardrails"`
}

func taskLifecycleOrchestrationSurfacePayload(snapshot control.Snapshot, queueCancelEnabled bool) taskLifecycleOrchestrationSurface {
	controllerState := "running"
	if snapshot.Paused {
		controllerState = "paused"
	}
	return taskLifecycleOrchestrationSurface{
		Title: "Task Lifecycle Orchestration Overview",
		Summary: taskLifecycleOrchestrationSummary{
			ControllerState:     controllerState,
			ControllerPaused:    snapshot.Paused,
			ActiveTakeovers:     snapshot.ActiveTakeovers,
			QueueCancelEnabled:  queueCancelEnabled,
			DefaultDispatchMode: "scheduler_driven_fifo_with_takeover_override",
			BatchStartAction:    "resume",
			BatchStopAction:     "pause",
		},
		Lifecycle: lifecycleOrchestrationModel{
			States: []string{
				string(domain.TaskQueued),
				string(domain.TaskLeased),
				string(domain.TaskRunning),
				string(domain.TaskBlocked),
				string(domain.TaskRetrying),
				string(domain.TaskSucceeded),
				string(domain.TaskFailed),
				string(domain.TaskCancelled),
				string(domain.TaskDeadLetter),
			},
			AutomaticTransitions: []lifecycleTransition{
				{From: []string{string(domain.TaskQueued)}, Action: string(domain.EventTaskLeased), To: string(domain.TaskLeased)},
				{From: []string{string(domain.TaskLeased)}, Action: string(domain.EventTaskStarted), To: string(domain.TaskRunning)},
				{From: []string{string(domain.TaskRunning)}, Action: string(domain.EventTaskCompleted), To: string(domain.TaskSucceeded)},
				{From: []string{string(domain.TaskRunning)}, Action: string(domain.EventTaskRetried), To: string(domain.TaskRetrying)},
				{From: []string{string(domain.TaskRetrying)}, Action: string(domain.EventTaskQueued), To: string(domain.TaskQueued)},
				{From: []string{string(domain.TaskQueued), string(domain.TaskLeased), string(domain.TaskRunning), string(domain.TaskRetrying)}, Action: string(domain.EventTaskCancelled), To: string(domain.TaskCancelled)},
				{From: []string{string(domain.TaskQueued), string(domain.TaskLeased), string(domain.TaskRunning), string(domain.TaskRetrying)}, Action: string(domain.EventTaskDeadLetter), To: string(domain.TaskDeadLetter)},
			},
			OperatorTransitions: []lifecycleTransition{
				{From: []string{string(domain.TaskQueued), string(domain.TaskLeased), string(domain.TaskRunning), string(domain.TaskRetrying)}, Action: "takeover", To: string(domain.TaskBlocked)},
				{From: []string{string(domain.TaskBlocked)}, Action: "release_takeover", To: string(domain.TaskQueued)},
				{From: []string{string(domain.TaskBlocked)}, Action: "annotate", To: string(domain.TaskBlocked)},
				{From: []string{string(domain.TaskBlocked)}, Action: "assign_owner", To: string(domain.TaskBlocked)},
				{From: []string{string(domain.TaskBlocked)}, Action: "assign_reviewer", To: string(domain.TaskBlocked)},
				{From: []string{string(domain.TaskQueued)}, Action: "cancel", To: string(domain.TaskCancelled)},
				{From: []string{string(domain.TaskFailed), string(domain.TaskCancelled), string(domain.TaskDeadLetter)}, Action: "retry", To: string(domain.TaskQueued)},
			},
			InterventionRules: []string{
				"Pause stops new batch dispatch without mutating queued task state.",
				"Takeover is the operator path for moving in-flight work into blocked review.",
				"Release takeover requeues blocked work instead of force-starting it.",
				"Batch stop can cancel queued work only when the queue backend exposes cancel support.",
			},
			TerminalStates: []string{
				string(domain.TaskSucceeded),
				string(domain.TaskFailed),
				string(domain.TaskCancelled),
				string(domain.TaskDeadLetter),
			},
		},
		BatchStartStrategy: lifecycleBatchStrategy{
			Intent:          "resume batched execution through normal scheduler dispatch",
			TriggerAction:   "resume",
			AppliesToStates: []string{string(domain.TaskQueued), string(domain.TaskRetrying), string(domain.TaskBlocked)},
			Sequence: []string{
				"Resume the global control gate to reopen scheduler dispatch.",
				"Let queued and retrying tasks drain in normal lease-to-run order.",
				"Release approved takeovers so blocked tasks return to queued before execution.",
			},
			Guardrails: []string{
				"Do not bypass blocked review with a force-start path.",
				"Keep batch restarts inside the advisory admission envelope rather than flooding the queue.",
			},
		},
		BatchStopStrategy: lifecycleBatchStrategy{
			Intent:          "stop new batch work first, then stabilize in-flight tasks for review",
			TriggerAction:   "pause",
			AppliesToStates: []string{string(domain.TaskQueued), string(domain.TaskLeased), string(domain.TaskRunning), string(domain.TaskRetrying)},
			Sequence: []string{
				"Pause the global control gate before issuing task-level interventions.",
				"Use takeover on in-flight tasks that need manual review or ownership assignment.",
				"Cancel queued tasks only when queue cancellation is enabled; otherwise keep them paused in queue.",
			},
			Guardrails: []string{
				"Pause first so batch stop does not race with fresh leases.",
				"Treat takeover and cancel as separate operator decisions instead of a single blanket kill switch.",
			},
		},
	}
}
