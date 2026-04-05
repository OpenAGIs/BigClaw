package reporting

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ExternalStoreValidationGenerator = "bigclaw-go/scripts/e2e/external_store_validation/main.go"

	externalStoreReplayTaskID          = "external-store-smoke-task"
	externalStoreReplayTraceID         = "external-store-smoke-trace"
	externalStoreRetentionTaskID       = "external-store-retention-task"
	externalStoreRetentionTraceID      = "external-store-retention-trace"
	externalStoreCheckpointSubscriber  = "subscriber-external-store"
	externalStoreLeaseGroupID          = "group-external-store"
	externalStoreLeaseSubscriberID     = "subscriber-external-store"
	defaultExternalStoreValidationPath = "bigclaw-go/docs/reports/external-store-validation-report.json"
)

type ExternalStoreValidationOptions struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	PollInterval   time.Duration
	Retention      string
	TimeNow        func() time.Time
}

type externalStoreHTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e externalStoreHTTPStatusError) Error() string {
	return fmt.Sprintf("http %d: %s", e.StatusCode, e.Body)
}

type externalStoreProcess struct {
	cmd     *exec.Cmd
	logFile *os.File
	logPath string
}

type externalStoreRuntimeNode struct {
	baseURL string
	env     map[string]string
}

type externalStoreRuntimeArtifacts struct {
	replayPayload     map[string]any
	replayEvents      []map[string]any
	checkpointWrite   map[string]any
	checkpointRead    map[string]any
	checkpointHistory map[string]any
	retentionPayload  map[string]any
	retentionEvents   []map[string]any
	retentionMark     map[string]any
	submittedTask     map[string]any
	finalStatus       map[string]any
	leaseA            map[string]any
	checkpointA       map[string]any
	leaseB            map[string]any
	checkpointB       map[string]any
	leaseStatus       map[string]any
	conflictStatus    int
	staleStatus       int
}

func RunExternalStoreValidation(options ExternalStoreValidationOptions) (map[string]any, error) {
	if strings.TrimSpace(options.GoRoot) == "" {
		root, err := FindRepoRoot(".")
		if err != nil {
			return nil, err
		}
		options.GoRoot = root
	}
	if strings.TrimSpace(options.ReportPath) == "" {
		options.ReportPath = defaultExternalStoreValidationPath
	}
	if options.TimeoutSeconds <= 0 {
		options.TimeoutSeconds = 120
	}
	if options.PollInterval <= 0 {
		options.PollInterval = 500 * time.Millisecond
	}
	if strings.TrimSpace(options.Retention) == "" {
		options.Retention = "2s"
	}
	if options.TimeNow == nil {
		options.TimeNow = time.Now
	}

	artifacts, err := runExternalStoreRuntime(options)
	if err != nil {
		return nil, err
	}
	report := buildExternalStoreValidationReport(options.TimeNow(), options.Retention, artifacts)
	if err := WriteJSON(resolveReportPath(options.GoRoot, options.ReportPath), report); err != nil {
		return nil, err
	}
	return report, nil
}

