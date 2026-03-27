package runtasksmoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Options struct {
	Executor      string
	Title         string
	Entrypoint    string
	Image         string
	BaseURL       string
	GoRoot        string
	Timeout       time.Duration
	PollInterval  time.Duration
	RuntimeEnv    map[string]any
	Metadata      map[string]any
	ReportPath    string
	Autostart     bool
	HealthPolls   int
	HealthBackoff time.Duration
}

type Runner struct {
	startProcess func(goRoot string, env []string) (*exec.Cmd, string, error)
	sleep        func(time.Duration)
}

func Run(opts Options) (map[string]any, int, error) {
	runner := Runner{
		startProcess: startBigclawd,
		sleep:        time.Sleep,
	}
	return runner.Run(opts)
}

func (r Runner) Run(opts Options) (map[string]any, int, error) {
	baseURL := strings.TrimSpace(opts.BaseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	goRoot := strings.TrimSpace(opts.GoRoot)
	if goRoot == "" {
		goRoot = "."
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	healthPolls := opts.HealthPolls
	if healthPolls <= 0 {
		healthPolls = 60
	}
	healthBackoff := opts.HealthBackoff
	if healthBackoff <= 0 {
		healthBackoff = time.Second
	}

	var cmd *exec.Cmd
	var logPath string
	stateDir := ""
	activeBaseURL := baseURL
	autostarted := false

	defer func() {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			done := make(chan struct{})
			go func() {
				_, _ = cmd.Process.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = cmd.Process.Kill()
				<-done
			}
		}
	}()

	if opts.Autostart {
		if err := waitForHealth(activeBaseURL, 2, 200*time.Millisecond, r.sleep); err != nil {
			env, autostartURL, dir, buildErr := buildAutostartEnv()
			if buildErr != nil {
				return nil, 1, buildErr
			}
			cmd, logPath, buildErr = r.startProcess(goRoot, env)
			if buildErr != nil {
				return nil, 1, buildErr
			}
			activeBaseURL = autostartURL
			stateDir = dir
			autostarted = true
			if err := waitForHealth(activeBaseURL, healthPolls, healthBackoff, r.sleep); err != nil {
				return nil, 1, err
			}
		}
	} else if err := waitForHealth(activeBaseURL, healthPolls, healthBackoff, r.sleep); err != nil {
		return nil, 1, err
	}

	task := map[string]any{
		"id":                        fmt.Sprintf("%s-smoke-%d", opts.Executor, time.Now().Unix()),
		"title":                     opts.Title,
		"required_executor":         opts.Executor,
		"entrypoint":                opts.Entrypoint,
		"execution_timeout_seconds": int(timeout / time.Second),
		"metadata": map[string]any{
			"smoke_test": "true",
			"executor":   opts.Executor,
		},
	}
	if strings.TrimSpace(opts.Image) != "" {
		task["container_image"] = opts.Image
	}
	if len(opts.RuntimeEnv) > 0 {
		task["runtime_env"] = opts.RuntimeEnv
	}
	for key, value := range opts.Metadata {
		nestedMap(task, "metadata")[key] = value
	}

	submittedPayload, err := requestJSON(activeBaseURL+"/tasks", http.MethodPost, task, 10*time.Second)
	if err != nil {
		return nil, 1, err
	}
	submitted := nestedMap(submittedPayload, "task")
	taskID := stringValue(submitted["id"], stringValue(task["id"], ""))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := requestJSON(activeBaseURL+"/tasks/"+taskID, http.MethodGet, nil, 10*time.Second)
		if err != nil {
			return nil, 1, err
		}
		state := stringValue(status["state"], "")
		if terminal(state) {
			eventsPayload, err := requestJSON(activeBaseURL+"/events?task_id="+taskID+"&limit=100", http.MethodGet, nil, 10*time.Second)
			if err != nil {
				return nil, 1, err
			}
			report := map[string]any{
				"autostarted": autostarted,
				"base_url":    activeBaseURL,
				"task":        submitted,
				"status":      status,
				"events":      anySliceAt(eventsPayload, "events"),
			}
			if stateDir != "" {
				report["state_dir"] = stateDir
			}
			if logPath != "" {
				report["service_log"] = logPath
			}
			if err := writeReport(goRoot, opts.ReportPath, report); err != nil {
				return nil, 1, err
			}
			exitCode := 1
			if state == "succeeded" {
				exitCode = 0
			}
			return report, exitCode, nil
		}
		r.sleep(pollInterval)
	}

	status, statusErr := requestJSON(activeBaseURL+"/tasks/"+taskID, http.MethodGet, nil, 10*time.Second)
	if statusErr != nil {
		return nil, 1, statusErr
	}
	eventsPayload, eventsErr := requestJSON(activeBaseURL+"/events?task_id="+taskID+"&limit=100", http.MethodGet, nil, 10*time.Second)
	if eventsErr != nil {
		return nil, 1, eventsErr
	}
	report := map[string]any{
		"autostarted": autostarted,
		"base_url":    activeBaseURL,
		"task":        submitted,
		"status":      status,
		"events":      anySliceAt(eventsPayload, "events"),
		"error":       "timeout waiting for terminal state",
	}
	if stateDir != "" {
		report["state_dir"] = stateDir
	}
	if logPath != "" {
		report["service_log"] = logPath
	}
	if err := writeReport(goRoot, opts.ReportPath, report); err != nil {
		return nil, 1, err
	}
	return report, 1, nil
}

