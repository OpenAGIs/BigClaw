package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	externalStoreReplayTaskID         = "external-store-smoke-task"
	externalStoreReplayTraceID        = "external-store-smoke-trace"
	externalStoreRetentionTaskID      = "external-store-retention-task"
	externalStoreRetentionTraceID     = "external-store-retention-trace"
	externalStoreCheckpointSubscriber = "subscriber-external-store"
	externalStoreLeaseGroupID         = "group-external-store"
	externalStoreLeaseSubscriberID    = "subscriber-external-store"
)

type automationExternalStoreValidationOptions struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	PollInterval   time.Duration
	Retention      string
	HTTPClient     *http.Client
	Now            func() time.Time
	Sleep          func(time.Duration)
}

type externalStoreHTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *externalStoreHTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

func runAutomationExternalStoreValidationCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e external-store-validation", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	reportPath := flags.String("report-path", "docs/reports/external-store-validation-report.json", "report path")
	timeoutSeconds := flags.Int("timeout-seconds", 120, "task timeout seconds")
	pollInterval := flags.Duration("poll-interval", 500*time.Millisecond, "task poll interval")
	retention := flags.String("retention", "2s", "event retention window")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e external-store-validation [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report, exitCode, err := automationExternalStoreValidation(automationExternalStoreValidationOptions{
		GoRoot:         absPath(*goRoot),
		ReportPath:     *reportPath,
		TimeoutSeconds: *timeoutSeconds,
		PollInterval:   *pollInterval,
		Retention:      *retention,
		HTTPClient:     http.DefaultClient,
		Now:            func() time.Time { return time.Now().UTC() },
		Sleep:          time.Sleep,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, exitCode)
}

func automationExternalStoreValidation(opts automationExternalStoreValidationOptions) (map[string]any, int, error) {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	runtimeRoot, err := os.MkdirTemp("", "bigclaw-external-store-")
	if err != nil {
		return nil, 0, err
	}
	serviceState := filepath.Join(runtimeRoot, "service")
	nodeAState := filepath.Join(runtimeRoot, "node-a")
	nodeBState := filepath.Join(runtimeRoot, "node-b")
	for _, path := range []string{serviceState, nodeAState, nodeBState} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, 0, err
		}
	}
	sharedLeaseDB := filepath.Join(runtimeRoot, "shared-subscriber-leases.db")

	serviceEnv, serviceBase, err := externalStoreBuildNodeEnv(serviceState, "file", "bigclawd-external-store-service", filepath.Join(serviceState, "event-log.db"), "", sharedLeaseDB, opts.Retention)
	if err != nil {
		return nil, 0, err
	}
	serviceProcess, serviceLogFile, _, err := externalStoreStartBigClawd(opts.GoRoot, serviceEnv, "external-store-service")
	if err != nil {
		return nil, 0, err
	}
	defer externalStoreTerminateProcess(serviceProcess, serviceLogFile)
	if err := automationWaitForHealth(client, serviceBase, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}
	remoteEventLogURL := serviceBase + "/internal/events/log"

	nodeAEnv, nodeABase, err := externalStoreBuildNodeEnv(nodeAState, "sqlite", "bigclawd-external-store-node-a", "", remoteEventLogURL, sharedLeaseDB, "")
	if err != nil {
		return nil, 0, err
	}
	nodeBEnv, nodeBBase, err := externalStoreBuildNodeEnv(nodeBState, "sqlite", "bigclawd-external-store-node-b", "", remoteEventLogURL, sharedLeaseDB, "")
	if err != nil {
		return nil, 0, err
	}
	nodeAProcess, nodeALogFile, _, err := externalStoreStartBigClawd(opts.GoRoot, nodeAEnv, "external-store-node-a")
	if err != nil {
		return nil, 0, err
	}
	defer externalStoreTerminateProcess(nodeAProcess, nodeALogFile)
	nodeBProcess, nodeBLogFile, _, err := externalStoreStartBigClawd(opts.GoRoot, nodeBEnv, "external-store-node-b")
	if err != nil {
		return nil, 0, err
	}
	defer externalStoreTerminateProcess(nodeBProcess, nodeBLogFile)
	if err := automationWaitForHealth(client, nodeABase, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}
	if err := automationWaitForHealth(client, nodeBBase, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}

	task := map[string]any{
		"id":                        externalStoreReplayTaskID,
		"trace_id":                  externalStoreReplayTraceID,
		"title":                     "External-store remote event-log smoke",
		"required_executor":         "local",
		"entrypoint":                "echo hello from remote event log",
		"execution_timeout_seconds": opts.TimeoutSeconds,
		"metadata": map[string]any{
			"scenario": "external-store-validation",
			"lane":     "remote_http_event_log",
		},
	}
	var submitted struct {
		Task map[string]any `json:"task"`
	}
	if err := externalStoreRequestJSON(client, http.MethodPost, nodeABase+"/tasks", task, &submitted); err != nil {
		return nil, 0, err
	}
	finalStatus, err := externalStoreWaitForTask(client, nodeABase, asString(submitted.Task["id"]), time.Duration(opts.TimeoutSeconds)*time.Second, opts.PollInterval, sleep)
	if err != nil {
		return nil, 0, err
	}
	replayPayload, err := externalStoreFetchJSON(client, nodeABase+"/events?task_id="+asString(submitted.Task["id"])+"&limit=100")
	if err != nil {
		return nil, 0, err
	}
	replayEvents := externalStoreMapSlice(replayPayload["events"])
	if err := externalStoreCheck(finalStatus["state"] == "succeeded", fmt.Sprintf("task did not succeed: %+v", finalStatus)); err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(asString(replayPayload["backend"]) == "http", fmt.Sprintf("expected remote replay backend http, got %+v", replayPayload["backend"])); err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(replayPayload["durable"] == true, fmt.Sprintf("expected durable replay payload, got %+v", replayPayload)); err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(len(replayEvents) >= 3, fmt.Sprintf("expected replay events for smoke task, got %+v", replayEvents)); err != nil {
		return nil, 0, err
	}
	latestEventID := asString(lastMapAny(replayEvents)["id"])

	checkpointPayload, err := externalStoreFetchJSONWithBody(client, http.MethodPost, nodeABase+"/stream/events/checkpoints/"+externalStoreCheckpointSubscriber, map[string]any{"event_id": latestEventID})
	if err != nil {
		return nil, 0, err
	}
	checkpointRead, err := externalStoreFetchJSON(client, nodeABase+"/stream/events/checkpoints/"+externalStoreCheckpointSubscriber)
	if err != nil {
		return nil, 0, err
	}
	if err := externalStoreRequestJSON(client, http.MethodDelete, nodeABase+"/stream/events/checkpoints/"+externalStoreCheckpointSubscriber, nil, nil); err != nil {
		return nil, 0, err
	}
	checkpointHistory, err := externalStoreFetchJSON(client, nodeABase+"/stream/events/checkpoints/"+externalStoreCheckpointSubscriber+"/history?limit=10")
	if err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(asString(lookupMap(checkpointPayload, "checkpoint", "event_id")) == latestEventID, fmt.Sprintf("expected checkpoint event %s, got %+v", latestEventID, checkpointPayload)); err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(asString(lookupMap(checkpointRead, "checkpoint", "event_id")) == latestEventID, fmt.Sprintf("expected checkpoint readback %s, got %+v", latestEventID, checkpointRead)); err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(lenMapSlice(checkpointHistory["history"]) >= 1, fmt.Sprintf("expected checkpoint reset history, got %+v", checkpointHistory)); err != nil {
		return nil, 0, err
	}

	retentionNow := now()
	if err := externalStoreRequestJSON(client, http.MethodPost, remoteEventLogURL+"/record", map[string]any{
		"id":        "evt-external-retention-old",
		"type":      "task.queued",
		"task_id":   externalStoreRetentionTaskID,
		"trace_id":  externalStoreRetentionTraceID,
		"timestamp": utcISOTime(retentionNow.Add(-10 * time.Second)),
	}, nil); err != nil {
		return nil, 0, err
	}
	if err := externalStoreRequestJSON(client, http.MethodPost, remoteEventLogURL+"/record", map[string]any{
		"id":        "evt-external-retention-new",
		"type":      "task.started",
		"task_id":   externalStoreRetentionTaskID,
		"trace_id":  externalStoreRetentionTraceID,
		"timestamp": utcISOTime(retentionNow),
	}, nil); err != nil {
		return nil, 0, err
	}
	retentionPayload, err := externalStoreFetchJSON(client, nodeABase+"/events?trace_id="+externalStoreRetentionTraceID+"&limit=10")
	if err != nil {
		return nil, 0, err
	}
	retentionEvents := externalStoreMapSlice(retentionPayload["events"])
	retentionWatermark, _ := retentionPayload["retention_watermark"].(map[string]any)
	if err := externalStoreCheck(len(retentionEvents) == 1 && asString(retentionEvents[0]["id"]) == "evt-external-retention-new", fmt.Sprintf("expected only retained external-store event, got %+v", retentionEvents)); err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(retentionWatermark["history_truncated"] == true, fmt.Sprintf("expected history truncation, got %+v", retentionWatermark)); err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(retentionWatermark["persisted_boundary"] == true, fmt.Sprintf("expected persisted boundary, got %+v", retentionWatermark)); err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(asString(retentionWatermark["trimmed_through_event_id"]) == "evt-external-retention-old", fmt.Sprintf("unexpected trimmed boundary: %+v", retentionWatermark)); err != nil {
		return nil, 0, err
	}

	leaseA, err := externalStoreFetchJSONWithBody(client, http.MethodPost, nodeABase+"/subscriber-groups/leases", map[string]any{
		"group_id":      externalStoreLeaseGroupID,
		"subscriber_id": externalStoreLeaseSubscriberID,
		"consumer_id":   "node-a",
		"ttl_seconds":   2,
	})
	if err != nil {
		return nil, 0, err
	}
	checkpointA, err := externalStoreFetchJSONWithBody(client, http.MethodPost, nodeABase+"/subscriber-groups/checkpoints", map[string]any{
		"group_id":            externalStoreLeaseGroupID,
		"subscriber_id":       externalStoreLeaseSubscriberID,
		"consumer_id":         "node-a",
		"lease_token":         lookupMap(leaseA, "lease", "lease_token"),
		"lease_epoch":         lookupMap(leaseA, "lease", "lease_epoch"),
		"checkpoint_offset":   11,
		"checkpoint_event_id": latestEventID,
	})
	if err != nil {
		return nil, 0, err
	}
	conflictStatus := 0
	conflictBody := ""
	err = externalStoreRequestJSON(client, http.MethodPost, nodeBBase+"/subscriber-groups/leases", map[string]any{
		"group_id":      externalStoreLeaseGroupID,
		"subscriber_id": externalStoreLeaseSubscriberID,
		"consumer_id":   "node-b",
		"ttl_seconds":   2,
	}, nil)
	var statusErr *externalStoreHTTPStatusError
	if errors.As(err, &statusErr) {
		conflictStatus = statusErr.StatusCode
		conflictBody = statusErr.Body
		err = nil
	}
	if err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(conflictStatus == 409, fmt.Sprintf("expected active leader conflict 409, got %d %s", conflictStatus, conflictBody)); err != nil {
		return nil, 0, err
	}
	sleep(2200 * time.Millisecond)
	leaseB, err := externalStoreFetchJSONWithBody(client, http.MethodPost, nodeBBase+"/subscriber-groups/leases", map[string]any{
		"group_id":      externalStoreLeaseGroupID,
		"subscriber_id": externalStoreLeaseSubscriberID,
		"consumer_id":   "node-b",
		"ttl_seconds":   2,
	})
	if err != nil {
		return nil, 0, err
	}
	staleStatus := 0
	staleBody := ""
	err = externalStoreRequestJSON(client, http.MethodPost, nodeABase+"/subscriber-groups/checkpoints", map[string]any{
		"group_id":            externalStoreLeaseGroupID,
		"subscriber_id":       externalStoreLeaseSubscriberID,
		"consumer_id":         "node-a",
		"lease_token":         lookupMap(leaseA, "lease", "lease_token"),
		"lease_epoch":         lookupMap(leaseA, "lease", "lease_epoch"),
		"checkpoint_offset":   12,
		"checkpoint_event_id": latestEventID,
	}, nil)
	if errors.As(err, &statusErr) {
		staleStatus = statusErr.StatusCode
		staleBody = statusErr.Body
		err = nil
	}
	if err != nil {
		return nil, 0, err
	}
	if err := externalStoreCheck(staleStatus == 409, fmt.Sprintf("expected stale writer conflict 409, got %d %s", staleStatus, staleBody)); err != nil {
		return nil, 0, err
	}
	checkpointB, err := externalStoreFetchJSONWithBody(client, http.MethodPost, nodeBBase+"/subscriber-groups/checkpoints", map[string]any{
		"group_id":            externalStoreLeaseGroupID,
		"subscriber_id":       externalStoreLeaseSubscriberID,
		"consumer_id":         "node-b",
		"lease_token":         lookupMap(leaseB, "lease", "lease_token"),
		"lease_epoch":         lookupMap(leaseB, "lease", "lease_epoch"),
		"checkpoint_offset":   15,
		"checkpoint_event_id": latestEventID,
	})
	if err != nil {
		return nil, 0, err
	}
	leaseStatus, err := externalStoreFetchJSON(client, nodeBBase+"/subscriber-groups/"+externalStoreLeaseGroupID+"/subscribers/"+externalStoreLeaseSubscriberID)
	if err != nil {
		return nil, 0, err
	}

	report := externalStoreBuildReport(now(), opts.Retention, asString(submitted.Task["id"]), asString(submitted.Task["trace_id"]), finalStatus, replayPayload, replayEvents, latestEventID, checkpointPayload, checkpointRead, checkpointHistory, retentionPayload, retentionEvents, retentionWatermark, leaseA, checkpointA, conflictStatus, leaseB, staleStatus, checkpointB, leaseStatus)
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	return report, 0, nil
}

