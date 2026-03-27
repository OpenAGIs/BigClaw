package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const (
	externalStoreReplayTaskID         = "external-store-smoke-task"
	externalStoreReplayTraceID        = "external-store-smoke-trace"
	externalStoreRetentionTaskID      = "external-store-retention-task"
	externalStoreRetentionTraceID     = "external-store-retention-trace"
	externalStoreCheckpointSubscriber = "subscriber-external-store"
	externalStoreLeaseGroup           = "group-external-store"
	externalStoreLeaseSubscriber      = "subscriber-external-store"
)

type automationExternalStoreValidationOptions struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	PollInterval   time.Duration
	Retention      time.Duration
	HTTPClient     *http.Client
	Sleep          func(time.Duration)
	Now            func() time.Time
}

type externalStoreNodeRuntime struct {
	BaseURL  string
	LogPath  string
	StateDir string
	Command  *exec.Cmd
	logFile  *os.File
}

func runAutomationExternalStoreValidationCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e external-store-validation", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	reportPath := flags.String("report-path", "docs/reports/external-store-validation-report.json", "report path")
	timeoutSeconds := flags.Int("timeout-seconds", 120, "task timeout seconds")
	pollInterval := flags.Duration("poll-interval", 500*time.Millisecond, "task poll interval")
	retention := flags.Duration("retention", 2*time.Second, "event retention duration")
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
		ReportPath:     trim(*reportPath),
		TimeoutSeconds: *timeoutSeconds,
		PollInterval:   *pollInterval,
		Retention:      *retention,
	})
	if report != nil {
		return emit(report, *asJSON, exitCode)
	}
	return err
}

