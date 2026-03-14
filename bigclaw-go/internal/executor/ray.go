package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type RayConfig struct {
	Address      string
	PollInterval time.Duration
	HTTPTimeout  time.Duration
	BearerToken  string
}

type RayRunner struct {
	baseURL *url.URL
	client  *http.Client
	cfg     RayConfig
	now     func() time.Time
}

type raySubmitRequest struct {
	Entrypoint string         `json:"entrypoint"`
	RuntimeEnv map[string]any `json:"runtime_env,omitempty"`
	JobID      string         `json:"job_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type raySubmitResponse struct {
	JobID string `json:"job_id"`
}

type rayJobInfo struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type rayLogsResponse struct {
	Logs string `json:"logs"`
}

type rayStopResponse struct {
	Stopped bool `json:"stopped"`
}

func NewRayRunner(cfg RayConfig) (*RayRunner, error) {
	if cfg.Address == "" {
		cfg.Address = "http://127.0.0.1:8265"
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 10 * time.Second
	}
	parsed, err := normalizeRayAddress(cfg.Address)
	if err != nil {
		return nil, err
	}
	return NewRayRunnerWithClient(cfg, &http.Client{Timeout: cfg.HTTPTimeout}, parsed), nil
}

func NewRayRunnerWithClient(cfg RayConfig, client *http.Client, parsed *url.URL) *RayRunner {
	return &RayRunner{baseURL: parsed, client: client, cfg: cfg, now: time.Now}
}

func normalizeRayAddress(address string) (*url.URL, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "ray" {
		host := parsed.Hostname()
		if host == "" {
			host = parsed.Path
		}
		if host == "" {
			return nil, fmt.Errorf("invalid ray address: %s", address)
		}
		return &url.URL{Scheme: "http", Host: net.JoinHostPort(host, "8265")}, nil
	}
	return parsed, nil
}

func (r *RayRunner) Kind() domain.ExecutorKind { return domain.ExecutorRay }

func (r *RayRunner) Capability() Capability {
	return Capability{Kind: domain.ExecutorRay, MaxConcurrency: 512, SupportsGPU: true, SupportsShell: true}
}

func (r *RayRunner) Execute(ctx context.Context, task domain.Task) Result {
	submit := raySubmitRequest{
		Entrypoint: resolveRayEntrypoint(task),
		RuntimeEnv: task.RuntimeEnv,
		JobID:      rayJobID(task),
		Metadata:   rayMetadata(task),
	}
	var created raySubmitResponse
	if err := r.doJSON(ctx, http.MethodPost, "/api/jobs/", submit, &created); err != nil {
		return Result{ShouldRetry: true, Message: fmt.Sprintf("submit ray job: %v", err), FinishedAt: r.now()}
	}
	artifacts := []string{fmt.Sprintf("ray://jobs/%s", created.JobID)}
	statusTicker := time.NewTicker(r.cfg.PollInterval)
	defer statusTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			_, _ = r.stopJob(context.Background(), created.JobID)
			return Result{ShouldRetry: true, Message: ctx.Err().Error(), Artifacts: artifacts, FinishedAt: r.now()}
		case <-statusTicker.C:
			info, err := r.jobInfo(ctx, created.JobID)
			if err != nil {
				return Result{ShouldRetry: true, Message: fmt.Sprintf("poll ray job: %v", err), Artifacts: artifacts, FinishedAt: r.now()}
			}
			switch strings.ToUpper(info.Status) {
			case "SUCCEEDED":
				logs, _ := r.jobLogs(ctx, created.JobID)
				return Result{Success: true, Message: fmt.Sprintf("ray job %s succeeded: %s", created.JobID, trimMessage(logs)), Artifacts: artifacts, FinishedAt: r.now()}
			case "FAILED":
				logs, _ := r.jobLogs(ctx, created.JobID)
				message := info.Message
				if logs != "" {
					message = trimMessage(logs)
				}
				return Result{DeadLetter: true, Message: fmt.Sprintf("ray job %s failed: %s", created.JobID, message), Artifacts: artifacts, FinishedAt: r.now()}
			case "STOPPED":
				return Result{ShouldRetry: true, Message: fmt.Sprintf("ray job %s stopped", created.JobID), Artifacts: artifacts, FinishedAt: r.now()}
			}
		}
	}
}

func (r *RayRunner) jobInfo(ctx context.Context, jobID string) (*rayJobInfo, error) {
	var info rayJobInfo
	if err := r.doJSON(ctx, http.MethodGet, "/api/jobs/"+jobID, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (r *RayRunner) jobLogs(ctx context.Context, jobID string) (string, error) {
	var logs rayLogsResponse
	if err := r.doJSON(ctx, http.MethodGet, "/api/jobs/"+jobID+"/logs", nil, &logs); err != nil {
		return "", err
	}
	return logs.Logs, nil
}

func (r *RayRunner) stopJob(ctx context.Context, jobID string) (bool, error) {
	var response rayStopResponse
	if err := r.doJSON(ctx, http.MethodPost, "/api/jobs/"+jobID+"/stop", nil, &response); err != nil {
		return false, err
	}
	return response.Stopped, nil
}

func (r *RayRunner) doJSON(ctx context.Context, method, path string, body any, out any) error {
	endpoint := r.baseURL.ResolveReference(&url.URL{Path: path})
	var payload io.Reader
	if body != nil {
		contents, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(contents)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.cfg.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.BearerToken)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		contents, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ray api status %d: %s", resp.StatusCode, strings.TrimSpace(string(contents)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func resolveRayEntrypoint(task domain.Task) string {
	if task.Entrypoint != "" {
		return task.Entrypoint
	}
	if len(task.Command) > 0 {
		return shellJoin(append(append([]string{}, task.Command...), task.Args...))
	}
	return fmt.Sprintf("echo %s", shellQuote(fmt.Sprintf("BigClaw task %s: %s", task.ID, task.Title)))
}

func rayJobID(task domain.Task) string {
	if task.IdempotencyKey != "" {
		return sanitizeRayJobID(task.IdempotencyKey)
	}
	return sanitizeRayJobID("bigclaw-" + task.ID)
}

func rayMetadata(task domain.Task) map[string]any {
	metadata := map[string]any{"task_id": task.ID, "title": task.Title}
	for key, value := range task.Metadata {
		metadata[key] = value
	}
	return metadata
}

func sanitizeRayJobID(input string) string {
	cleaned := strings.NewReplacer("/", "-", " ", "-", ":", "-", "_", "-").Replace(strings.ToLower(input))
	cleaned = strings.Trim(cleaned, "-")
	if cleaned == "" {
		return fmt.Sprintf("bigclaw-%d", time.Now().Unix())
	}
	return cleaned
}

func shellJoin(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, shellQuote(part))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
