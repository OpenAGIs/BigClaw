package main

import "testing"

func TestParseBenchmarkStdout(t *testing.T) {
	stdout := `
BenchmarkMemoryQueueEnqueueLease-8        1000               321.50 ns/op
BenchmarkSchedulerDecide-8               20000                42.00 ns/op
PASS
`
	parsed := parseBenchmarkStdout(stdout)
	if len(parsed) != 2 {
		t.Fatalf("unexpected parsed benchmark count: %+v", parsed)
	}
	if parsed["BenchmarkMemoryQueueEnqueueLease-8"].NSPerOp != 321.50 {
		t.Fatalf("unexpected memory queue value: %+v", parsed)
	}
	if parsed["BenchmarkSchedulerDecide-8"].NSPerOp != 42.00 {
		t.Fatalf("unexpected scheduler value: %+v", parsed)
	}
}

func TestParseScenario(t *testing.T) {
	scenario, err := parseScenario("50:8")
	if err != nil {
		t.Fatalf("parse scenario: %v", err)
	}
	if scenario.Count != 50 || scenario.Workers != 8 {
		t.Fatalf("unexpected scenario: %+v", scenario)
	}
	if _, err := parseScenario("invalid"); err == nil {
		t.Fatal("expected invalid scenario error")
	}
}