func runExternalStoreRuntime(options ExternalStoreValidationOptions) (externalStoreRuntimeArtifacts, error) {
	runtimeRoot, err := os.MkdirTemp("", "bigclaw-external-store-")
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	serviceState := filepath.Join(runtimeRoot, "service")
	nodeAState := filepath.Join(runtimeRoot, "node-a")
	nodeBState := filepath.Join(runtimeRoot, "node-b")
	for _, path := range []string{serviceState, nodeAState, nodeBState} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return externalStoreRuntimeArtifacts{}, err
		}
	}
	sharedLeaseDB := filepath.Join(runtimeRoot, "shared-subscriber-leases.db")

	serviceNode, err := externalStoreBuildNodeEnv(serviceState, "file", "bigclawd-external-store-service", filepath.Join(serviceState, "event-log.db"), "", sharedLeaseDB, options.Retention, false)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	serviceProcess, err := startExternalStoreBigclawd(options.GoRoot, serviceNode.env, "external-store-service")
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	defer stopTaskSmokeProcess(serviceProcess.cmd, serviceProcess.logFile)
	if err := waitForTaskSmokeHealth(serviceNode.baseURL, 60, time.Second); err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	remoteEventLogURL := serviceNode.baseURL + "/internal/events/log"

	nodeA, err := externalStoreBuildNodeEnv(nodeAState, "sqlite", "bigclawd-external-store-node-a", "", remoteEventLogURL, sharedLeaseDB, "", true)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	nodeB, err := externalStoreBuildNodeEnv(nodeBState, "sqlite", "bigclawd-external-store-node-b", "", remoteEventLogURL, sharedLeaseDB, "", true)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	nodeAProcess, err := startExternalStoreBigclawd(options.GoRoot, nodeA.env, "external-store-node-a")
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	defer stopTaskSmokeProcess(nodeAProcess.cmd, nodeAProcess.logFile)
	nodeBProcess, err := startExternalStoreBigclawd(options.GoRoot, nodeB.env, "external-store-node-b")
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	defer stopTaskSmokeProcess(nodeBProcess.cmd, nodeBProcess.logFile)
	if err := waitForTaskSmokeHealth(nodeA.baseURL, 60, time.Second); err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	if err := waitForTaskSmokeHealth(nodeB.baseURL, 60, time.Second); err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}

	task := map[string]any{
		"id":                        externalStoreReplayTaskID,
		"trace_id":                  externalStoreReplayTraceID,
		"title":                     "External-store remote event-log smoke",
		"required_executor":         "local",
		"entrypoint":                "echo hello from remote event log",
		"execution_timeout_seconds": options.TimeoutSeconds,
		"metadata": map[string]any{
			"scenario": "external-store-validation",
			"lane":     "remote_http_event_log",
		},
	}
	submittedPayload, err := taskSmokeHTTPJSON(nodeA.baseURL+"/tasks", http.MethodPost, task, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	submittedTask := asMap(submittedPayload["task"])
	finalStatus, err := externalStoreWaitForTask(nodeA.baseURL, asString(submittedTask["id"]), options.TimeoutSeconds, options.PollInterval)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	replayPayload, err := taskSmokeHTTPJSON(nodeA.baseURL+"/events?task_id="+asString(submittedTask["id"])+"&limit=100", http.MethodGet, nil, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	replayEvents := anyToMapSlice(replayPayload["events"])
	if asString(finalStatus["state"]) != "succeeded" {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("task did not succeed: %+v", finalStatus)
	}
	if asString(replayPayload["backend"]) != "http" {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("expected remote replay backend http, got %q", asString(replayPayload["backend"]))
	}
	if !asBool(replayPayload["durable"]) {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("expected durable replay payload, got %+v", replayPayload)
	}
	if len(replayEvents) < 3 {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("expected replay events for smoke task, got %+v", replayEvents)
	}
	latestEventID := asString(replayEvents[len(replayEvents)-1]["id"])

	checkpointWrite, err := taskSmokeHTTPJSON(nodeA.baseURL+"/stream/events/checkpoints/"+externalStoreCheckpointSubscriber, http.MethodPost, map[string]any{"event_id": latestEventID}, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	checkpointRead, err := taskSmokeHTTPJSON(nodeA.baseURL+"/stream/events/checkpoints/"+externalStoreCheckpointSubscriber, http.MethodGet, nil, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	if _, err := externalStoreHTTPJSON(nodeA.baseURL+"/stream/events/checkpoints/"+externalStoreCheckpointSubscriber, http.MethodDelete, nil, 10*time.Second); err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	checkpointHistory, err := taskSmokeHTTPJSON(nodeA.baseURL+"/stream/events/checkpoints/"+externalStoreCheckpointSubscriber+"/history?limit=10", http.MethodGet, nil, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	if asString(asMap(checkpointWrite["checkpoint"])["event_id"]) != latestEventID || asString(asMap(checkpointRead["checkpoint"])["event_id"]) != latestEventID {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("checkpoint write/read mismatch: write=%+v read=%+v", checkpointWrite, checkpointRead)
	}
	if len(asSlice(checkpointHistory["history"])) < 1 {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("expected checkpoint reset history, got %+v", checkpointHistory)
	}

	retentionNow := options.TimeNow().UTC()
	if _, err := taskSmokeHTTPJSON(remoteEventLogURL+"/record", http.MethodPost, map[string]any{
		"id":        "evt-external-retention-old",
		"type":      "task.queued",
		"task_id":   externalStoreRetentionTaskID,
		"trace_id":  externalStoreRetentionTraceID,
		"timestamp": retentionNow.Add(-10 * time.Second).Format(time.RFC3339),
	}, 10*time.Second); err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	if _, err := taskSmokeHTTPJSON(remoteEventLogURL+"/record", http.MethodPost, map[string]any{
		"id":        "evt-external-retention-new",
		"type":      "task.started",
		"task_id":   externalStoreRetentionTaskID,
		"trace_id":  externalStoreRetentionTraceID,
		"timestamp": retentionNow.Format(time.RFC3339),
	}, 10*time.Second); err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	retentionPayload, err := taskSmokeHTTPJSON(nodeA.baseURL+"/events?trace_id="+externalStoreRetentionTraceID+"&limit=10", http.MethodGet, nil, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	retentionEvents := anyToMapSlice(retentionPayload["events"])
	retentionMark := asMap(retentionPayload["retention_watermark"])
	if len(retentionEvents) != 1 || asString(retentionEvents[0]["id"]) != "evt-external-retention-new" {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("expected only retained external-store event, got %+v", retentionEvents)
	}
	if !asBool(retentionMark["history_truncated"]) || !asBool(retentionMark["persisted_boundary"]) || asString(retentionMark["trimmed_through_event_id"]) != "evt-external-retention-old" {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("unexpected retention watermark: %+v", retentionMark)
	}

	leaseA, err := taskSmokeHTTPJSON(nodeA.baseURL+"/subscriber-groups/leases", http.MethodPost, map[string]any{
		"group_id":      externalStoreLeaseGroupID,
		"subscriber_id": externalStoreLeaseSubscriberID,
		"consumer_id":   "node-a",
		"ttl_seconds":   2,
	}, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	checkpointA, err := taskSmokeHTTPJSON(nodeA.baseURL+"/subscriber-groups/checkpoints", http.MethodPost, map[string]any{
		"group_id":            externalStoreLeaseGroupID,
		"subscriber_id":       externalStoreLeaseSubscriberID,
		"consumer_id":         "node-a",
		"lease_token":         asString(asMap(leaseA["lease"])["lease_token"]),
		"lease_epoch":         asInt(asMap(leaseA["lease"])["lease_epoch"]),
		"checkpoint_offset":   11,
		"checkpoint_event_id": latestEventID,
	}, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}

	conflictStatus := 0
	if _, err := externalStoreHTTPJSON(nodeB.baseURL+"/subscriber-groups/leases", http.MethodPost, map[string]any{
		"group_id":      externalStoreLeaseGroupID,
		"subscriber_id": externalStoreLeaseSubscriberID,
		"consumer_id":   "node-b",
		"ttl_seconds":   2,
	}, 10*time.Second); err != nil {
		var statusErr externalStoreHTTPStatusError
		if errors.As(err, &statusErr) {
			conflictStatus = statusErr.StatusCode
		} else {
			return externalStoreRuntimeArtifacts{}, err
		}
	}
	if conflictStatus != 409 {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("expected active leader conflict 409, got %d", conflictStatus)
	}

	time.Sleep(2200 * time.Millisecond)
	leaseB, err := taskSmokeHTTPJSON(nodeB.baseURL+"/subscriber-groups/leases", http.MethodPost, map[string]any{
		"group_id":      externalStoreLeaseGroupID,
		"subscriber_id": externalStoreLeaseSubscriberID,
		"consumer_id":   "node-b",
		"ttl_seconds":   2,
	}, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}

	staleStatus := 0
	if _, err := externalStoreHTTPJSON(nodeA.baseURL+"/subscriber-groups/checkpoints", http.MethodPost, map[string]any{
		"group_id":            externalStoreLeaseGroupID,
		"subscriber_id":       externalStoreLeaseSubscriberID,
		"consumer_id":         "node-a",
		"lease_token":         asString(asMap(leaseA["lease"])["lease_token"]),
		"lease_epoch":         asInt(asMap(leaseA["lease"])["lease_epoch"]),
		"checkpoint_offset":   12,
		"checkpoint_event_id": latestEventID,
	}, 10*time.Second); err != nil {
		var statusErr externalStoreHTTPStatusError
		if errors.As(err, &statusErr) {
			staleStatus = statusErr.StatusCode
		} else {
			return externalStoreRuntimeArtifacts{}, err
		}
	}
	if staleStatus != 409 {
		return externalStoreRuntimeArtifacts{}, fmt.Errorf("expected stale writer conflict 409, got %d", staleStatus)
	}

	checkpointB, err := taskSmokeHTTPJSON(nodeB.baseURL+"/subscriber-groups/checkpoints", http.MethodPost, map[string]any{
		"group_id":            externalStoreLeaseGroupID,
		"subscriber_id":       externalStoreLeaseSubscriberID,
		"consumer_id":         "node-b",
		"lease_token":         asString(asMap(leaseB["lease"])["lease_token"]),
		"lease_epoch":         asInt(asMap(leaseB["lease"])["lease_epoch"]),
		"checkpoint_offset":   15,
		"checkpoint_event_id": latestEventID,
	}, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}
	leaseStatus, err := taskSmokeHTTPJSON(nodeB.baseURL+"/subscriber-groups/"+externalStoreLeaseGroupID+"/subscribers/"+externalStoreLeaseSubscriberID, http.MethodGet, nil, 10*time.Second)
	if err != nil {
		return externalStoreRuntimeArtifacts{}, err
	}

	return externalStoreRuntimeArtifacts{
		replayPayload:     replayPayload,
		replayEvents:      replayEvents,
		checkpointWrite:   checkpointWrite,
		checkpointRead:    checkpointRead,
		checkpointHistory: checkpointHistory,
		retentionPayload:  retentionPayload,
		retentionEvents:   retentionEvents,
		retentionMark:     retentionMark,
		submittedTask:     submittedTask,
		finalStatus:       finalStatus,
		leaseA:            leaseA,
		checkpointA:       checkpointA,
		leaseB:            leaseB,
		checkpointB:       checkpointB,
		leaseStatus:       leaseStatus,
		conflictStatus:    conflictStatus,
		staleStatus:       staleStatus,
	}, nil
}

func buildExternalStoreValidationReport(now time.Time, retention string, artifacts externalStoreRuntimeArtifacts) map[string]any {
	if now.IsZero() {
		now = time.Now()
	}
	replayEvents := artifacts.replayEvents
	retentionEvents := artifacts.retentionEvents
	replayPayload := artifacts.replayPayload
	retentionMark := artifacts.retentionMark
	latestReplayEvent := map[string]any{}
	if len(replayEvents) > 0 {
		latestReplayEvent = replayEvents[len(replayEvents)-1]
	}
	return map[string]any{
		"generated_at": now.UTC().Format(time.RFC3339Nano),
		"ticket":       "BIG-PAR-102",
		"title":        "External-store validation backend matrix and broker placeholders",
		"status":       "validated",
		"lane": map[string]any{
			"service_backend":           "sqlite_event_log_service",
			"runtime_event_log_backend": "http_remote_service",
			"queue_backend":             "sqlite",
			"subscriber_lease_backend":  "sqlite_shared",
			"retention":                 retention,
			"node_count":                3,
		},
		"summary": map[string]any{
			"task_succeeded":             asString(artifacts.finalStatus["state"]) == "succeeded",
			"remote_replay_backend":      asString(replayPayload["backend"]),
			"replay_event_count":         len(replayEvents),
			"checkpoint_acknowledged":    asString(asMap(artifacts.checkpointWrite["checkpoint"])["event_id"]) == asString(latestReplayEvent["id"]),
			"checkpoint_reset_recorded":  len(asSlice(artifacts.checkpointHistory["history"])) >= 1,
			"retention_boundary_visible": asBool(retentionMark["history_truncated"]),
			"retained_event_count":       len(retentionEvents),
			"takeover_conflict_rejected": artifacts.conflictStatus == 409,
			"takeover_after_expiry":      asInt(asMap(artifacts.leaseB["lease"])["lease_epoch"]) == 2,
			"stale_writer_rejected":      artifacts.staleStatus == 409,
		},
		"backend_matrix": buildExternalStoreBackendMatrix(asString(replayPayload["backend"]), asBool(retentionMark["history_truncated"])),
		"replay_validation": map[string]any{
			"task_id":           asString(artifacts.submittedTask["id"]),
			"trace_id":          asString(artifacts.submittedTask["trace_id"]),
			"backend":           asString(replayPayload["backend"]),
			"durable":           asBool(replayPayload["durable"]),
			"latest_event_id":   asString(latestReplayEvent["id"]),
			"latest_event_type": asString(latestReplayEvent["type"]),
		},
		"checkpoint_validation": map[string]any{
			"subscriber_id":         externalStoreCheckpointSubscriber,
			"acked_event_id":        asString(asMap(artifacts.checkpointWrite["checkpoint"])["event_id"]),
			"checkpoint_event_id":   asString(asMap(artifacts.checkpointRead["checkpoint"])["event_id"]),
			"reset_history_entries": len(asSlice(artifacts.checkpointHistory["history"])),
		},
		"retention_validation": map[string]any{
			"trace_id":                 externalStoreRetentionTraceID,
			"history_truncated":        asBool(retentionMark["history_truncated"]),
			"persisted_boundary":       asBool(retentionMark["persisted_boundary"]),
			"trimmed_through_event_id": asString(retentionMark["trimmed_through_event_id"]),
			"oldest_event_id":          asString(retentionMark["oldest_event_id"]),
			"newest_event_id":          asString(retentionMark["newest_event_id"]),
		},
		"takeover_validation": map[string]any{
			"group_id":                  externalStoreLeaseGroupID,
			"subscriber_id":             externalStoreLeaseSubscriberID,
			"initial_consumer":          asString(asMap(artifacts.leaseA["lease"])["consumer_id"]),
			"initial_epoch":             asInt(asMap(artifacts.leaseA["lease"])["lease_epoch"]),
			"initial_checkpoint_offset": asInt(asMap(artifacts.checkpointA["lease"])["checkpoint_offset"]),
			"conflict_status":           artifacts.conflictStatus,
			"takeover_consumer":         asString(asMap(artifacts.leaseB["lease"])["consumer_id"]),
			"takeover_epoch":            asInt(asMap(artifacts.leaseB["lease"])["lease_epoch"]),
			"stale_writer_status":       artifacts.staleStatus,
			"final_checkpoint_offset":   asInt(asMap(artifacts.checkpointB["lease"])["checkpoint_offset"]),
			"final_lease_consumer":      asString(asMap(artifacts.leaseStatus["lease"])["consumer_id"]),
		},
		"artifacts": map[string]any{
			"e2e_doc":          "docs/e2e-validation.md",
			"retention_report": "docs/reports/replay-retention-semantics-report.md",
			"epic_report":      "docs/reports/epic-closure-readiness-report.md",
		},
		"limitations": []string{
			"The backend matrix marks the HTTP remote-service lane as live validated, while broker-backed and quorum-backed durability remain explicit placeholders.",
			"Event replay and checkpoint storage are validated through the remote HTTP event-log service, while shared-queue coordination and takeover still rely on the current shared SQLite lease store.",
		},
	}
}

func buildExternalStoreBackendMatrix(replayBackend string, retentionBoundaryVisible bool) map[string]any {
	return map[string]any{
		"status_definitions": map[string]any{
			"live_validated": "This checked-in repo-native lane executed the backend path and captured evidence.",
			"not_configured": "The backend lane is intentionally not configured in the current runtime proof and remains a placeholder.",
			"contract_only":  "Only the rollout contract defines the expected backend semantics today.",
		},
		"summary": map[string]any{
			"live_validated_lanes": 1,
			"not_configured_lanes": 1,
			"contract_only_lanes":  1,
		},
		"lanes": []map[string]any{
			{
				"backend":                    "http_remote_service",
				"role":                       "runtime_event_log",
				"validation_status":          "live_validated",
				"configuration_state":        "configured",
				"proof_kind":                 "repo_native_e2e",
				"replay_backend":             replayBackend,
				"checkpoint_backend":         "http_remote_service",
				"retention_boundary_visible": retentionBoundaryVisible,
				"takeover_backend":           "sqlite_shared_lease",
				"report_links": []string{
					"docs/e2e-validation.md",
					"docs/reports/replay-retention-semantics-report.md",
					"docs/reports/epic-closure-readiness-report.md",
				},
				"notes": "Replay, checkpoint state, and retention-boundary visibility are validated through the remote HTTP event-log service boundary.",
			},
			{
				"backend":             "broker_replicated",
				"role":                "runtime_event_log",
				"validation_status":   "not_configured",
				"configuration_state": "not_configured",
				"proof_kind":          "placeholder",
				"reason":              "not_configured",
				"report_links": []string{
					"docs/reports/broker-failover-fault-injection-validation-pack.md",
					"docs/reports/broker-failover-stub-report.json",
				},
				"notes": "The checked-in repo-native external-store lane does not start a live broker-backed event-log adapter yet.",
			},
			{
				"backend":             "quorum_replicated",
				"role":                "runtime_event_log",
				"validation_status":   "contract_only",
				"configuration_state": "contract_documented",
				"proof_kind":          "placeholder",
				"reason":              "contract_only",
				"report_links": []string{
					"docs/reports/replicated-event-log-durability-rollout-contract.md",
					"docs/reports/replicated-broker-durability-rollout-spike.md",
				},
				"notes": "Quorum-backed durability expectations are documented, but no executable quorum lane or adapter is checked in.",
			},
		},
	}
}

func externalStoreBuildNodeEnv(stateDir string, queueBackend string, serviceName string, eventLogSQLitePath string, eventLogRemoteURL string, subscriberLeaseSQLitePath string, eventRetention string, clearEventLogSQLite bool) (externalStoreRuntimeNode, error) {
	env := environmentMap(os.Environ())
	env["BIGCLAW_QUEUE_BACKEND"] = queueBackend
	switch queueBackend {
	case "sqlite":
		env["BIGCLAW_QUEUE_SQLITE_PATH"] = filepath.Join(stateDir, "queue.db")
		delete(env, "BIGCLAW_QUEUE_FILE")
	case "file":
		env["BIGCLAW_QUEUE_FILE"] = filepath.Join(stateDir, "queue.json")
		delete(env, "BIGCLAW_QUEUE_SQLITE_PATH")
	default:
		return externalStoreRuntimeNode{}, fmt.Errorf("unsupported queue backend %q", queueBackend)
	}
	env["BIGCLAW_AUDIT_LOG_PATH"] = filepath.Join(stateDir, "audit.jsonl")
	env["BIGCLAW_SERVICE_NAME"] = serviceName
	env["BIGCLAW_BOOTSTRAP_TASKS"] = "0"
	env["BIGCLAW_MAX_CONCURRENT_RUNS"] = "2"
	if strings.TrimSpace(eventLogSQLitePath) != "" {
		env["BIGCLAW_EVENT_LOG_SQLITE_PATH"] = eventLogSQLitePath
	} else if clearEventLogSQLite {
		delete(env, "BIGCLAW_EVENT_LOG_SQLITE_PATH")
	}
	if strings.TrimSpace(eventLogRemoteURL) != "" {
		env["BIGCLAW_EVENT_LOG_REMOTE_URL"] = eventLogRemoteURL
	} else {
		delete(env, "BIGCLAW_EVENT_LOG_REMOTE_URL")
	}
	if strings.TrimSpace(subscriberLeaseSQLitePath) != "" {
		env["BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH"] = subscriberLeaseSQLitePath
	}
	if strings.TrimSpace(eventRetention) != "" {
		env["BIGCLAW_EVENT_RETENTION"] = eventRetention
	} else {
		delete(env, "BIGCLAW_EVENT_RETENTION")
	}
	baseURL, httpAddr, err := reserveTaskSmokeLocalBaseURL()
	if err != nil {
		return externalStoreRuntimeNode{}, err
	}
	env["BIGCLAW_HTTP_ADDR"] = httpAddr
	return externalStoreRuntimeNode{
		baseURL: baseURL,
		env:     env,
	}, nil
}

func startExternalStoreBigclawd(goRoot string, env map[string]string, logName string) (externalStoreProcess, error) {
	logPath := filepath.Join(goRoot, "docs", "reports", logName+".log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return externalStoreProcess{}, err
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return externalStoreProcess{}, err
	}
	cmd := exec.Command("go", "run", "./cmd/bigclawd")
	cmd.Dir = goRoot
	cmd.Env = environmentSlice(env)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return externalStoreProcess{}, err
	}
	return externalStoreProcess{cmd: cmd, logFile: logFile, logPath: logPath}, nil
}

func externalStoreWaitForTask(baseURL string, taskID string, timeoutSeconds int, pollInterval time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	for time.Now().Before(deadline) {
		status, err := fetchTaskSmokeStatus(baseURL, taskID)
		if err != nil {
			return nil, err
		}
		if taskSmokeTerminal(asString(status["state"])) {
			return status, nil
		}
		time.Sleep(pollInterval)
	}
	return nil, fmt.Errorf("task %s did not reach terminal state before timeout", taskID)
}

func externalStoreHTTPJSON(url string, method string, payload any, timeout time.Duration) (map[string]any, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, externalStoreHTTPStatusError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(raw))}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
