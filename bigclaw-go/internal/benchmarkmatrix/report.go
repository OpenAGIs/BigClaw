package benchmarkmatrix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var benchmarkLinePattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$`)

type Runner interface {
	Run(cmd *exec.Cmd) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

type Scenario struct {
	Count   int `json:"count"`
	Workers int `json:"workers"`
}

type SoakResult struct {
	Scenario   Scenario       `json:"scenario"`
	ReportPath string         `json:"report_path"`
	Result     map[string]any `json:"result"`
}

type Report struct {
	Benchmark  map[string]any `json:"benchmark"`
	SoakMatrix []SoakResult   `json:"soak_matrix"`
}

func ParseBenchmarkStdout(stdout string) map[string]map[string]float64 {
	parsed := make(map[string]map[string]float64)
	for _, rawLine := range strings.Split(stdout, "\n") {
		line := strings.TrimSpace(rawLine)
		match := benchmarkLinePattern.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		value, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			continue
		}
		parsed[match[1]] = map[string]float64{"ns_per_op": value}
	}
	return parsed
}

func ParseScenario(raw string) (Scenario, error) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return Scenario{}, fmt.Errorf("invalid scenario %q; expected count:workers", raw)
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil {
		return Scenario{}, fmt.Errorf("invalid scenario count %q: %w", parts[0], err)
	}
	workers, err := strconv.Atoi(parts[1])
	if err != nil {
		return Scenario{}, fmt.Errorf("invalid scenario workers %q: %w", parts[1], err)
	}
	return Scenario{Count: count, Workers: workers}, nil
}

func RunBenchmarks(goRoot string, runner Runner) (map[string]any, error) {
	cmd := exec.Command("go", "test", "-bench", ".", "./internal/queue", "./internal/scheduler")
	cmd.Dir = goRoot
	output, err := runner.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("run go benchmarks: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	stdout := string(output)
	return map[string]any{
		"stdout": stdout,
		"parsed": ParseBenchmarkStdout(stdout),
	}, nil
}

func RunSoak(goRoot string, scenario Scenario, timeoutSeconds int, reportPath string, runner Runner) (map[string]any, error) {
	cmd := exec.Command(
		"python3", "scripts/benchmark/soak_local.py",
		"--autostart",
		"--count", strconv.Itoa(scenario.Count),
		"--workers", strconv.Itoa(scenario.Workers),
		"--timeout-seconds", strconv.Itoa(timeoutSeconds),
		"--report-path", reportPath,
	)
	cmd.Dir = goRoot
	output, err := runner.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("run soak_local.py: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	body, err := os.ReadFile(filepath.Join(goRoot, reportPath))
	if err != nil {
		return nil, err
	}
	var report map[string]any
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}
	return report, nil
}

func BuildReport(goRoot string, scenarioArgs []string, timeoutSeconds int, runner Runner) (Report, error) {
	if runner == nil {
		runner = ExecRunner{}
	}
	scenarioStrings := scenarioArgs
	if len(scenarioStrings) == 0 {
		scenarioStrings = []string{"50:8", "100:12"}
	}
	benchmark, err := RunBenchmarks(goRoot, runner)
	if err != nil {
		return Report{}, err
	}
	soakMatrix := make([]SoakResult, 0, len(scenarioStrings))
	for _, raw := range scenarioStrings {
		scenario, err := ParseScenario(raw)
		if err != nil {
			return Report{}, err
		}
		reportPath := filepath.ToSlash(filepath.Join("docs/reports", fmt.Sprintf("soak-local-%dx%d.json", scenario.Count, scenario.Workers)))
		soakReport, err := RunSoak(goRoot, scenario, timeoutSeconds, reportPath, runner)
		if err != nil {
			return Report{}, err
		}
		soakMatrix = append(soakMatrix, SoakResult{
			Scenario:   scenario,
			ReportPath: reportPath,
			Result:     soakReport,
		})
	}
	return Report{Benchmark: benchmark, SoakMatrix: soakMatrix}, nil
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

type StubRunner struct {
	Outputs map[string][]byte
}

func (s StubRunner) Run(cmd *exec.Cmd) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(strings.Join(cmd.Args, " "))
	key := buf.String()
	if output, ok := s.Outputs[key]; ok {
		return output, nil
	}
	return nil, fmt.Errorf("unexpected command: %s", key)
}