func buildAutostartEnv() ([]string, string, string, error) {
	envMap := map[string]string{}
	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	stateDir, err := os.MkdirTemp("", "bigclawd-state-")
	if err != nil {
		return nil, "", "", err
	}
	queueBackend := envMap["BIGCLAW_QUEUE_BACKEND"]
	if queueBackend == "" {
		queueBackend = "file"
	}
	switch queueBackend {
	case "sqlite":
		envMap["BIGCLAW_QUEUE_SQLITE_PATH"] = filepath.Join(stateDir, "queue.db")
	case "file":
		envMap["BIGCLAW_QUEUE_FILE"] = filepath.Join(stateDir, "queue.json")
	}
	envMap["BIGCLAW_AUDIT_LOG_PATH"] = filepath.Join(stateDir, "audit.jsonl")
	baseURL, httpAddr, err := reserveLocalBaseURL()
	if err != nil {
		return nil, "", "", err
	}
	envMap["BIGCLAW_HTTP_ADDR"] = httpAddr
	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, key+"="+value)
	}
	return env, baseURL, stateDir, nil
}

func reserveLocalBaseURL() (string, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	defer listener.Close()
	addr := listener.Addr().String()
	return "http://" + addr, addr, nil
}

func startBigclawd(goRoot string, env []string) (*exec.Cmd, string, error) {
	logFile, err := os.CreateTemp("", "bigclawd-e2e-*.log")
	if err != nil {
		return nil, "", err
	}
	cmd := exec.Command("go", "run", "./cmd/bigclawd")
	cmd.Dir = goRoot
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, "", err
	}
	_ = logFile.Close()
	return cmd, logFile.Name(), nil
}

func waitForHealth(baseURL string, attempts int, interval time.Duration, sleep func(time.Duration)) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		payload, err := requestJSON(strings.TrimRight(baseURL, "/")+"/healthz", http.MethodGet, nil, 10*time.Second)
		if err == nil && boolValue(payload["ok"]) {
			return nil
		}
		if err != nil {
			lastErr = err
		}
		sleep(interval)
	}
	return fmt.Errorf("service did not become healthy: %v", lastErr)
}

func requestJSON(url, method string, payload map[string]any, timeout time.Duration) (map[string]any, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func writeReport(goRoot, reportPath string, payload map[string]any) error {
	if strings.TrimSpace(reportPath) == "" {
		return nil
	}
	outputPath := filepath.Join(goRoot, reportPath)
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, body, 0o644)
}

func terminal(state string) bool {
	switch state {
	case "succeeded", "dead_letter", "cancelled", "failed":
		return true
	default:
		return false
	}
}

func nestedMap(target map[string]any, key string) map[string]any {
	if value, ok := target[key].(map[string]any); ok && value != nil {
		return value
	}
	value := map[string]any{}
	target[key] = value
	return value
}

func anySliceAt(target map[string]any, key string) []any {
	if value, ok := target[key].([]any); ok {
		return value
	}
	return nil
}

func boolValue(value any) bool {
	flag, _ := value.(bool)
	return flag
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok || text == "" {
		return fallback
	}
	return text
}