func externalStoreBuildReport(generatedAt time.Time, retention, taskID, traceID string, finalStatus, replayPayload map[string]any, replayEvents []map[string]any, latestEventID string, checkpointPayload, checkpointRead, checkpointHistory, retentionPayload map[string]any, retentionEvents []map[string]any, retentionWatermark, leaseA, checkpointA map[string]any, conflictStatus int, leaseB map[string]any, staleStatus int, checkpointB, leaseStatus map[string]any) map[string]any {
	return map[string]any{
		"generated_at": utcISOTime(generatedAt),
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
			"task_succeeded":             finalStatus["state"] == "succeeded",
			"remote_replay_backend":      replayPayload["backend"],
			"replay_event_count":         len(replayEvents),
			"checkpoint_acknowledged":    asString(lookupMap(checkpointPayload, "checkpoint", "event_id")) == latestEventID,
			"checkpoint_reset_recorded":  lenMapSlice(checkpointHistory["history"]) >= 1,
			"retention_boundary_visible": retentionWatermark["history_truncated"] == true,
			"retained_event_count":       len(retentionEvents),
			"takeover_conflict_rejected": conflictStatus == 409,
			"takeover_after_expiry":      asInt(lookupMap(leaseB, "lease", "lease_epoch")) == 2,
			"stale_writer_rejected":      staleStatus == 409,
		},
		"backend_matrix": externalStoreBuildBackendMatrix(asString(replayPayload["backend"]), retentionWatermark["history_truncated"] == true),
		"replay_validation": map[string]any{
			"task_id":           taskID,
			"trace_id":          traceID,
			"backend":           replayPayload["backend"],
			"durable":           replayPayload["durable"],
			"latest_event_id":   latestEventID,
			"latest_event_type": asString(lastMapAny(replayEvents)["type"]),
		},
		"checkpoint_validation": map[string]any{
			"subscriber_id":         externalStoreCheckpointSubscriber,
			"acked_event_id":        lookupMap(checkpointPayload, "checkpoint", "event_id"),
			"checkpoint_event_id":   lookupMap(checkpointRead, "checkpoint", "event_id"),
			"reset_history_entries": lenMapSlice(checkpointHistory["history"]),
		},
		"retention_validation": map[string]any{
			"trace_id":                 externalStoreRetentionTraceID,
			"history_truncated":        retentionWatermark["history_truncated"],
			"persisted_boundary":       retentionWatermark["persisted_boundary"],
			"trimmed_through_event_id": retentionWatermark["trimmed_through_event_id"],
			"oldest_event_id":          retentionWatermark["oldest_event_id"],
			"newest_event_id":          retentionWatermark["newest_event_id"],
		},
		"takeover_validation": map[string]any{
			"group_id":                  externalStoreLeaseGroupID,
			"subscriber_id":             externalStoreLeaseSubscriberID,
			"initial_consumer":          lookupMap(leaseA, "lease", "consumer_id"),
			"initial_epoch":             lookupMap(leaseA, "lease", "lease_epoch"),
			"initial_checkpoint_offset": lookupMap(checkpointA, "lease", "checkpoint_offset"),
			"conflict_status":           conflictStatus,
			"takeover_consumer":         lookupMap(leaseB, "lease", "consumer_id"),
			"takeover_epoch":            lookupMap(leaseB, "lease", "lease_epoch"),
			"stale_writer_status":       staleStatus,
			"final_checkpoint_offset":   lookupMap(checkpointB, "lease", "checkpoint_offset"),
			"final_lease_consumer":      lookupMap(leaseStatus, "lease", "consumer_id"),
		},
		"artifacts": map[string]any{
			"e2e_doc":          "docs/e2e-validation.md",
			"retention_report": "docs/reports/replay-retention-semantics-report.md",
			"epic_report":      "docs/reports/epic-closure-readiness-report.md",
		},
		"limitations": []any{
			"The backend matrix marks the HTTP remote-service lane as live validated, while broker-backed and quorum-backed durability remain explicit placeholders.",
			"Event replay and checkpoint storage are validated through the remote HTTP event-log service, while shared-queue coordination and takeover still rely on the current shared SQLite lease store.",
		},
	}
}

