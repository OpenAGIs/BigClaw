package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type mixedWorkloadArgs struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	Autostart      bool
}

type mixedTaskSpec struct {
	Name             string
	ExpectedExecutor string
	Task             map[string]any
}

type mixedAutostartState struct {
	Env      map[string]string
	BaseURL  string
	StateDir string
}

func main() {
	args := parseMixedWorkloadArgs()
	os.Exit(runMixedWorkloadMatrix(args))
}

func parseMixedWorkloadArgs() mixedWorkloadArgs {
	defaultRoot, err := os.Getwd()
	if err != nil {
		defaultRoot = "."
	}
	args := mixedWorkloadArgs{}
	flag.StringVar(&args.GoRoot, "go-root", defaultRoot, "repo root")
	flag.StringVar(&args.ReportPath, "report-path", "docs/reports/mixed-workload-matrix-report.json", "report output")
	flag.IntVar(&args.TimeoutSeconds, "timeout-seconds", 240, "task timeout in seconds")
	flag.BoolVar(&args.Autostart, "autostart", true, "start a local bigclawd process automatically")
	flag.Parse()
	return args
}

func runMixedWorkloadMatrix(args mixedWorkloadArgs) int {
	goRoot, err := filepath.Abs(args.GoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	process := (*exec.Cmd)(nil)
	logHandle := (*os.File)(nil)
	logPath := ""
	baseURL := getenvDefault("BIGCLAW_ADDR", "http://127.0.0.1:8080")
	stateDir := ""

	defer func() {
		if process != nil {
			terminateProcess(process, logHandle)
			if logPath != "" {
				fmt.Fprintf(os.Stderr, "bigclawd log: %s\n", logPath)
			}
		}
	}()

	if args.Autostart {
		autostartState, err := buildMixedAutostartEnv()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		stateDir = autostartState.StateDir
		baseURL = autostartState.BaseURL
		process, logHandle, logPath, err = startMixedBigclawd(goRoot, autostartState.Env)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	if err := waitForMixedHealth(baseURL, 60, time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	timestamp := time.Now().Unix()
	results := make([]map[string]any, 0, 5)
	allOK := true
	for _, spec := range defaultMixedTasks(timestamp) {
		if err := postMixedTask(baseURL, spec.Task); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		status, err := waitMixedTask(baseURL, asString(spec.Task["id"]), args.TimeoutSeconds)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		events, err := fetchMixedEvents(baseURL, asString(spec.Task["id"]))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		routed := firstRoutedEvent(events)
		routedExecutor := ""
		routedReason := ""
		if routed != nil {
			payload := asMap(routed["payload"])
			routedExecutor = asString(payload["executor"])
			routedReason = asString(payload["reason"])
		}
		statusMap := asMap(status)
		ok := asString(statusMap["state"]) == "succeeded" && routedExecutor == spec.ExpectedExecutor
		if !ok {
			allOK = false
		}
		results = append(results, map[string]any{
			"name":              spec.Name,
			"task_id":           asString(spec.Task["id"]),
			"trace_id":          asString(spec.Task["trace_id"]),
			"expected_executor": spec.ExpectedExecutor,
			"routed_executor":   routedExecutor,
			"routed_reason":     routedReason,
			"final_state":       asString(statusMap["state"]),
			"latest_event_type": latestEventType(statusMap),
			"events":            events,
			"ok":                ok,
		})
	}

	report := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"base_url":     baseURL,
		"state_dir":    stateDir,
		"service_log":  logPath,
		"all_ok":       allOK,
		"tasks":        results,
	}
	if err := writeMixedReport(goRoot, args.ReportPath, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(string(encoded))
	if allOK {
		return 0
	}
	return 1
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func buildMixedAutostartEnv() (mixedAutostartState, error) {
	stateDir, err := os.MkdirTemp("", "bigclawd-mixed-state-")
	if err != nil {
		return mixedAutostartState{}, err
	}
	baseURL, httpAddr, err := reserveMixedBaseURL()
	if err != nil {
		return mixedAutostartState{}, err
	}
	env := copyMixedEnv()
	env["BIGCLAW_HTTP_ADDR"] = httpAddr
	env["BIGCLAW_QUEUE_BACKEND"] = getenvMapDefault(env, "BIGCLAW_QUEUE_BACKEND", "sqlite")
	env["BIGCLAW_QUEUE_SQLITE_PATH"] = filepath.Join(stateDir, "queue.db")
	env["BIGCLAW_AUDIT_LOG_PATH"] = filepath.Join(stateDir, "audit.jsonl")
	env["BIGCLAW_SERVICE_NAME"] = getenvMapDefault(env, "BIGCLAW_SERVICE_NAME", "bigclawd-mixed")
	return mixedAutostartState{Env: env, BaseURL: baseURL, StateDir: stateDir}, nil
}

func getenvMapDefault(env map[string]string, key, fallback string) string {
	if value := env[key]; value != "" {
		return value
	}
	return fallback
}

func copyMixedEnv() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		parts := bytes.SplitN([]byte(entry), []byte("="), 2)
		if len(parts) == 2 {
			env[string(parts[0])] = string(parts[1])
		}
	}
	return env
}

func reserveMixedBaseURL() (string, string, error) {
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

func startMixedBigclawd(goRoot string, env map[string]string) (*exec.Cmd, *os.File, string, error) {
	logPath, err := os.CreateTemp("", "bigclawd-mixed-*.log")
	if err != nil {
		return nil, nil, "", err
	}
	cmd := exec.Command("go", "run", "./cmd/bigclawd")
	cmd.Dir = goRoot
	cmd.Stdout = logPath
	cmd.Stderr = logPath
	cmd.Env = mixedEnvSlice(env)
	if err := cmd.Start(); err != nil {
		_ = logPath.Close()
		return nil, nil, "", err
	}
	return cmd, logPath, logPath.Name(), nil
}

func mixedEnvSlice(env map[string]string) []string {
	values := make([]string, 0, len(env))
	for key, value := range env {
		values = append(values, key+"="+value)
	}
	return values
}

func terminateProcess(process *exec.Cmd, logHandle *os.File) {
	if process == nil || process.Process == nil {
		return
	}
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
		<-done
	}
	if logHandle != nil {
		_ = logHandle.Close()
	}
}

func waitForMixedHealth(baseURL string, attempts int, interval time.Duration) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		payload, err := mixedHTTPJSON(baseURL+"/healthz", http.MethodGet, nil, 30*time.Second)
		if err == nil && asBool(payload["ok"]) {
			return nil
		}
		lastErr = err
		time.Sleep(interval)
	}
	return fmt.Errorf("service did not become healthy: %v", lastErr)
}