func automationExternalStoreValidation(opts automationExternalStoreValidationOptions) (map[string]any, int, error) {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	runtimeRoot, err := os.MkdirTemp("", "bigclaw-external-store-")
	if err != nil {
		return nil, 0, err
	}
	defer os.RemoveAll(runtimeRoot)

	serviceState := filepath.Join(runtimeRoot, "service")
	nodeAState := filepath.Join(runtimeRoot, "node-a")
	nodeBState := filepath.Join(runtimeRoot, "node-b")
	for _, path := range []string{serviceState, nodeAState, nodeBState} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, 0, err
		}
	}
	sharedLeaseDB := filepath.Join(runtimeRoot, "shared-subscriber-leases.db")

	var serviceNode, nodeA, nodeB *externalStoreNodeRuntime
	defer stopExternalStoreRuntime(nodeB)
	defer stopExternalStoreRuntime(nodeA)
	defer stopExternalStoreRuntime(serviceNode)

	serviceNode, err = startExternalStoreNode(opts.GoRoot, "external-store-service", map[string]string{
		"BIGCLAW_QUEUE_BACKEND":         "file",
		"BIGCLAW_QUEUE_FILE":            filepath.Join(serviceState, "queue.json"),
		"BIGCLAW_AUDIT_LOG_PATH":        filepath.Join(serviceState, "audit.jsonl"),
		"BIGCLAW_SERVICE_NAME":          "bigclawd-external-store-service",
		"BIGCLAW_BOOTSTRAP_TASKS":       "0",
		"BIGCLAW_MAX_CONCURRENT_RUNS":   "2",
		"BIGCLAW_EVENT_LOG_SQLITE_PATH": filepath.Join(serviceState, "event-log.db"),
		"BIGCLAW_EVENT_RETENTION":       durationString(opts.Retention),
	})
	if err != nil {
		return nil, 0, err
	}
	if err := automationWaitForHealth(client, serviceNode.BaseURL, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}
	remoteEventLogURL := serviceNode.BaseURL + "/internal/events/log"

	nodeA, err = startExternalStoreNode(opts.GoRoot, "external-store-node-a", map[string]string{
		"BIGCLAW_QUEUE_BACKEND":                "sqlite",
		"BIGCLAW_QUEUE_SQLITE_PATH":            filepath.Join(nodeAState, "queue.db"),
		"BIGCLAW_AUDIT_LOG_PATH":               filepath.Join(nodeAState, "audit.jsonl"),
		"BIGCLAW_SERVICE_NAME":                 "bigclawd-external-store-node-a",
		"BIGCLAW_BOOTSTRAP_TASKS":              "0",
		"BIGCLAW_MAX_CONCURRENT_RUNS":          "2",
		"BIGCLAW_EVENT_LOG_REMOTE_URL":         remoteEventLogURL,
		"BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH": sharedLeaseDB,
	})
	if err != nil {
		return nil, 0, err
	}
	nodeB, err = startExternalStoreNode(opts.GoRoot, "external-store-node-b", map[string]string{
		"BIGCLAW_QUEUE_BACKEND":                "sqlite",
		"BIGCLAW_QUEUE_SQLITE_PATH":            filepath.Join(nodeBState, "queue.db"),
		"BIGCLAW_AUDIT_LOG_PATH":               filepath.Join(nodeBState, "audit.jsonl"),
		"BIGCLAW_SERVICE_NAME":                 "bigclawd-external-store-node-b",
		"BIGCLAW_BOOTSTRAP_TASKS":              "0",
		"BIGCLAW_MAX_CONCURRENT_RUNS":          "2",
		"BIGCLAW_EVENT_LOG_REMOTE_URL":         remoteEventLogURL,
		"BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH": sharedLeaseDB,
	})
	if err != nil {
		return nil, 0, err
	}
	if err := automationWaitForHealth(client, nodeA.BaseURL, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}
	if err := automationWaitForHealth(client, nodeB.BaseURL, 60, time.Second, sleep); err != nil {
		return nil, 0, err
	}

	task := automationTask{
		ID:                      externalStoreReplayTaskID,
		TraceID:                 externalStoreReplayTraceID,
		Title:                   "External-store remote event-log smoke",
		RequiredExecutor:        "local",
		Entrypoint:              "echo hello from remote event log",
		ExecutionTimeoutSeconds: opts.TimeoutSeconds,
		Metadata: map[string]string{
			"scenario": "external-store-validation",
			"lane":     "remote_http_event_log",
		},
	}
	submitted := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, nodeA.BaseURL, "/tasks", task, &submitted); err != nil {
		return nil, 0, err
	}
	submittedTask, _ := submitted["task"].(map[string]any)
	if submittedTask == nil {
		submittedTask = structToMap(task)
	}
	finalStatus, err := automationWaitForTask(client, nodeA.BaseURL, task.ID, time.Duration(opts.TimeoutSeconds)*time.Second, sleep)
	if err != nil {
		return nil, 0, err
	}
	replayPayload := map[string]any{}
	if err := automationRequestJSON(client, http.MethodGet, nodeA.BaseURL, "/events?task_id="+task.ID+"&limit=100", nil, &replayPayload); err != nil {
		return nil, 0, err
	}
	replayEvents := anySliceToMaps(replayPayload["events"])
	if fmt.Sprint(finalStatus["state"]) != "succeeded" {
		return nil, 0, fmt.Errorf("task did not succeed: %+v", finalStatus)
	}
	if fmt.Sprint(replayPayload["backend"]) != "http" || replayPayload["durable"] != true || len(replayEvents) < 3 {
		return nil, 0, fmt.Errorf("unexpected replay payload: %+v", replayPayload)
	}
	latestEventID := fmt.Sprint(replayEvents[len(replayEvents)-1]["id"])

	checkpointPayload := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, nodeA.BaseURL, "/stream/events/checkpoints/"+externalStoreCheckpointSubscriber, map[string]any{"event_id": latestEventID}, &checkpointPayload); err != nil {
		return nil, 0, err
	}
	checkpointRead := map[string]any{}
	if err := automationRequestJSON(client, http.MethodGet, nodeA.BaseURL, "/stream/events/checkpoints/"+externalStoreCheckpointSubscriber, nil, &checkpointRead); err != nil {
		return nil, 0, err
	}
	if err := externalStoreRequestJSON(client, http.MethodDelete, nodeA.BaseURL, "/stream/events/checkpoints/"+externalStoreCheckpointSubscriber, nil, nil); err != nil {
		return nil, 0, err
	}
	checkpointHistory := map[string]any{}
	if err := automationRequestJSON(client, http.MethodGet, nodeA.BaseURL, "/stream/events/checkpoints/"+externalStoreCheckpointSubscriber+"/history?limit=10", nil, &checkpointHistory); err != nil {
		return nil, 0, err
	}
	if eventIDFromCheckpoint(checkpointPayload) != latestEventID || eventIDFromCheckpoint(checkpointRead) != latestEventID {
		return nil, 0, fmt.Errorf("checkpoint event mismatch: payload=%+v read=%+v", checkpointPayload, checkpointRead)
	}
	if len(anySlice(checkpointHistory["history"])) < 1 {
		return nil, 0, fmt.Errorf("expected checkpoint history: %+v", checkpointHistory)
	}

	retentionNow := now().UTC()
	for _, payload := range []map[string]any{
		{
			"id":        "evt-external-retention-old",
			"type":      "task.queued",
			"task_id":   externalStoreRetentionTaskID,
			"trace_id":  externalStoreRetentionTraceID,
			"timestamp": retentionNow.Add(-10 * time.Second).Format(time.RFC3339),
		},
		{
			"id":        "evt-external-retention-new",
			"type":      "task.started",
			"task_id":   externalStoreRetentionTaskID,
			"trace_id":  externalStoreRetentionTraceID,
			"timestamp": retentionNow.Format(time.RFC3339),
		},
	} {
		if err := automationRequestJSON(client, http.MethodPost, remoteEventLogURL, "/record", payload, nil); err != nil {
			return nil, 0, err
		}
	}
	retentionPayload := map[string]any{}
	if err := automationRequestJSON(client, http.MethodGet, nodeA.BaseURL, "/events?trace_id="+externalStoreRetentionTraceID+"&limit=10", nil, &retentionPayload); err != nil {
		return nil, 0, err
	}
	retentionEvents := anySliceToMaps(retentionPayload["events"])
	retentionWatermark, _ := retentionPayload["retention_watermark"].(map[string]any)
	if len(retentionEvents) != 1 || fmt.Sprint(retentionEvents[0]["id"]) != "evt-external-retention-new" {
		return nil, 0, fmt.Errorf("unexpected retention events: %+v", retentionEvents)
	}
	if retentionWatermark["history_truncated"] != true || retentionWatermark["persisted_boundary"] != true || fmt.Sprint(retentionWatermark["trimmed_through_event_id"]) != "evt-external-retention-old" {
		return nil, 0, fmt.Errorf("unexpected retention watermark: %+v", retentionWatermark)
	}

	leaseA := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, nodeA.BaseURL, "/subscriber-groups/leases", map[string]any{
		"group_id":      externalStoreLeaseGroup,
		"subscriber_id": externalStoreLeaseSubscriber,
		"consumer_id":   "node-a",
		"ttl_seconds":   2,
	}, &leaseA); err != nil {
		return nil, 0, err
	}
	leaseAMap, _ := leaseA["lease"].(map[string]any)
	checkpointA := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, nodeA.BaseURL, "/subscriber-groups/checkpoints", map[string]any{
		"group_id":            externalStoreLeaseGroup,
		"subscriber_id":       externalStoreLeaseSubscriber,
		"consumer_id":         "node-a",
		"lease_token":         leaseAMap["lease_token"],
		"lease_epoch":         leaseAMap["lease_epoch"],
		"checkpoint_offset":   11,
		"checkpoint_event_id": latestEventID,
	}, &checkpointA); err != nil {
		return nil, 0, err
	}
	conflictStatus, conflictBody, err := externalStoreStatusOnly(client, http.MethodPost, nodeB.BaseURL, "/subscriber-groups/leases", map[string]any{
		"group_id":      externalStoreLeaseGroup,
		"subscriber_id": externalStoreLeaseSubscriber,
		"consumer_id":   "node-b",
		"ttl_seconds":   2,
	})
	if err != nil {
		return nil, 0, err
	}
	if conflictStatus != http.StatusConflict {
		return nil, 0, fmt.Errorf("expected active leader conflict 409, got %d %s", conflictStatus, conflictBody)
	}
	sleep(2200 * time.Millisecond)
	leaseB := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, nodeB.BaseURL, "/subscriber-groups/leases", map[string]any{
		"group_id":      externalStoreLeaseGroup,
		"subscriber_id": externalStoreLeaseSubscriber,
		"consumer_id":   "node-b",
		"ttl_seconds":   2,
	}, &leaseB); err != nil {
		return nil, 0, err
	}
	leaseBMap, _ := leaseB["lease"].(map[string]any)
	staleStatus, staleBody, err := externalStoreStatusOnly(client, http.MethodPost, nodeA.BaseURL, "/subscriber-groups/checkpoints", map[string]any{
		"group_id":            externalStoreLeaseGroup,
		"subscriber_id":       externalStoreLeaseSubscriber,
		"consumer_id":         "node-a",
		"lease_token":         leaseAMap["lease_token"],
		"lease_epoch":         leaseAMap["lease_epoch"],
		"checkpoint_offset":   12,
		"checkpoint_event_id": latestEventID,
	})
	if err != nil {
		return nil, 0, err
	}
	if staleStatus != http.StatusConflict {
		return nil, 0, fmt.Errorf("expected stale writer conflict 409, got %d %s", staleStatus, staleBody)
	}
	checkpointB := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, nodeB.BaseURL, "/subscriber-groups/checkpoints", map[string]any{
		"group_id":            externalStoreLeaseGroup,
		"subscriber_id":       externalStoreLeaseSubscriber,
		"consumer_id":         "node-b",
		"lease_token":         leaseBMap["lease_token"],
		"lease_epoch":         leaseBMap["lease_epoch"],
		"checkpoint_offset":   15,
		"checkpoint_event_id": latestEventID,
	}, &checkpointB); err != nil {
		return nil, 0, err
	}
	leaseStatus := map[string]any{}
	if err := automationRequestJSON(client, http.MethodGet, nodeB.BaseURL, "/subscriber-groups/"+externalStoreLeaseGroup+"/subscribers/"+externalStoreLeaseSubscriber, nil, &leaseStatus); err != nil {
		return nil, 0, err
	}
	leaseStatusMap, _ := leaseStatus["lease"].(map[string]any)

	report := map[string]any{
		"generated_at": now().UTC().Format(time.RFC3339),
		"ticket":       "BIG-PAR-102",
		"title":        "External-store validation backend matrix and broker placeholders",
		"status":       "validated",
		"lane": map[string]any{
			"service_backend":           "sqlite_event_log_service",
			"runtime_event_log_backend": "http_remote_service",
			"queue_backend":             "sqlite",
			"subscriber_lease_backend":  "sqlite_shared",
			"retention":                 durationString(opts.Retention),
			"node_count":                3,
		},
		"summary": map[string]any{
			"task_succeeded":             fmt.Sprint(finalStatus["state"]) == "succeeded",
			"remote_replay_backend":      replayPayload["backend"],
			"replay_event_count":         len(replayEvents),
			"checkpoint_acknowledged":    eventIDFromCheckpoint(checkpointPayload) == latestEventID,
			"checkpoint_reset_recorded":  len(anySlice(checkpointHistory["history"])) >= 1,
			"retention_boundary_visible": retentionWatermark["history_truncated"] == true,
			"retained_event_count":       len(retentionEvents),
			"takeover_conflict_rejected": conflictStatus == http.StatusConflict,
			"takeover_after_expiry":      intValue(leaseBMap["lease_epoch"]) == 2,
			"stale_writer_rejected":      staleStatus == http.StatusConflict,
		},
		"backend_matrix": buildExternalStoreBackendMatrix(fmt.Sprint(replayPayload["backend"]), retentionWatermark["history_truncated"] == true),
		"replay_validation": map[string]any{
			"task_id":           submittedTask["id"],
			"trace_id":          submittedTask["trace_id"],
			"backend":           replayPayload["backend"],
			"durable":           replayPayload["durable"],
			"latest_event_id":   latestEventID,
			"latest_event_type": replayEvents[len(replayEvents)-1]["type"],
		},
		"checkpoint_validation": map[string]any{
			"subscriber_id":         externalStoreCheckpointSubscriber,
			"acked_event_id":        eventIDFromCheckpoint(checkpointPayload),
			"checkpoint_event_id":   eventIDFromCheckpoint(checkpointRead),
			"reset_history_entries": len(anySlice(checkpointHistory["history"])),
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
			"group_id":                  externalStoreLeaseGroup,
			"subscriber_id":             externalStoreLeaseSubscriber,
			"initial_consumer":          leaseAMap["consumer_id"],
			"initial_epoch":             leaseAMap["lease_epoch"],
			"initial_checkpoint_offset": nestedInt(checkpointA, "lease", "checkpoint_offset"),
			"conflict_status":           conflictStatus,
			"takeover_consumer":         leaseBMap["consumer_id"],
			"takeover_epoch":            leaseBMap["lease_epoch"],
			"stale_writer_status":       staleStatus,
			"final_checkpoint_offset":   nestedInt(checkpointB, "lease", "checkpoint_offset"),
			"final_lease_consumer":      leaseStatusMap["consumer_id"],
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
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	return report, 0, nil
}

func startExternalStoreNode(goRoot string, logName string, extraEnv map[string]string) (*externalStoreNodeRuntime, error) {
	baseURL, httpAddr, err := automationReserveLocalBaseURL()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(goRoot, "docs", "reports", logName+".log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}
	env := os.Environ()
	env = append(env, "BIGCLAW_HTTP_ADDR="+httpAddr)
	for key, value := range extraEnv {
		env = append(env, key+"="+value)
	}
	cmd := exec.Command("go", "run", "./cmd/bigclawd")
	cmd.Dir = goRoot
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	return &externalStoreNodeRuntime{
		BaseURL: baseURL,
		LogPath: logPath,
		Command: cmd,
		logFile: logFile,
	}, nil
}

func stopExternalStoreRuntime(node *externalStoreNodeRuntime) {
	if node == nil {
		return
	}
	if node.Command != nil && node.Command.Process != nil {
		_ = node.Command.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() {
			_, _ = node.Command.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = node.Command.Process.Kill()
		}
	}
	if node.logFile != nil {
		_ = node.logFile.Close()
	}
}

func externalStoreRequestJSON(client *http.Client, method string, baseURL string, path string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, trim(baseURL)+path, body)
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
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("http %d: %s", response.StatusCode, trim(string(body)))
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func externalStoreStatusOnly(client *http.Client, method string, baseURL string, path string, payload any) (int, string, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, "", err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, trim(baseURL)+path, body)
	if err != nil {
		return 0, "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	return response.StatusCode, trim(string(responseBody)), nil
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

func eventIDFromCheckpoint(payload map[string]any) string {
	checkpoint, _ := payload["checkpoint"].(map[string]any)
	return fmt.Sprint(checkpoint["event_id"])
}

func nestedInt(payload map[string]any, keys ...string) int {
	current := any(payload)
	for _, key := range keys {
		entry, _ := current.(map[string]any)
		current = entry[key]
	}
	return intValue(current)
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

func anySlice(value any) []any {
	items, _ := value.([]any)
	return items
}

func anySliceToMaps(value any) []map[string]any {
	items := anySlice(value)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry, _ := item.(map[string]any)
		result = append(result, entry)
	}
	return result
}

func durationString(value time.Duration) string {
	if value%time.Second == 0 {
		return strconv.Itoa(int(value/time.Second)) + "s"
	}
	return value.String()
}
