package reporting

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"bigclaw-go/internal/domain"
)

const (
	TaskSmokeGenerator = "bigclaw-go/scripts/e2e/run_task_smoke/main.go"
)

type TaskSmokeOptions struct {
	Executor       string
	Title          string
	Entrypoint     string
	Image          string
	BaseURL        string
	GoRoot         string
	TimeoutSeconds int
	PollInterval   time.Duration
	RuntimeEnvJSON string
	MetadataJSON   string
	ReportPath     string
	Autostart      bool
	HealthAttempts int
	HealthInterval time.Duration
	TimeNow        func() time.Time
	StartBigclawd  func(string, map[string]string) (*exec.Cmd, *os.File, string, error)
}

type TaskSmokeReport struct {
	Autostarted bool             `json:"autostarted"`
	BaseURL     string           `json:"base_url"`
	Task        map[string]any   `json:"task"`
	Status      map[string]any   `json:"status"`
	Events      []map[string]any `json:"events"`
	StateDir    string           `json:"state_dir,omitempty"`
	ServiceLog  string           `json:"service_log,omitempty"`
	Error       string           `json:"error,omitempty"`
}

type TaskSmokeResult struct {
	Report         TaskSmokeReport
	ExitCode       int
	ServiceLogPath string
}

func RunTaskSmoke(options TaskSmokeOptions) (TaskSmokeResult, error) {
	if strings.TrimSpace(options.Executor) == "" {
		return TaskSmokeResult{}, errors.New("executor is required")
	}
	if strings.TrimSpace(options.Title) == "" {
		return TaskSmokeResult{}, errors.New("title is required")
	}
	if strings.TrimSpace(options.Entrypoint) == "" {
		return TaskSmokeResult{}, errors.New("entrypoint is required")
	}
	if options.BaseURL == "" {
		options.BaseURL = "http://127.0.0.1:8080"
	}
	if options.GoRoot == "" {
		root, err := FindRepoRoot(".")
		if err != nil {
			return TaskSmokeResult{}, err
		}
		options.GoRoot = root
	}
	if options.TimeoutSeconds <= 0 {
		options.TimeoutSeconds = 180
	}
	if options.PollInterval <= 0 {
		options.PollInterval = time.Second
	}
	if options.HealthAttempts <= 0 {
		options.HealthAttempts = 60
	}
	if options.HealthInterval <= 0 {
		options.HealthInterval = time.Second
	}
	if options.TimeNow == nil {
		options.TimeNow = time.Now
	}
	if options.StartBigclawd == nil {
		options.StartBigclawd = startTaskSmokeBigclawd
	}

	process, logFile, logPath, stateDir, activeBaseURL, err := prepareTaskSmokeService(options)
	if err != nil {
		return TaskSmokeResult{}, err
	}
	defer stopTaskSmokeProcess(process, logFile)

	submitted, err := submitTaskSmokeTask(activeBaseURL, options)
	if err != nil {
		return TaskSmokeResult{}, err
	}

	deadline := options.TimeNow().Add(time.Duration(options.TimeoutSeconds) * time.Second)
	for options.TimeNow().Before(deadline) {
		status, err := fetchTaskSmokeStatus(activeBaseURL, asString(submitted["id"]))
		if err != nil {
			return TaskSmokeResult{}, err
		}
		state := asString(status["state"])
		if taskSmokeTerminal(state) {
			report, resultErr := finalizeTaskSmokeReport(options, activeBaseURL, submitted, status, stateDir, logPath, process != nil, "")
			if resultErr != nil {
				return TaskSmokeResult{}, resultErr
			}
			exitCode := 1
			if state == string(domain.TaskSucceeded) {
				exitCode = 0
			}
			return TaskSmokeResult{
				Report:         report,
				ExitCode:       exitCode,
				ServiceLogPath: logPath,
			}, nil
		}
		time.Sleep(options.PollInterval)
	}

	status, err := fetchTaskSmokeStatus(activeBaseURL, asString(submitted["id"]))
	if err != nil {
		return TaskSmokeResult{}, err
	}
	report, err := finalizeTaskSmokeReport(options, activeBaseURL, submitted, status, stateDir, logPath, process != nil, "timeout waiting for terminal state")
	if err != nil {
		return TaskSmokeResult{}, err
	}
	return TaskSmokeResult{
		Report:         report,
		ExitCode:       1,
		ServiceLogPath: logPath,
	}, nil
}