func mixedHTTPJSON(url string, method string, payload any, timeout time.Duration) (map[string]any, error) {
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
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d: %v", resp.StatusCode, decoded)
	}
	return decoded, nil
}

func postMixedTask(baseURL string, task map[string]any) error {
	_, err := mixedHTTPJSON(baseURL+"/tasks", http.MethodPost, task, 30*time.Second)
	return err
}

func waitMixedTask(baseURL string, taskID string, timeoutSeconds int) (map[string]any, error) {
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	for time.Now().Before(deadline) {
		status, err := mixedHTTPJSON(baseURL+"/tasks/"+taskID, http.MethodGet, nil, 30*time.Second)
		if err != nil {
			return nil, err
		}
		if isMixedTerminal(asString(status["state"])) {
			return status, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("task %s did not reach terminal state before timeout", taskID)
}

func isMixedTerminal(state string) bool {
	switch state {
	case "succeeded", "dead_letter", "cancelled", "failed":
		return true
	default:
		return false
	}
}

func fetchMixedEvents(baseURL string, taskID string) ([]map[string]any, error) {
	payload, err := mixedHTTPJSON(baseURL+"/events?task_id="+taskID+"&limit=100", http.MethodGet, nil, 30*time.Second)
	if err != nil {
		return nil, err
	}
	return asMapSlice(payload["events"]), nil
}

func firstRoutedEvent(events []map[string]any) map[string]any {
	for _, event := range events {
		if asString(event["type"]) == "scheduler.routed" {
			return event
		}
	}
	return nil
}

func latestEventType(status map[string]any) string {
	latest := asMap(status["latest_event"])
	return asString(latest["type"])
}

func defaultMixedTasks(timestamp int64) []mixedTaskSpec {
	return []mixedTaskSpec{
		{
			Name:             "local-default",
			ExpectedExecutor: "local",
			Task: map[string]any{
				"id":         fmt.Sprintf("mixed-local-%d", timestamp),
				"trace_id":   fmt.Sprintf("mixed-local-%d", timestamp),
				"title":      "Mixed workload local default",
				"entrypoint": "echo local default",
				"metadata": map[string]any{
					"scenario": "mixed-workload",
					"profile":  "local-default",
				},
			},
		},
		{
			Name:             "browser-auto",
			ExpectedExecutor: "kubernetes",
			Task: map[string]any{
				"id":              fmt.Sprintf("mixed-browser-%d", timestamp),
				"trace_id":        fmt.Sprintf("mixed-browser-%d", timestamp),
				"title":           "Mixed workload browser auto-route",
				"required_tools":  []string{"browser"},
				"container_image": "alpine:3.20",
				"entrypoint":      "echo browser via kubernetes",
				"metadata": map[string]any{
					"scenario": "mixed-workload",
					"profile":  "browser-auto",
				},
			},
		},
		{
			Name:             "gpu-auto",
			ExpectedExecutor: "ray",
			Task: map[string]any{
				"id":             fmt.Sprintf("mixed-gpu-%d", timestamp),
				"trace_id":       fmt.Sprintf("mixed-gpu-%d", timestamp),
				"title":          "Mixed workload gpu auto-route",
				"required_tools": []string{"gpu"},
				"entrypoint":     "python -c \"print('gpu via ray')\"",
				"metadata": map[string]any{
					"scenario": "mixed-workload",
					"profile":  "gpu-auto",
				},
			},
		},
		{
			Name:             "high-risk-auto",
			ExpectedExecutor: "kubernetes",
			Task: map[string]any{
				"id":              fmt.Sprintf("mixed-risk-%d", timestamp),
				"trace_id":        fmt.Sprintf("mixed-risk-%d", timestamp),
				"title":           "Mixed workload high-risk auto-route",
				"risk_level":      "high",
				"container_image": "alpine:3.20",
				"entrypoint":      "echo high risk via kubernetes",
				"metadata": map[string]any{
					"scenario": "mixed-workload",
					"profile":  "high-risk-auto",
				},
			},
		},
		{
			Name:             "required-ray",
			ExpectedExecutor: "ray",
			Task: map[string]any{
				"id":                fmt.Sprintf("mixed-required-ray-%d", timestamp),
				"trace_id":          fmt.Sprintf("mixed-required-ray-%d", timestamp),
				"title":             "Mixed workload explicit ray",
				"required_executor": "ray",
				"entrypoint":        "python -c \"print('required ray')\"",
				"metadata": map[string]any{
					"scenario": "mixed-workload",
					"profile":  "required-ray",
				},
			},
		},
	}
}

func writeMixedReport(goRoot string, reportPath string, payload map[string]any) error {
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