func externalStoreBuildBackendMatrix(replayBackend string, retentionBoundaryVisible bool) map[string]any {
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
		"lanes": []any{
			map[string]any{
				"backend":                    "http_remote_service",
				"role":                       "runtime_event_log",
				"validation_status":          "live_validated",
				"configuration_state":        "configured",
				"proof_kind":                 "repo_native_e2e",
				"replay_backend":             replayBackend,
				"checkpoint_backend":         "http_remote_service",
				"retention_boundary_visible": retentionBoundaryVisible,
				"takeover_backend":           "sqlite_shared_lease",
				"report_links": []any{
					"docs/e2e-validation.md",
					"docs/reports/replay-retention-semantics-report.md",
					"docs/reports/epic-closure-readiness-report.md",
				},
				"notes": "Replay, checkpoint state, and retention-boundary visibility are validated through the remote HTTP event-log service boundary.",
			},
			map[string]any{
				"backend":             "broker_replicated",
				"role":                "runtime_event_log",
				"validation_status":   "not_configured",
				"configuration_state": "not_configured",
				"proof_kind":          "placeholder",
				"reason":              "not_configured",
				"report_links": []any{
					"docs/reports/broker-failover-fault-injection-validation-pack.md",
					"docs/reports/broker-failover-stub-report.json",
				},
				"notes": "The checked-in repo-native external-store lane does not start a live broker-backed event-log adapter yet.",
			},
			map[string]any{
				"backend":             "quorum_replicated",
				"role":                "runtime_event_log",
				"validation_status":   "contract_only",
				"configuration_state": "contract_documented",
				"proof_kind":          "placeholder",
				"reason":              "contract_only",
				"report_links": []any{
					"docs/reports/replicated-event-log-durability-rollout-contract.md",
					"docs/reports/replicated-broker-durability-rollout-spike.md",
				},
				"notes": "Quorum-backed durability expectations are documented, but no executable quorum lane or adapter is checked in.",
			},
		},
	}
}

