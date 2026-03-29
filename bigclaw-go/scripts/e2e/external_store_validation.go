package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	replayTaskID           = "external-store-smoke-task"
	replayTraceID          = "external-store-smoke-trace"
	retentionTaskID        = "external-store-retention-task"
	retentionTraceID       = "external-store-retention-trace"
	checkpointSubscriberID = "subscriber-external-store"
	leaseGroupID           = "group-external-store"
	leaseSubscriberID      = "subscriber-external-store"
)

type externalStoreArgs struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	PollInterval   time.Duration
	Retention      string
}

type externalNodeConfig struct {
	Process   *exec.Cmd
	LogHandle *os.File
	LogPath   string
	BaseURL   string
}

type externalHTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *externalHTTPStatusError) Error() string {
	return fmt.Sprintf("http %d: %s", e.StatusCode, e.Body)
}

func main() {
	args := parseExternalStoreArgs()
	os.Exit(runExternalStoreValidation(args))
}

func parseExternalStoreArgs() externalStoreArgs {
	defaultRoot, err := os.Getwd()
	if err != nil {
		defaultRoot = "."
	}
	args := externalStoreArgs{}
	flag.StringVar(&args.GoRoot, "go-root", defaultRoot, "repo root")
	flag.StringVar(&args.ReportPath, "report-path", "docs/reports/external-store-validation-report.json", "report output")
	flag.IntVar(&args.TimeoutSeconds, "timeout-seconds", 120, "timeout in seconds")
	pollInterval := flag.Float64("poll-interval", 0.5, "poll interval in seconds")
	flag.StringVar(&args.Retention, "retention", "2s", "event retention duration")
	flag.Parse()
	args.PollInterval = time.Duration(*pollInterval * float64(time.Second))
	return args
}