func prepareTaskSmokeService(options TaskSmokeOptions) (*exec.Cmd, *os.File, string, string, string, error) {
	activeBaseURL := options.BaseURL
	if !options.Autostart {
		if err := waitForTaskSmokeHealth(activeBaseURL, options.HealthAttempts, options.HealthInterval); err != nil {
			return nil, nil, "", "", activeBaseURL, err
		}
		return nil, nil, "", "", activeBaseURL, nil
	}

	if err := waitForTaskSmokeHealth(activeBaseURL, 2, 200*time.Millisecond); err == nil {
		return nil, nil, "", "", activeBaseURL, nil
	}

	env, reservedBaseURL, stateDir, err := buildTaskSmokeAutostartEnv()
	if err != nil {
		return nil, nil, "", "", activeBaseURL, err
	}
	process, logFile, logPath, err := options.StartBigclawd(options.GoRoot, env)
	if err != nil {
		return nil, nil, "", stateDir, reservedBaseURL, err
	}
	if err := waitForTaskSmokeHealth(reservedBaseURL, options.HealthAttempts, options.HealthInterval); err != nil {
		stopTaskSmokeProcess(process, logFile)
		return nil, nil, "", stateDir, reservedBaseURL, err
	}
	return process, logFile, logPath, stateDir, reservedBaseURL, nil
}

func finalizeTaskSmokeReport(options TaskSmokeOptions, baseURL string, task map[string]any, status map[string]any, stateDir string, logPath string, autostarted bool, reportError string) (TaskSmokeReport, error) {
	report := TaskSmokeReport{
		Autostarted: autostarted,
		BaseURL:     baseURL,
		Task:        task,
		Status:      status,
	}
	events, err := fetchTaskSmokeEvents(baseURL, asString(task["id"]))
	if err != nil {
		return TaskSmokeReport{}, err
	}
	report.Events = events
	if stateDir != "" {
		report.StateDir = stateDir
	}
	if logPath != "" {
		report.ServiceLog = logPath
	}
	if reportError != "" {
		report.Error = reportError
	}
	if options.ReportPath != "" {
		if err := writeTaskSmokeReport(options.GoRoot, options.ReportPath, report); err != nil {
			return TaskSmokeReport{}, err
		}
	}
	return report, nil
}

func buildTaskSmokeAutostartEnv() (map[string]string, string, string, error) {
	env := environmentMap(os.Environ())
	stateDir, err := os.MkdirTemp("", "bigclawd-state-")
	if err != nil {
		return nil, "", "", err
	}
	queueBackend := strings.TrimSpace(env["BIGCLAW_QUEUE_BACKEND"])
	if queueBackend == "" {
		queueBackend = "file"
	}
	switch queueBackend {
	case "sqlite":
		env["BIGCLAW_QUEUE_SQLITE_PATH"] = filepath.Join(stateDir, "queue.db")
	case "file":
		env["BIGCLAW_QUEUE_FILE"] = filepath.Join(stateDir, "queue.json")
	}
	env["BIGCLAW_AUDIT_LOG_PATH"] = filepath.Join(stateDir, "audit.jsonl")
	baseURL, httpAddr, err := reserveTaskSmokeLocalBaseURL()
	if err != nil {
		return nil, "", "", err
	}
	env["BIGCLAW_HTTP_ADDR"] = httpAddr
	return env, baseURL, stateDir, nil
}

func startTaskSmokeBigclawd(goRoot string, env map[string]string) (*exec.Cmd, *os.File, string, error) {
	logFile, err := os.CreateTemp("", "bigclawd-e2e-*.log")
	if err != nil {
		return nil, nil, "", err
	}
	cmd := exec.Command("go", "run", "./cmd/bigclawd")
	cmd.Dir = goRoot
	cmd.Env = environmentSlice(env)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, nil, "", err
	}
	return cmd, logFile, logFile.Name(), nil
}

func stopTaskSmokeProcess(cmd *exec.Cmd, logFile *os.File) {
	if cmd == nil {
		return
	}
	if cmd.Process != nil {
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
	if logFile != nil {
		_ = logFile.Close()
	}
}

func writeTaskSmokeReport(goRoot string, reportPath string, payload TaskSmokeReport) error {
	outputPath := reportPath
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(goRoot, reportPath)
	}
	return WriteJSON(outputPath, payload)
}