func externalStoreBuildNodeEnv(stateDir, queueBackend, serviceName, eventLogSQLitePath, eventLogRemoteURL, subscriberLeaseSQLitePath, eventRetention string) (map[string]string, string, error) {
	env := map[string]string{}
	for _, item := range os.Environ() {
		parts := bytes.SplitN([]byte(item), []byte("="), 2)
		if len(parts) == 2 {
			env[string(parts[0])] = string(parts[1])
		}
	}
	env["BIGCLAW_QUEUE_BACKEND"] = queueBackend
	switch queueBackend {
	case "sqlite":
		env["BIGCLAW_QUEUE_SQLITE_PATH"] = filepath.Join(stateDir, "queue.db")
	case "file":
		env["BIGCLAW_QUEUE_FILE"] = filepath.Join(stateDir, "queue.json")
	}
	env["BIGCLAW_AUDIT_LOG_PATH"] = filepath.Join(stateDir, "audit.jsonl")
	env["BIGCLAW_SERVICE_NAME"] = serviceName
	env["BIGCLAW_BOOTSTRAP_TASKS"] = "0"
	env["BIGCLAW_MAX_CONCURRENT_RUNS"] = "2"
	if trim(eventLogSQLitePath) != "" {
		env["BIGCLAW_EVENT_LOG_SQLITE_PATH"] = eventLogSQLitePath
	} else {
		delete(env, "BIGCLAW_EVENT_LOG_SQLITE_PATH")
	}
	if trim(eventLogRemoteURL) != "" {
		env["BIGCLAW_EVENT_LOG_REMOTE_URL"] = eventLogRemoteURL
	} else {
		delete(env, "BIGCLAW_EVENT_LOG_REMOTE_URL")
	}
	if trim(subscriberLeaseSQLitePath) != "" {
		env["BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH"] = subscriberLeaseSQLitePath
	} else {
		delete(env, "BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH")
	}
	if trim(eventRetention) != "" {
		env["BIGCLAW_EVENT_RETENTION"] = eventRetention
	} else {
		delete(env, "BIGCLAW_EVENT_RETENTION")
	}
	baseURL, httpAddr, err := externalStoreReserveLocalBaseURL()
	if err != nil {
		return nil, "", err
	}
	env["BIGCLAW_HTTP_ADDR"] = httpAddr
	return env, baseURL, nil
}