func runExternalStoreValidation(args externalStoreArgs) int {
	goRoot, err := filepath.Abs(args.GoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	runtimeRoot, err := os.MkdirTemp("", "bigclaw-external-store-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	serviceState := filepath.Join(runtimeRoot, "service")
	nodeAState := filepath.Join(runtimeRoot, "node-a")
	nodeBState := filepath.Join(runtimeRoot, "node-b")
	for _, path := range []string{serviceState, nodeAState, nodeBState} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	sharedLeaseDB := filepath.Join(runtimeRoot, "shared-subscriber-leases.db")

	serviceEnv, serviceBase, err := buildExternalNodeEnv(serviceState, "file", "bigclawd-external-store-service", filepath.Join(serviceState, "event-log.db"), "", "", args.Retention)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	serviceNode, err := startExternalNode(goRoot, serviceEnv, serviceBase, "external-store-service")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer terminateExternalNode(serviceNode)
	if err := waitForExternalHealth(serviceNode.BaseURL, 60, time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	remoteEventLogURL := serviceNode.BaseURL + "/internal/events/log"

	nodeAEnv, nodeABase, err := buildExternalNodeEnv(nodeAState, "sqlite", "bigclawd-external-store-node-a", "", remoteEventLogURL, sharedLeaseDB, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	nodeANode, err := startExternalNode(goRoot, nodeAEnv, nodeABase, "external-store-node-a")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer terminateExternalNode(nodeANode)
	if err := waitForExternalHealth(nodeANode.BaseURL, 60, time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	nodeBEnv, nodeBBase, err := buildExternalNodeEnv(nodeBState, "sqlite", "bigclawd-external-store-node-b", "", remoteEventLogURL, sharedLeaseDB, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	nodeBNode, err := startExternalNode(goRoot, nodeBEnv, nodeBBase, "external-store-node-b")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer terminateExternalNode(nodeBNode)
	if err := waitForExternalHealth(nodeBNode.BaseURL, 60, time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	task := map[string]any{
		"id":                        replayTaskID,
		"trace_id":                  replayTraceID,
		"title":                     "External-store remote event-log smoke",
		"required_executor":         "local",
		"entrypoint":                "echo hello from remote event log",
		"execution_timeout_seconds": args.TimeoutSeconds,
		"metadata": map[string]any{
			"scenario": "external-store-validation",
			"lane":     "remote_http_event_log",
		},
	}
	submitted, err := submitExternalTask(nodeANode.BaseURL, task)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	finalStatus, err := waitForExternalTask(nodeANode.BaseURL, asString(submitted["id"]), args.TimeoutSeconds, args.PollInterval)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	replayPayload, err := externalHTTPJSON(nodeANode.BaseURL+"/events?task_id="+asString(submitted["id"])+"&limit=100", http.MethodGet, nil, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	replayEvents := asMapSlice(replayPayload["events"])
	if err := externalCheck(asString(finalStatus["state"]) == "succeeded", fmt.Sprintf("task did not succeed: %+v", finalStatus)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(asString(replayPayload["backend"]) == "http", fmt.Sprintf("expected remote replay backend http, got %v", replayPayload["backend"])); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(asBool(replayPayload["durable"]), fmt.Sprintf("expected durable replay payload, got %+v", replayPayload)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(len(replayEvents) >= 3, fmt.Sprintf("expected replay events for smoke task, got %+v", replayEvents)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	latestEventID := asString(replayEvents[len(replayEvents)-1]["id"])

	checkpointPayload, err := externalHTTPJSON(nodeANode.BaseURL+"/stream/events/checkpoints/"+checkpointSubscriberID, http.MethodPost, map[string]any{"event_id": latestEventID}, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	checkpointRead, err := externalHTTPJSON(nodeANode.BaseURL+"/stream/events/checkpoints/"+checkpointSubscriberID, http.MethodGet, nil, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := externalHTTPJSON(nodeANode.BaseURL+"/stream/events/checkpoints/"+checkpointSubscriberID, http.MethodDelete, nil, 30*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	checkpointHistory, err := externalHTTPJSON(nodeANode.BaseURL+"/stream/events/checkpoints/"+checkpointSubscriberID+"/history?limit=10", http.MethodGet, nil, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(asString(asMap(checkpointPayload["checkpoint"])["event_id"]) == latestEventID, fmt.Sprintf("expected checkpoint event %s, got %+v", latestEventID, checkpointPayload)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(asString(asMap(checkpointRead["checkpoint"])["event_id"]) == latestEventID, fmt.Sprintf("expected checkpoint readback %s, got %+v", latestEventID, checkpointRead)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(len(asSlice(checkpointHistory["history"])) >= 1, fmt.Sprintf("expected checkpoint reset history, got %+v", checkpointHistory)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	retentionNow := time.Now().UTC()
	if _, err := externalHTTPJSON(remoteEventLogURL+"/record", http.MethodPost, map[string]any{
		"id":        "evt-external-retention-old",
		"type":      "task.queued",
		"task_id":   retentionTaskID,
		"trace_id":  retentionTraceID,
		"timestamp": retentionNow.Add(-10 * time.Second).Format(time.RFC3339Nano),
	}, 30*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := externalHTTPJSON(remoteEventLogURL+"/record", http.MethodPost, map[string]any{
		"id":        "evt-external-retention-new",
		"type":      "task.started",
		"task_id":   retentionTaskID,
		"trace_id":  retentionTraceID,
		"timestamp": retentionNow.Format(time.RFC3339Nano),
	}, 30*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	retentionPayload, err := externalHTTPJSON(nodeANode.BaseURL+"/events?trace_id="+retentionTraceID+"&limit=10", http.MethodGet, nil, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	retentionEvents := asMapSlice(retentionPayload["events"])
	retentionWatermark := asMap(retentionPayload["retention_watermark"])
	if err := externalCheck(len(retentionEvents) == 1 && asString(retentionEvents[0]["id"]) == "evt-external-retention-new", fmt.Sprintf("expected only retained external-store event, got %+v", retentionEvents)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(asBool(retentionWatermark["history_truncated"]), fmt.Sprintf("expected history truncation, got %+v", retentionWatermark)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(asBool(retentionWatermark["persisted_boundary"]), fmt.Sprintf("expected persisted boundary, got %+v", retentionWatermark)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := externalCheck(asString(retentionWatermark["trimmed_through_event_id"]) == "evt-external-retention-old", fmt.Sprintf("unexpected trimmed boundary: %+v", retentionWatermark)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	leaseA, err := externalHTTPJSON(nodeANode.BaseURL+"/subscriber-groups/leases", http.MethodPost, map[string]any{
		"group_id":      leaseGroupID,
		"subscriber_id": leaseSubscriberID,
		"consumer_id":   "node-a",
		"ttl_seconds":   2,
	}, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	checkpointA, err := externalHTTPJSON(nodeANode.BaseURL+"/subscriber-groups/checkpoints", http.MethodPost, map[string]any{
		"group_id":            leaseGroupID,
		"subscriber_id":       leaseSubscriberID,
		"consumer_id":         "node-a",
		"lease_token":         asString(asMap(leaseA["lease"])["lease_token"]),
		"lease_epoch":         asInt(asMap(leaseA["lease"])["lease_epoch"]),
		"checkpoint_offset":   11,
		"checkpoint_event_id": latestEventID,
	}, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	conflictStatus, conflictBody := 0, ""
	if _, err := externalHTTPJSON(nodeBNode.BaseURL+"/subscriber-groups/leases", http.MethodPost, map[string]any{
		"group_id":      leaseGroupID,
		"subscriber_id": leaseSubscriberID,
		"consumer_id":   "node-b",
		"ttl_seconds":   2,
	}, 30*time.Second); err != nil {
		if statusErr, ok := err.(*externalHTTPStatusError); ok {
			conflictStatus = statusErr.StatusCode
			conflictBody = statusErr.Body
		} else {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if err := externalCheck(conflictStatus == 409, fmt.Sprintf("expected active leader conflict 409, got %d %s", conflictStatus, conflictBody)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	time.Sleep(2200 * time.Millisecond)
	leaseB, err := externalHTTPJSON(nodeBNode.BaseURL+"/subscriber-groups/leases", http.MethodPost, map[string]any{
		"group_id":      leaseGroupID,
		"subscriber_id": leaseSubscriberID,
		"consumer_id":   "node-b",
		"ttl_seconds":   2,
	}, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	staleStatus, staleBody := 0, ""
	if _, err := externalHTTPJSON(nodeANode.BaseURL+"/subscriber-groups/checkpoints", http.MethodPost, map[string]any{
		"group_id":            leaseGroupID,
		"subscriber_id":       leaseSubscriberID,
		"consumer_id":         "node-a",
		"lease_token":         asString(asMap(leaseA["lease"])["lease_token"]),
		"lease_epoch":         asInt(asMap(leaseA["lease"])["lease_epoch"]),
		"checkpoint_offset":   12,
		"checkpoint_event_id": latestEventID,
	}, 30*time.Second); err != nil {
		if statusErr, ok := err.(*externalHTTPStatusError); ok {
			staleStatus = statusErr.StatusCode
			staleBody = statusErr.Body
		} else {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if err := externalCheck(staleStatus == 409, fmt.Sprintf("expected stale writer conflict 409, got %d %s", staleStatus, staleBody)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	checkpointB, err := externalHTTPJSON(nodeBNode.BaseURL+"/subscriber-groups/checkpoints", http.MethodPost, map[string]any{
		"group_id":            leaseGroupID,
		"subscriber_id":       leaseSubscriberID,
		"consumer_id":         "node-b",
		"lease_token":         asString(asMap(leaseB["lease"])["lease_token"]),
		"lease_epoch":         asInt(asMap(leaseB["lease"])["lease_epoch"]),
		"checkpoint_offset":   15,
		"checkpoint_event_id": latestEventID,
	}, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	leaseStatus, err := externalHTTPJSON(nodeBNode.BaseURL+"/subscriber-groups/"+leaseGroupID+"/subscribers/"+leaseSubscriberID, http.MethodGet, nil, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	report := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339Nano),
		"ticket":       "BIG-PAR-102",
		"title":        "External-store validation backend matrix and broker placeholders",
		"status":       "validated",
		"lane": map[string]any{
			"service_backend":           "sqlite_event_log_service",
			"runtime_event_log_backend": "http_remote_service",
			"queue_backend":             "sqlite",
			"subscriber_lease_backend":  "sqlite_shared",
			"retention":                 args.Retention,
			"node_count":                3,
		},
		"summary": map[string]any{
			"task_succeeded":             asString(finalStatus["state"]) == "succeeded",
			"remote_replay_backend":      asString(replayPayload["backend"]),
			"replay_event_count":         len(replayEvents),
			"checkpoint_acknowledged":    asString(asMap(checkpointPayload["checkpoint"])["event_id"]) == latestEventID,
			"checkpoint_reset_recorded":  len(asSlice(checkpointHistory["history"])) >= 1,
			"retention_boundary_visible": asBool(retentionWatermark["history_truncated"]),
			"retained_event_count":       len(retentionEvents),
			"takeover_conflict_rejected": conflictStatus == 409,
			"takeover_after_expiry":      asInt(asMap(leaseB["lease"])["lease_epoch"]) == 2,
			"stale_writer_rejected":      staleStatus == 409,
		},
		"backend_matrix": buildExternalBackendMatrix(asString(replayPayload["backend"]), asBool(retentionWatermark["history_truncated"])),
		"replay_validation": map[string]any{
			"task_id":           asString(submitted["id"]),
			"trace_id":          asString(submitted["trace_id"]),
			"backend":           asString(replayPayload["backend"]),
			"durable":           asBool(replayPayload["durable"]),
			"latest_event_id":   latestEventID,
			"latest_event_type": asString(replayEvents[len(replayEvents)-1]["type"]),
		},
		"checkpoint_validation": map[string]any{
			"subscriber_id":         checkpointSubscriberID,
			"acked_event_id":        asString(asMap(checkpointPayload["checkpoint"])["event_id"]),
			"checkpoint_event_id":   asString(asMap(checkpointRead["checkpoint"])["event_id"]),
			"reset_history_entries": len(asSlice(checkpointHistory["history"])),
		},
		"retention_validation": map[string]any{
			"trace_id":                 retentionTraceID,
			"history_truncated":        asBool(retentionWatermark["history_truncated"]),
			"persisted_boundary":       asBool(retentionWatermark["persisted_boundary"]),
			"trimmed_through_event_id": asString(retentionWatermark["trimmed_through_event_id"]),
			"oldest_event_id":          asString(retentionWatermark["oldest_event_id"]),
			"newest_event_id":          asString(retentionWatermark["newest_event_id"]),
		},
		"takeover_validation": map[string]any{
			"group_id":                  leaseGroupID,
			"subscriber_id":             leaseSubscriberID,
			"initial_consumer":          asString(asMap(leaseA["lease"])["consumer_id"]),
			"initial_epoch":             asInt(asMap(leaseA["lease"])["lease_epoch"]),
			"initial_checkpoint_offset": asInt(asMap(checkpointA["lease"])["checkpoint_offset"]),
			"conflict_status":           conflictStatus,
			"takeover_consumer":         asString(asMap(leaseB["lease"])["consumer_id"]),
			"takeover_epoch":            asInt(asMap(leaseB["lease"])["lease_epoch"]),
			"stale_writer_status":       staleStatus,
			"final_checkpoint_offset":   asInt(asMap(checkpointB["lease"])["checkpoint_offset"]),
			"final_lease_consumer":      asString(asMap(leaseStatus["lease"])["consumer_id"]),
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
	if err := writeExternalReport(goRoot, args.ReportPath, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(string(encoded))
	return 0
}

func buildExternalNodeEnv(stateDir string, queueBackend string, serviceName string, eventLogSQLitePath string, eventLogRemoteURL string, subscriberLeaseSQLitePath string, eventRetention string) (map[string]string, string, error) {
	env := copyExternalEnv()
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
	if eventLogSQLitePath != "" {
		env["BIGCLAW_EVENT_LOG_SQLITE_PATH"] = eventLogSQLitePath
	} else {
		delete(env, "BIGCLAW_EVENT_LOG_SQLITE_PATH")
	}
	if eventLogRemoteURL != "" {
		env["BIGCLAW_EVENT_LOG_REMOTE_URL"] = eventLogRemoteURL
	} else {
		delete(env, "BIGCLAW_EVENT_LOG_REMOTE_URL")
	}
	if subscriberLeaseSQLitePath != "" {
		env["BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH"] = subscriberLeaseSQLitePath
	} else {
		delete(env, "BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH")
	}
	if eventRetention != "" {
		env["BIGCLAW_EVENT_RETENTION"] = eventRetention
	} else {
		delete(env, "BIGCLAW_EVENT_RETENTION")
	}
	baseURL, httpAddr, err := reserveExternalBaseURL()
	if err != nil {
		return nil, "", err
	}
	env["BIGCLAW_HTTP_ADDR"] = httpAddr
	return env, baseURL, nil
}

func copyExternalEnv() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

func reserveExternalBaseURL() (string, string, error) {
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

func startExternalNode(goRoot string, env map[string]string, baseURL string, logName string) (externalNodeConfig, error) {
	logHandle, err := os.CreateTemp("", logName+"-*.log")
	if err != nil {
		return externalNodeConfig{}, err
	}
	cmd := exec.Command("go", "run", "./cmd/bigclawd")
	cmd.Dir = goRoot
	cmd.Stdout = logHandle
	cmd.Stderr = logHandle
	cmd.Env = externalEnvSlice(env)
	if err := cmd.Start(); err != nil {
		_ = logHandle.Close()
		return externalNodeConfig{}, err
	}
	return externalNodeConfig{Process: cmd, LogHandle: logHandle, LogPath: logHandle.Name(), BaseURL: baseURL}, nil
}

func externalEnvSlice(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+env[key])
	}
	return values
}

func terminateExternalNode(node externalNodeConfig) {
	if node.Process == nil || node.Process.Process == nil {
		return
	}
	_ = node.Process.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() {
		_, _ = node.Process.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = node.Process.Process.Kill()
		<-done
	}
	if node.LogHandle != nil {
		_ = node.LogHandle.Close()
	}
}

func waitForExternalHealth(baseURL string, attempts int, interval time.Duration) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		payload, err := externalHTTPJSON(baseURL+"/healthz", http.MethodGet, nil, 30*time.Second)
		if err == nil && asBool(payload["ok"]) {
			return nil
		}
		lastErr = err
		time.Sleep(interval)
	}
	return fmt.Errorf("service did not become healthy: %v", lastErr)
}

func externalHTTPJSON(url string, method string, payload any, timeout time.Duration) (map[string]any, error) {
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
	} else {
		decoded = map[string]any{}
	}
	if resp.StatusCode >= 400 {
		return nil, &externalHTTPStatusError{StatusCode: resp.StatusCode, Body: string(raw)}
	}
	return decoded, nil
}

func submitExternalTask(baseURL string, task map[string]any) (map[string]any, error) {
	payload, err := externalHTTPJSON(baseURL+"/tasks", http.MethodPost, task, 30*time.Second)
	if err != nil {
		return nil, err
	}
	return asMap(payload["task"]), nil
}

func waitForExternalTask(baseURL string, taskID string, timeoutSeconds int, pollInterval time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	for time.Now().Before(deadline) {
		status, err := externalHTTPJSON(baseURL+"/tasks/"+taskID, http.MethodGet, nil, 30*time.Second)
		if err != nil {
			return nil, err
		}
		if isExternalTerminal(asString(status["state"])) {
			return status, nil
		}
		time.Sleep(pollInterval)
	}
	return nil, fmt.Errorf("task %s did not reach terminal state before timeout", taskID)
}

func isExternalTerminal(state string) bool {
	switch state {
	case "succeeded", "dead_letter", "cancelled", "failed":
		return true
	default:
		return false
	}
}

func externalCheck(condition bool, message string) error {
	if condition {
		return nil
	}
	return fmt.Errorf("%s", message)
}

func buildExternalBackendMatrix(replayBackend string, retentionBoundaryVisible bool) map[string]any {
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

func writeExternalReport(goRoot string, reportPath string, payload map[string]any) error {
	outputPath := filepath.Join(goRoot, filepath.FromSlash(reportPath))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, append(encoded, '\n'), 0o644)
}

func asMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func asMapSlice(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]map[string]any); ok {
			return typed
		}
		return nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, asMap(item))
	}
	return result
}

func asSlice(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func asString(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func asBool(value any) bool {
	if typed, ok := value.(bool); ok {
		return typed
	}
	return false
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}