func waitForTaskSmokeHealth(baseURL string, attempts int, interval time.Duration) error {
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		payload, err := taskSmokeHTTPJSON(baseURL+"/healthz", http.MethodGet, nil, 10*time.Second)
		if err == nil && asBool(payload["ok"]) {
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("healthz did not report ok: %v", payload)
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("service did not become healthy: %w", lastErr)
}

func reserveTaskSmokeLocalBaseURL() (string, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	defer listener.Close()
	addr := listener.Addr().String()
	return "http://" + addr, addr, nil
}

func submitTaskSmokeTask(baseURL string, options TaskSmokeOptions) (map[string]any, error) {
	taskID := fmt.Sprintf("%s-smoke-%d", options.Executor, options.TimeNow().Unix())
	metadata := map[string]any{
		"smoke_test": "true",
		"executor":   options.Executor,
	}
	if strings.TrimSpace(options.MetadataJSON) != "" {
		var extra map[string]any
		if err := json.Unmarshal([]byte(options.MetadataJSON), &extra); err != nil {
			return nil, err
		}
		for key, value := range extra {
			metadata[key] = value
		}
	}
	task := map[string]any{
		"id":                        taskID,
		"title":                     options.Title,
		"required_executor":         options.Executor,
		"entrypoint":                options.Entrypoint,
		"execution_timeout_seconds": options.TimeoutSeconds,
		"metadata":                  metadata,
	}
	if options.Image != "" {
		task["container_image"] = options.Image
	}
	if strings.TrimSpace(options.RuntimeEnvJSON) != "" {
		var runtimeEnv map[string]any
		if err := json.Unmarshal([]byte(options.RuntimeEnvJSON), &runtimeEnv); err != nil {
			return nil, err
		}
		task["runtime_env"] = runtimeEnv
	}
	payload, err := taskSmokeHTTPJSON(baseURL+"/tasks", http.MethodPost, task, 10*time.Second)
	if err != nil {
		return nil, err
	}
	submitted := asMap(payload["task"])
	if len(submitted) == 0 {
		return nil, errors.New("task submission did not return a task payload")
	}
	return submitted, nil
}

func fetchTaskSmokeStatus(baseURL string, taskID string) (map[string]any, error) {
	return taskSmokeHTTPJSON(baseURL+"/tasks/"+taskID, http.MethodGet, nil, 10*time.Second)
}

func fetchTaskSmokeEvents(baseURL string, taskID string) ([]map[string]any, error) {
	payload, err := taskSmokeHTTPJSON(baseURL+"/events?task_id="+taskID+"&limit=100", http.MethodGet, nil, 10*time.Second)
	if err != nil {
		return nil, err
	}
	eventsRaw := asSlice(payload["events"])
	events := make([]map[string]any, 0, len(eventsRaw))
	for _, item := range eventsRaw {
		events = append(events, asMap(item))
	}
	return events, nil
}

func taskSmokeTerminal(state string) bool {
	switch state {
	case string(domain.TaskSucceeded), string(domain.TaskDeadLetter), string(domain.TaskCancelled), string(domain.TaskFailed):
		return true
	default:
		return false
	}
}

func taskSmokeHTTPJSON(url string, method string, payload any, timeout time.Duration) (map[string]any, error) {
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
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		contents, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s %s returned %d: %s", method, url, resp.StatusCode, strings.TrimSpace(string(contents)))
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func environmentMap(entries []string) map[string]string {
	env := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		env[key] = value
	}
	return env
}

func environmentSlice(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+env[key])
	}
	return out
}

func ParseTaskSmokeCLIFlags(args []string) (TaskSmokeOptions, bool, error) {
	options := TaskSmokeOptions{
		BaseURL:        envOrDefault("BIGCLAW_ADDR", "http://127.0.0.1:8080"),
		TimeoutSeconds: 180,
		PollInterval:   time.Second,
	}
	pretty := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--executor":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --executor")
			}
			options.Executor = args[i]
		case "--title":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --title")
			}
			options.Title = args[i]
		case "--entrypoint":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --entrypoint")
			}
			options.Entrypoint = args[i]
		case "--image":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --image")
			}
			options.Image = args[i]
		case "--base-url":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --base-url")
			}
			options.BaseURL = args[i]
		case "--go-root":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --go-root")
			}
			options.GoRoot = args[i]
		case "--timeout-seconds":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --timeout-seconds")
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return TaskSmokeOptions{}, false, err
			}
			options.TimeoutSeconds = value
		case "--poll-interval":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --poll-interval")
			}
			value, err := strconv.ParseFloat(args[i], 64)
			if err != nil {
				return TaskSmokeOptions{}, false, err
			}
			options.PollInterval = time.Duration(value * float64(time.Second))
		case "--runtime-env-json":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --runtime-env-json")
			}
			options.RuntimeEnvJSON = args[i]
		case "--metadata-json":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --metadata-json")
			}
			options.MetadataJSON = args[i]
		case "--report-path":
			i++
			if i >= len(args) {
				return TaskSmokeOptions{}, false, errors.New("missing value for --report-path")
			}
			options.ReportPath = args[i]
		case "--autostart":
			options.Autostart = true
		case "--pretty":
			pretty = true
		default:
			return TaskSmokeOptions{}, false, fmt.Errorf("unknown argument: %s", args[i])
		}
	}
	if options.Executor != "local" && options.Executor != "kubernetes" && options.Executor != "ray" {
		return TaskSmokeOptions{}, false, fmt.Errorf("unsupported executor: %s", options.Executor)
	}
	return options, pretty, nil
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
