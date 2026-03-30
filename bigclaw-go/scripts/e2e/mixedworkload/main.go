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
	"strings"
	"syscall"
	"time"
)

type taskSpec struct {
	Name             string         `json:"name"`
	ExpectedExecutor string         `json:"expected_executor"`
	Task             map[string]any `json:"task"`
}

type eventRecord struct {
	ID        string         `json:"id,omitempty"`
	Type      string         `json:"type,omitempty"`
	TaskID    string         `json:"task_id,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	Timestamp string         `json:"timestamp,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type taskStatus struct {
	ID          string       `json:"id,omitempty"`
	State       string       `json:"state"`
	LatestEvent *eventRecord `json:"latest_event,omitempty"`
}

type eventsResponse struct {
	Events []eventRecord `json:"events"`
}

type taskResult struct {
	Name             string        `json:"name"`
	TaskID           string        `json:"task_id"`
	TraceID          string        `json:"trace_id"`
	ExpectedExecutor string        `json:"expected_executor"`
	RoutedExecutor   string        `json:"routed_executor,omitempty"`
	RoutedReason     string        `json:"routed_reason,omitempty"`
	FinalState       string        `json:"final_state"`
	LatestEventType  string        `json:"latest_event_type"`
	Events           []eventRecord `json:"events"`
	OK               bool          `json:"ok"`
}

type report struct {
	GeneratedAt string       `json:"generated_at"`
	BaseURL     string       `json:"base_url"`
	StateDir    string       `json:"state_dir"`
	ServiceLog  string       `json:"service_log"`
	AllOK       bool         `json:"all_ok"`
	Tasks       []taskResult `json:"tasks"`
}

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	flags := flag.NewFlagSet("mixed-workload-matrix", flag.ExitOnError)
	goRoot := flags.String("go-root", wd, "Go repo root")
	reportPath := flags.String("report-path", "docs/reports/mixed-workload-matrix-report.json", "report path")
	timeoutSeconds := flags.Int("timeout-seconds", 240, "task timeout in seconds")
	autostart := flags.Bool("autostart", true, "autostart an isolated bigclawd if BIGCLAW_ADDR is not already healthy")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	baseURL := getenvDefault("BIGCLAW_ADDR", "http://127.0.0.1:8080")
	var process *exec.Cmd
	var logPath string
	var stateDir string

	cleanup := func() {}
	if *autostart {
		var env []string
		env, baseURL, stateDir, err = buildAutostartEnv()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		process, logPath, err = startBigclawd(*goRoot, env)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cleanup = func() {
			if process == nil || process.Process == nil {
				return
			}
			_ = process.Process.Signal(syscall.SIGTERM)
			done := make(chan error, 1)
			go func() {
				done <- process.Wait()
			}()
			select {
			case <-time.After(5 * time.Second):
				_ = process.Process.Kill()
				<-done
			case <-done:
			}
			if logPath != "" {
				fmt.Fprintf(os.Stderr, "bigclawd log: %s\n", logPath)
			}
		}
	}
	defer cleanup()

	if err := waitForHealth(baseURL, 60, time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	timestamp := time.Now().Unix()
	taskSpecs := defaultTasks(timestamp)
	results := make([]taskResult, 0, len(taskSpecs))
	allOK := true
	for _, spec := range taskSpecs {
		if err := requestJSON(http.MethodPost, baseURL+"/tasks", spec.Task, &map[string]any{}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		status, err := waitTask(baseURL, stringValue(spec.Task["id"]), time.Duration(*timeoutSeconds)*time.Second)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		events, err := fetchEvents(baseURL, stringValue(spec.Task["id"]))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		routedExecutor, routedReason := routedEventPayload(events)
		latestEventType := ""
		if status.LatestEvent != nil {
			latestEventType = status.LatestEvent.Type
		}
		ok := status.State == "succeeded" && routedExecutor == spec.ExpectedExecutor
		if !ok {
			allOK = false
		}
		results = append(results, taskResult{
			Name:             spec.Name,
			TaskID:           stringValue(spec.Task["id"]),
			TraceID:          stringValue(spec.Task["trace_id"]),
			ExpectedExecutor: spec.ExpectedExecutor,
			RoutedExecutor:   routedExecutor,
			RoutedReason:     routedReason,
			FinalState:       status.State,
			LatestEventType:  latestEventType,
			Events:           events,
			OK:               ok,
		})
	}

	rep := report{
		GeneratedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		BaseURL:     baseURL,
		StateDir:    stateDir,
		ServiceLog:  logPath,
		AllOK:       allOK,
		Tasks:       results,
	}
	if err := writeReport(*goRoot, *reportPath, rep); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(body))
	if rep.AllOK {
		os.Exit(0)
	}
	os.Exit(1)
}

