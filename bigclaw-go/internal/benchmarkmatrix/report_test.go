package benchmarkmatrix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseBenchmarkStdout(t *testing.T) {
	stdout := "BenchmarkQueuePush-8    12345    678.9 ns/op\nPASS\n"
	got := ParseBenchmarkStdout(stdout)
	want := map[string]map[string]float64{
		"BenchmarkQueuePush-8": {"ns_per_op": 678.9},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parsed benchmark output: got=%v want=%v", got, want)
	}
}

func TestParseScenario(t *testing.T) {
	got, err := ParseScenario("50:8")
	if err != nil {
		t.Fatalf("parse scenario: %v", err)
	}
	if got.Count != 50 || got.Workers != 8 {
		t.Fatalf("unexpected scenario: %+v", got)
	}
}

func TestBuildReport(t *testing.T) {
	goRoot := t.TempDir()
	report50x8 := filepath.Join(goRoot, "docs/reports/soak-local-50x8.json")
	if err := os.MkdirAll(filepath.Dir(report50x8), 0o755); err != nil {
		t.Fatalf("mkdir reports dir: %v", err)
	}
	if err := os.WriteFile(report50x8, []byte(`{"status":"ok"}`), 0o644); err != nil {
		t.Fatalf("write soak report: %v", err)
	}
	runner := StubRunner{Outputs: map[string][]byte{
		"go test -bench . ./internal/queue ./internal/scheduler": []byte("BenchmarkQueuePush-8 123 456.7 ns/op\n"),
		"go run ./scripts/benchmark/soak_local.go --autostart --go-root " + goRoot + " --count 50 --workers 8 --timeout-seconds 180 --report-path docs/reports/soak-local-50x8.json": []byte(""),
	}}
	report, err := BuildReport(goRoot, []string{"50:8"}, 180, runner)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if len(report.SoakMatrix) != 1 {
		t.Fatalf("unexpected soak matrix length: %d", len(report.SoakMatrix))
	}
	if report.SoakMatrix[0].ReportPath != "docs/reports/soak-local-50x8.json" {
		t.Fatalf("unexpected report path: %s", report.SoakMatrix[0].ReportPath)
	}
}

func TestWriteReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docs/reports/benchmark-matrix-report.json")
	report := Report{
		Benchmark: map[string]any{"stdout": "ok"},
		SoakMatrix: []SoakResult{
			{Scenario: Scenario{Count: 1, Workers: 2}, ReportPath: "docs/reports/soak.json", Result: map[string]any{"status": "ok"}},
		},
	}
	if err := WriteReport(path, report); err != nil {
		t.Fatalf("write report: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var decoded Report
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded.SoakMatrix[0].Scenario.Workers != 2 {
		t.Fatalf("unexpected decoded report: %+v", decoded)
	}
}
