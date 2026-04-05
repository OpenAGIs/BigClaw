package reporting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	BenchmarkMatrixGenerator = "bigclaw-go/scripts/benchmark/run_matrix/main.go"
	LocalSoakGenerator       = "bigclaw-go/scripts/benchmark/soak_local/main.go"
)

var benchmarkStdoutPattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$`)

type LocalSoakOptions struct {
	Count          int
	Workers        int
	BaseURL        string
	GoRoot         string
	TimeoutSeconds int
	Autostart      bool
	ReportPath     string
	TimeNow        func() time.Time
	HTTPClient     *http.Client
	StartBigclawd  func(string, map[string]string) (*exec.Cmd, *os.File, string, error)
}

type BenchmarkMatrixOptions struct {
	GoRoot          string
	ReportPath      string
	TimeoutSeconds  int
	Scenarios       []string
	BenchmarkRunner func(string) (map[string]any, error)
	SoakRunner      func(string, int, int, int, string) (map[string]any, error)
}

func RunLocalSoak(options LocalSoakOptions) (map[string]any, int, error) {
	if options.Count <= 0 {
		options.Count = 50
	}
	if options.Workers <= 0 {
		options.Workers = 8
	}
	if options.BaseURL == "" {
		options.BaseURL = "http://127.0.0.1:8080"
	}
	if options.TimeoutSeconds <= 0 {
		options.TimeoutSeconds = 180
	}
	if options.TimeNow == nil {
		options.TimeNow = time.Now
	}
	if options.GoRoot == "" {
		root, err := FindRepoRoot(".")
		if err != nil {
			return nil, 1, err
		}
		options.GoRoot = root
	}
	if options.StartBigclawd == nil {
		options.StartBigclawd = startTaskSmokeBigclawd
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	process, logFile, _, _, activeBaseURL, err := prepareTaskSmokeService(TaskSmokeOptions{
		BaseURL:        options.BaseURL,
		GoRoot:         options.GoRoot,
		Autostart:      options.Autostart,
		HealthAttempts: 60,
		HealthInterval: time.Second,
		StartBigclawd:  options.StartBigclawd,
	})
	if err != nil {
		return nil, 1, err
	}
	defer stopTaskSmokeProcess(process, logFile)

	start := options.TimeNow()
	tasks := make([]map[string]any, 0, options.Count)
	for idx := 0; idx < options.Count; idx++ {
		tasks = append(tasks, map[string]any{
			"id":                        fmt.Sprintf("soak-%d-%d", idx, start.Unix()),
			"title":                     fmt.Sprintf("soak task %d", idx),
			"required_executor":         "local",
			"entrypoint":                fmt.Sprintf("echo soak %d", idx),
			"execution_timeout_seconds": options.TimeoutSeconds,
		})
	}

	ids, submitErr := runParallelLocalSoak(options.Workers, tasks, func(task map[string]any) (string, error) {
		if _, err := requestTaskSmokeJSON(client, activeBaseURL, http.MethodPost, "/tasks", task); err != nil {
			return "", err
		}
		return asString(task["id"]), nil
	})
	if submitErr != nil {
		return nil, 1, submitErr
	}
	statuses, waitErr := runParallelLocalSoak(options.Workers, ids, func(taskID string) (map[string]any, error) {
		return waitLocalSoakTask(client, activeBaseURL, taskID, time.Duration(options.TimeoutSeconds)*time.Second)
	})
	if waitErr != nil {
		return nil, 1, waitErr
	}

	elapsed := options.TimeNow().Sub(start).Seconds()
	succeeded := 0
	for _, status := range statuses {
		if asString(status["state"]) == "succeeded" {
			succeeded++
		}
	}
	report := map[string]any{
		"count":                    options.Count,
		"workers":                  options.Workers,
		"elapsed_seconds":          elapsed,
		"throughput_tasks_per_sec": 0.0,
		"succeeded":                succeeded,
		"failed":                   options.Count - succeeded,
		"sample_status":            firstStatusSamples(statuses, 3),
	}
	if elapsed > 0 {
		report["throughput_tasks_per_sec"] = float64(options.Count) / elapsed
	}
	if options.ReportPath != "" {
		if err := WriteJSON(resolveReportPath(options.GoRoot, options.ReportPath), report); err != nil {
			return nil, 1, err
		}
	}
	exitCode := 1
	if succeeded == options.Count {
		exitCode = 0
	}
	return report, exitCode, nil
}

func RunBenchmarkMatrix(options BenchmarkMatrixOptions) (map[string]any, error) {
	if options.GoRoot == "" {
		root, err := FindRepoRoot(".")
		if err != nil {
			return nil, err
		}
		options.GoRoot = root
	}
	if options.ReportPath == "" {
		options.ReportPath = "docs/reports/benchmark-matrix-report.json"
	}
	if options.TimeoutSeconds <= 0 {
		options.TimeoutSeconds = 180
	}
	if len(options.Scenarios) == 0 {
		options.Scenarios = []string{"50:8", "100:12"}
	}
	if options.BenchmarkRunner == nil {
		options.BenchmarkRunner = runBenchmarks
	}
	if options.SoakRunner == nil {
		options.SoakRunner = func(goRoot string, count int, workers int, timeoutSeconds int, reportPath string) (map[string]any, error) {
			report, _, err := RunLocalSoak(LocalSoakOptions{
				Count:          count,
				Workers:        workers,
				GoRoot:         goRoot,
				TimeoutSeconds: timeoutSeconds,
				Autostart:      true,
				ReportPath:     reportPath,
			})
			return report, err
		}
	}

	benchmarkResult, err := options.BenchmarkRunner(options.GoRoot)
	if err != nil {
		return nil, err
	}
	soakResults := make([]map[string]any, 0, len(options.Scenarios))
	for _, scenario := range options.Scenarios {
		count, workers, err := parseBenchmarkScenario(scenario)
		if err != nil {
			return nil, err
		}
		reportPath := fmt.Sprintf("docs/reports/soak-local-%dx%d.json", count, workers)
		soakReport, err := options.SoakRunner(options.GoRoot, count, workers, options.TimeoutSeconds, reportPath)
		if err != nil {
			return nil, err
		}
		soakResults = append(soakResults, map[string]any{
			"scenario": map[string]any{
				"count":   count,
				"workers": workers,
			},
			"report_path": reportPath,
			"result":      soakReport,
		})
	}
	report := map[string]any{
		"benchmark":   benchmarkResult,
		"soak_matrix": soakResults,
	}
	if err := WriteJSON(resolveReportPath(options.GoRoot, options.ReportPath), report); err != nil {
		return nil, err
	}
	return report, nil
}

func parseBenchmarkStdout(stdout string) map[string]any {
	parsed := map[string]any{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		matches := benchmarkStdoutPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		value, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			continue
		}
		parsed[matches[1]] = map[string]any{"ns_per_op": value}
	}
	return parsed
}

func runBenchmarks(goRoot string) (map[string]any, error) {
	cmd := exec.Command("go", "test", "-bench", ".", "./internal/queue", "./internal/scheduler")
	cmd.Dir = goRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run benchmarks: %w: %s", err, strings.TrimSpace(string(output)))
	}
	stdout := string(output)
	return map[string]any{
		"stdout": stdout,
		"parsed": parseBenchmarkStdout(stdout),
	}, nil
}

func parseBenchmarkScenario(value string) (int, int, error) {
	countText, workerText, found := strings.Cut(strings.TrimSpace(value), ":")
	if !found {
		return 0, 0, fmt.Errorf("invalid scenario %q; expected count:workers", value)
	}
	count, err := strconv.Atoi(countText)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid count in scenario %q: %w", value, err)
	}
	workers, err := strconv.Atoi(workerText)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid workers in scenario %q: %w", value, err)
	}
	return count, workers, nil
}

func requestTaskSmokeJSON(client *http.Client, baseURL string, method string, path string, payload any) (map[string]any, error) {
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		contents, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(contents)
	}
	req, err := http.NewRequest(method, strings.TrimRight(baseURL, "/")+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s failed with status %d", method, req.URL.String(), resp.StatusCode)
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func waitLocalSoakTask(client *http.Client, baseURL string, taskID string, timeout time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := requestTaskSmokeJSON(client, baseURL, http.MethodGet, "/tasks/"+taskID, nil)
		if err != nil {
			return nil, err
		}
		switch asString(status["state"]) {
		case "succeeded", "dead_letter", "failed", "cancelled":
			return status, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for %s", taskID)
}

func firstStatusSamples(statuses []map[string]any, limit int) []map[string]any {
	if limit > len(statuses) {
		limit = len(statuses)
	}
	out := make([]map[string]any, 0, limit)
	for idx := 0; idx < limit; idx++ {
		out = append(out, statuses[idx])
	}
	return out
}

func runParallelLocalSoak[T any, R any](workers int, inputs []T, fn func(T) (R, error)) ([]R, error) {
	if workers <= 0 {
		workers = 1
	}
	type item struct {
		index int
		value T
	}
	type result struct {
		index int
		value R
		err   error
	}
	jobs := make(chan item)
	results := make(chan result, len(inputs))
	var wg sync.WaitGroup
	for idx := 0; idx < workers; idx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				value, err := fn(job.value)
				results <- result{index: job.index, value: value, err: err}
			}
		}()
	}
	go func() {
		for idx, input := range inputs {
			jobs <- item{index: idx, value: input}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	ordered := make([]R, len(inputs))
	for result := range results {
		if result.err != nil {
			return nil, result.err
		}
		ordered[result.index] = result.value
	}
	return ordered, nil
}