func requestJSON(method, url string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s failed: %s %s", method, url, resp.Status, strings.TrimSpace(string(data)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func waitForHealth(baseURL string, attempts int, interval time.Duration) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		var payload map[string]any
		if err := requestJSON(http.MethodGet, baseURL+"/healthz", nil, &payload); err == nil && payload["ok"] == true {
			return nil
		} else if err != nil {
			lastErr = err
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("service did not become healthy: %v", lastErr)
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

func buildAutostartEnv() ([]string, string, string, error) {
	env := os.Environ()
	stateDir, err := os.MkdirTemp("", "bigclawd-mixed-state-")
	if err != nil {
		return nil, "", "", err
	}
	baseURL, httpAddr, err := reserveLocalBaseURL()
	if err != nil {
		return nil, "", "", err
	}
	env = append(env, "BIGCLAW_HTTP_ADDR="+httpAddr)
	queueBackend := getenvDefault("BIGCLAW_QUEUE_BACKEND", "sqlite")
	env = append(env, "BIGCLAW_QUEUE_BACKEND="+queueBackend)
	env = append(env, "BIGCLAW_QUEUE_SQLITE_PATH="+filepath.Join(stateDir, "queue.db"))
	env = append(env, "BIGCLAW_AUDIT_LOG_PATH="+filepath.Join(stateDir, "audit.jsonl"))
	env = append(env, "BIGCLAW_SERVICE_NAME="+getenvDefault("BIGCLAW_SERVICE_NAME", "bigclawd-mixed"))
	return env, baseURL, stateDir, nil
}

func startBigclawd(goRoot string, env []string) (*exec.Cmd, string, error) {
	logFile, err := os.CreateTemp("", "bigclawd-mixed-*.log")
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

func terminal(state string) bool {
	switch state {
	case "succeeded", "dead_letter", "cancelled", "failed":
		return true
	default:
		return false
	}
}

func waitTask(baseURL, taskID string, timeout time.Duration) (taskStatus, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var status taskStatus
		if err := requestJSON(http.MethodGet, baseURL+"/tasks/"+taskID, nil, &status); err != nil {
			return taskStatus{}, err
		}
		if terminal(status.State) {
			return status, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return taskStatus{}, fmt.Errorf("timeout waiting for task %s", taskID)
}

func fetchEvents(baseURL, taskID string) ([]eventRecord, error) {
	var payload eventsResponse
	if err := requestJSON(http.MethodGet, baseURL+"/events?task_id="+taskID+"&limit=100", nil, &payload); err != nil {
		return nil, err
	}
	return payload.Events, nil
}

func routedEventPayload(events []eventRecord) (string, string) {
	for _, event := range events {
		if event.Type != "scheduler.routed" || event.Payload == nil {
			continue
		}
		return stringValue(event.Payload["executor"]), stringValue(event.Payload["reason"])
	}
	return "", ""
}

func defaultTasks(timestamp int64) []taskSpec {
	ts := fmt.Sprintf("%d", timestamp)
	return []taskSpec{
		{
			Name:             "local-default",
			ExpectedExecutor: "local",
			Task: map[string]any{
				"id":         "mixed-local-" + ts,
				"trace_id":   "mixed-local-" + ts,
				"title":      "Mixed workload local default",
				"entrypoint": "echo local default",
				"metadata":   map[string]any{"scenario": "mixed-workload", "profile": "local-default"},
			},
		},
		{
			Name:             "browser-auto",
			ExpectedExecutor: "kubernetes",
			Task: map[string]any{
				"id":              "mixed-browser-" + ts,
				"trace_id":        "mixed-browser-" + ts,
				"title":           "Mixed workload browser auto-route",
				"required_tools":  []string{"browser"},
				"container_image": "alpine:3.20",
				"entrypoint":      "echo browser via kubernetes",
				"metadata":        map[string]any{"scenario": "mixed-workload", "profile": "browser-auto"},
			},
		},
		{
			Name:             "gpu-auto",
			ExpectedExecutor: "ray",
			Task: map[string]any{
				"id":             "mixed-gpu-" + ts,
				"trace_id":       "mixed-gpu-" + ts,
				"title":          "Mixed workload gpu auto-route",
				"required_tools": []string{"gpu"},
				"entrypoint":     "python -c \"print('gpu via ray')\"",
				"metadata":       map[string]any{"scenario": "mixed-workload", "profile": "gpu-auto"},
			},
		},
		{
			Name:             "high-risk-auto",
			ExpectedExecutor: "kubernetes",
			Task: map[string]any{
				"id":              "mixed-risk-" + ts,
				"trace_id":        "mixed-risk-" + ts,
				"title":           "Mixed workload high-risk auto-route",
				"risk_level":      "high",
				"container_image": "alpine:3.20",
				"entrypoint":      "echo high risk via kubernetes",
				"metadata":        map[string]any{"scenario": "mixed-workload", "profile": "high-risk-auto"},
			},
		},
		{
			Name:             "required-ray",
			ExpectedExecutor: "ray",
			Task: map[string]any{
				"id":                "mixed-required-ray-" + ts,
				"trace_id":          "mixed-required-ray-" + ts,
				"title":             "Mixed workload explicit ray",
				"required_executor": "ray",
				"entrypoint":        "python -c \"print('required ray')\"",
				"metadata":          map[string]any{"scenario": "mixed-workload", "profile": "required-ray"},
			},
		},
	}
}

func writeReport(goRoot, reportPath string, rep report) error {
	outputPath := reportPath
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(goRoot, outputPath)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, append(body, '\n'), 0o644)
}

func getenvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
