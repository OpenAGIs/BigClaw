package soaklocal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TaskStatus struct {
	ID    string `json:"id,omitempty"`
	State string `json:"state,omitempty"`
}

type Report struct {
	Count                 int          `json:"count"`
	Workers               int          `json:"workers"`
	ElapsedSeconds        float64      `json:"elapsed_seconds"`
	ThroughputTasksPerSec float64      `json:"throughput_tasks_per_sec"`
	Succeeded             int          `json:"succeeded"`
	Failed                int          `json:"failed"`
	SampleStatus          []TaskStatus `json:"sample_status"`
}

type Options struct {
	Count          int
	Workers        int
	BaseURL        string
	GoRoot         string
	TimeoutSeconds int
	Autostart      bool
	ReportPath     string
}

type ServiceProcess struct {
	Cmd     *exec.Cmd
	LogPath string
}

func requestJSON(client *http.Client, baseURL, method, path string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, strings.TrimRight(baseURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s returned %d: %s", method, path, resp.StatusCode, string(raw))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func WaitHealth(baseURL string, timeoutSeconds int) error {
	client := &http.Client{Timeout: 30 * time.Second}
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	for time.Now().Before(deadline) {
		var payload map[string]any
		if err := requestJSON(client, baseURL, http.MethodGet, "/healthz", nil, &payload); err == nil {
			if ok, _ := payload["ok"].(bool); ok {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("health check timeout")
}

func StartService(goRoot string, env []string) (*ServiceProcess, error) {
	logFile, err := os.CreateTemp("", "bigclawd-soak-*.log")
	if err != nil {
		return nil, err
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
	_ = logFile.Close()
	return &ServiceProcess{Cmd: cmd, LogPath: logFile.Name()}, nil
}

func (p *ServiceProcess) Stop() error {
	if p == nil || p.Cmd == nil || p.Cmd.Process == nil {
		return nil
	}
	if err := p.Cmd.Process.Signal(os.Interrupt); err != nil {
		_ = p.Cmd.Process.Kill()
	}
	done := make(chan error, 1)
	go func() { done <- p.Cmd.Wait() }()
	select {
	case <-time.After(5 * time.Second):
		_ = p.Cmd.Process.Kill()
		<-done
	case <-done:
	}
	return nil
}

func submitTask(client *http.Client, baseURL string, task map[string]any) (string, error) {
	var ignored map[string]any
	if err := requestJSON(client, baseURL, http.MethodPost, "/tasks", task, &ignored); err != nil {
		return "", err
	}
	return task["id"].(string), nil
}

func waitTask(client *http.Client, baseURL, taskID string, timeoutSeconds int) (TaskStatus, error) {
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	for time.Now().Before(deadline) {
		var status TaskStatus
		if err := requestJSON(client, baseURL, http.MethodGet, "/tasks/"+taskID, nil, &status); err != nil {
			return TaskStatus{}, err
		}
		switch status.State {
		case "succeeded", "dead_letter", "failed", "cancelled":
			return status, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return TaskStatus{}, fmt.Errorf("timeout waiting for %s", taskID)
}

func Run(opts Options) (Report, string, error) {
	if opts.Count == 0 {
		opts.Count = 50
	}
	if opts.Workers == 0 {
		opts.Workers = 8
	}
	if opts.BaseURL == "" {
		opts.BaseURL = "http://127.0.0.1:8080"
	}
	if opts.TimeoutSeconds == 0 {
		opts.TimeoutSeconds = 180
	}
	if opts.ReportPath == "" {
		opts.ReportPath = "docs/reports/soak-local-report.json"
	}
	start := time.Now()
	var proc *ServiceProcess
	var err error
	if opts.Autostart {
		proc, err = StartService(opts.GoRoot, os.Environ())
		if err != nil {
			return Report{}, "", err
		}
		defer proc.Stop()
	}
	if err := WaitHealth(opts.BaseURL, 60); err != nil {
		return Report{}, procLog(proc), err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	type result struct {
		index  int
		status TaskStatus
		err    error
	}
	sem := make(chan struct{}, opts.Workers)
	results := make(chan result, opts.Count)
	var wg sync.WaitGroup
	for i := 0; i < opts.Count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			task := map[string]any{
				"id":                        fmt.Sprintf("soak-%d-%d", i, start.Unix()),
				"title":                     fmt.Sprintf("soak task %d", i),
				"required_executor":         "local",
				"entrypoint":                fmt.Sprintf("echo soak %d", i),
				"execution_timeout_seconds": opts.TimeoutSeconds,
			}
			taskID, err := submitTask(client, opts.BaseURL, task)
			if err != nil {
				results <- result{index: i, err: err}
				return
			}
			status, err := waitTask(client, opts.BaseURL, taskID, opts.TimeoutSeconds)
			results <- result{index: i, status: status, err: err}
		}(i)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	statuses := make([]TaskStatus, 0, opts.Count)
	var firstErr error
	for item := range results {
		if item.err != nil && firstErr == nil {
			firstErr = item.err
		}
		if item.err == nil {
			statuses = append(statuses, item.status)
		}
	}
	elapsed := time.Since(start).Seconds()
	succeeded := 0
	for _, status := range statuses {
		if status.State == "succeeded" {
			succeeded++
		}
	}
	report := Report{
		Count:          opts.Count,
		Workers:        opts.Workers,
		ElapsedSeconds: elapsed,
		Succeeded:      succeeded,
		Failed:         opts.Count - succeeded,
		SampleStatus:   sampleStatuses(statuses, 3),
	}
	if elapsed > 0 {
		report.ThroughputTasksPerSec = float64(opts.Count) / elapsed
	}
	if err := WriteReport(filepath.Join(opts.GoRoot, opts.ReportPath), report); err != nil {
		return report, procLog(proc), err
	}
	if firstErr != nil {
		return report, procLog(proc), firstErr
	}
	if succeeded != opts.Count {
		return report, procLog(proc), fmt.Errorf("only %d/%d tasks succeeded", succeeded, opts.Count)
	}
	return report, procLog(proc), nil
}

func WriteReport(path string, report Report) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func sampleStatuses(statuses []TaskStatus, limit int) []TaskStatus {
	if len(statuses) <= limit {
		return statuses
	}
	return statuses[:limit]
}

func procLog(proc *ServiceProcess) string {
	if proc == nil {
		return ""
	}
	return proc.LogPath
}