func externalStoreReserveLocalBaseURL() (string, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return "", "", err
	}
	return "http://" + addr, addr, nil
}

func externalStoreStartBigClawd(goRoot string, env map[string]string, logName string) (*exec.Cmd, *os.File, string, error) {
	logPath := filepath.Join(goRoot, "docs", "reports", logName+".log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, nil, "", err
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, nil, "", err
	}
	command := exec.Command("go", "run", "./cmd/bigclawd")
	command.Dir = goRoot
	command.Stdout = logFile
	command.Stderr = logFile
	command.Env = externalStoreMapToEnv(env)
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return nil, nil, "", err
	}
	return command, logFile, logPath, nil
}

func externalStoreTerminateProcess(process *exec.Cmd, logFile *os.File) {
	if process == nil {
		return
	}
	if process.Process != nil {
		_ = process.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() {
			_, _ = process.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = process.Process.Kill()
		}
	}
	if logFile != nil {
		_ = logFile.Close()
	}
}

func externalStoreRequestJSON(client *http.Client, method, url string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		raw, _ := io.ReadAll(response.Body)
		return &externalStoreHTTPStatusError{StatusCode: response.StatusCode, Body: trim(string(raw))}
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func externalStoreFetchJSON(client *http.Client, url string) (map[string]any, error) {
	return externalStoreFetchJSONWithBody(client, http.MethodGet, url, nil)
}

func externalStoreFetchJSONWithBody(client *http.Client, method, url string, payload any) (map[string]any, error) {
	result := map[string]any{}
	if err := externalStoreRequestJSON(client, method, url, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func externalStoreWaitForTask(client *http.Client, baseURL, taskID string, timeout, pollInterval time.Duration, sleep func(time.Duration)) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := externalStoreFetchJSON(client, baseURL+"/tasks/"+taskID)
		if err != nil {
			return nil, err
		}
		if automationIsTerminal(status["state"]) {
			return status, nil
		}
		sleep(pollInterval)
	}
	return nil, fmt.Errorf("task %s did not reach terminal state before timeout", taskID)
}

func externalStoreCheck(condition bool, message string) error {
	if !condition {
		return errors.New(message)
	}
	return nil
}

func externalStoreMapSlice(value any) []map[string]any {
	items, _ := value.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry, _ := item.(map[string]any)
		out = append(out, entry)
	}
	return out
}

func externalStoreMapToEnv(values map[string]string) []string {
	env := make([]string, 0, len(values))
	for key, value := range values {
		env = append(env, key+"="+value)
	}
	return env
}

func lastMapAny(items []map[string]any) map[string]any {
	if len(items) == 0 {
		return map[string]any{}
	}
	return items[len(items)-1]
}
