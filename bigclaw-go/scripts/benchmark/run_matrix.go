package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var benchmarkLinePattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$`)

type benchmarkParsed struct {
	NSPerOp float64 `json:"ns_per_op"`
}

type benchmarkResult struct {
	Stdout string                     `json:"stdout"`
	Parsed map[string]benchmarkParsed `json:"parsed"`
}

type soakScenario struct {
	Count   int `json:"count"`
	Workers int `json:"workers"`
}

type soakMatrixEntry struct {
	Scenario   soakScenario    `json:"scenario"`
	ReportPath string          `json:"report_path"`
	Result     json.RawMessage `json:"result"`
}

type matrixReport struct {
	Benchmark  benchmarkResult   `json:"benchmark"`
	SoakMatrix []soakMatrixEntry `json:"soak_matrix"`
}

func parseBenchmarkStdout(stdout string) map[string]benchmarkParsed {
	parsed := make(map[string]benchmarkParsed)
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		matches := benchmarkLinePattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		value, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			continue
		}
		parsed[matches[1]] = benchmarkParsed{NSPerOp: value}
	}
	return parsed
}

func runBenchmarks(goRoot string) (benchmarkResult, error) {
	cmd := exec.Command("go", "test", "-bench", ".", "./internal/queue", "./internal/scheduler")
	cmd.Dir = goRoot
	output, err := cmd.Output()
	if err != nil {
		return benchmarkResult{}, err
	}
	stdout := string(output)
	return benchmarkResult{
		Stdout: stdout,
		Parsed: parseBenchmarkStdout(stdout),
	}, nil
}

func runSoak(goRoot string, count int, workers int, timeoutSeconds int, reportPath string) (json.RawMessage, error) {
	cmd := exec.Command(
		"go", "run", "./cmd/bigclawctl", "automation", "benchmark", "soak-local",
		"--autostart",
		"--count", strconv.Itoa(count),
		"--workers", strconv.Itoa(workers),
		"--timeout-seconds", strconv.Itoa(timeoutSeconds),
		"--report-path", reportPath,
	)
	cmd.Dir = goRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("run soak-local: %w (%s)", err, output)
	}
	body, err := os.ReadFile(filepath.Join(goRoot, reportPath))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func parseScenario(value string) (soakScenario, error) {
	countValue, workersValue, ok := strings.Cut(value, ":")
	if !ok {
		return soakScenario{}, fmt.Errorf("invalid scenario %q", value)
	}
	count, err := strconv.Atoi(countValue)
	if err != nil {
		return soakScenario{}, fmt.Errorf("parse count from %q: %w", value, err)
	}
	workers, err := strconv.Atoi(workersValue)
	if err != nil {
		return soakScenario{}, fmt.Errorf("parse workers from %q: %w", value, err)
	}
	return soakScenario{Count: count, Workers: workers}, nil
}

type scenarioList []string

func (s *scenarioList) String() string {
	return strings.Join(*s, ",")
}

func (s *scenarioList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	defaultRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flags := flag.NewFlagSet("run-matrix", flag.ExitOnError)
	goRoot := flags.String("go-root", defaultRoot, "bigclaw-go repo root")
	reportPath := flags.String("report-path", "docs/reports/benchmark-matrix-report.json", "report path")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "timeout seconds")
	var scenarios scenarioList
	flags.Var(&scenarios, "scenario", "count:workers scenario; default 50:8 and 100:12")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if len(scenarios) == 0 {
		scenarios = scenarioList{"50:8", "100:12"}
	}

	benchmark, err := runBenchmarks(*goRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	entries := make([]soakMatrixEntry, 0, len(scenarios))
	for _, item := range scenarios {
		scenario, err := parseScenario(item)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		scenarioReportPath := filepath.ToSlash(filepath.Join("docs", "reports", fmt.Sprintf("soak-local-%dx%d.json", scenario.Count, scenario.Workers)))
		result, err := runSoak(*goRoot, scenario.Count, scenario.Workers, *timeoutSeconds, scenarioReportPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		entries = append(entries, soakMatrixEntry{
			Scenario:   scenario,
			ReportPath: scenarioReportPath,
			Result:     result,
		})
	}

	report := matrixReport{
		Benchmark:  benchmark,
		SoakMatrix: entries,
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	outputPath := filepath.Join(*goRoot, *reportPath)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputPath, body, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(body))
}
