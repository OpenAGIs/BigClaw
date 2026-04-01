package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
)

func TestRunDetailReportContractIncludesCloseoutArtifactsAndAuditNotes(t *testing.T) {
	recorder := observability.NewRecorder()
	controller := control.New()
	base := time.Unix(1700010000, 0)
	task := domain.Task{
		ID:                 "task-contract-report",
		TraceID:            "trace-contract-report",
		Title:              "Observe execution",
		State:              domain.TaskSucceeded,
		Priority:           1,
		AcceptanceCriteria: []string{"ship report"},
		ValidationPlan:     []string{"replay trace"},
		Metadata: map[string]string{
			"team":                "platform",
			"project":             "alpha",
			"plan":                "premium",
			"workpad":             "https://docs.example.com/workpads/task-contract-report",
			"validation_evidence": `["go test ./internal/api","playback-smoke"]`,
			"git_push_succeeded":  "true",
			"git_push_output":     "main -> origin/main",
			"git_log_stat_output": "commit fedcba\n 1 file changed, 1 insertion(+)",
			"remote_synced":       "true",
			"local_sha":           "fedcba",
			"remote_sha":          "fedcba",
		},
		CreatedAt: base,
		UpdatedAt: base.Add(3 * time.Second),
	}
	recorder.StoreTask(task)
	recorder.Record(domain.Event{
		ID:        "evt-started-contract",
		Type:      domain.EventTaskStarted,
		TaskID:    task.ID,
		TraceID:   task.TraceID,
		RunID:     "run-contract-1",
		Timestamp: base.Add(time.Second),
		Payload:   map[string]any{"executor": domain.ExecutorKubernetes, "required_tools": []string{"browser"}},
	})
	recorder.Record(domain.Event{
		ID:        "evt-completed-contract",
		Type:      domain.EventTaskCompleted,
		TaskID:    task.ID,
		TraceID:   task.TraceID,
		RunID:     "run-contract-1",
		Timestamp: base.Add(2 * time.Second),
		Payload: map[string]any{
			"executor":     domain.ExecutorKubernetes,
			"message":      "detail page ready",
			"artifacts":    []string{"https://docs.example.com/reports/task-contract-report.md"},
			"report_path":  "reports/task-contract-report/run-contract-1.md",
			"journal_path": "journals/platform/run-contract-1.json",
		},
	})
	controller.Takeover(task.ID, "pm", "design", "Loop in @design before we publish the replay.", base.Add(4*time.Second))
	recorder.Record(domain.Event{
		ID:        "evt-takeover-contract",
		Type:      domain.EventRunTakeover,
		TaskID:    task.ID,
		TraceID:   task.TraceID,
		Timestamp: base.Add(4 * time.Second),
		Payload:   map[string]any{"actor": "pm", "reviewer": "design", "note": "Loop in @design before we publish the replay.", "team": "platform", "project": "alpha"},
	})

	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Control: controller, Now: func() time.Time { return base }}
	handler := server.Handler()

	runResponse := httptest.NewRecorder()
	handler.ServeHTTP(runResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-contract-report?limit=20", nil))
	if runResponse.Code != http.StatusOK {
		t.Fatalf("expected run detail 200, got %d %s", runResponse.Code, runResponse.Body.String())
	}
	var decoded struct {
		Artifacts    map[string]string `json:"artifacts"`
		AuditSummary struct {
			Total      int `json:"total"`
			NotesCount int `json:"notes_count"`
		} `json:"audit_summary"`
		Closeout struct {
			ValidationEvidence []string `json:"validation_evidence"`
			GitPushSucceeded   bool     `json:"git_push_succeeded"`
			GitPushOutput      string   `json:"git_push_output"`
			GitLogStatOutput   string   `json:"git_log_stat_output"`
			RemoteSynced       bool     `json:"remote_synced"`
			LocalSHA           string   `json:"local_sha"`
			RemoteSHA          string   `json:"remote_sha"`
			Complete           bool     `json:"complete"`
		} `json:"closeout"`
	}
	if err := json.Unmarshal(runResponse.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode run detail: %v", err)
	}
	if decoded.Artifacts["report"] != "/v2/runs/task-contract-report/report?limit=20" || decoded.Artifacts["audit"] != "/v2/runs/task-contract-report/audit?limit=20" {
		t.Fatalf("expected run detail artifact links, got %+v", decoded.Artifacts)
	}
	if decoded.AuditSummary.Total != 1 || decoded.AuditSummary.NotesCount != 1 {
		t.Fatalf("expected single takeover note in audit summary, got %+v", decoded.AuditSummary)
	}
	if len(decoded.Closeout.ValidationEvidence) != 2 || !decoded.Closeout.GitPushSucceeded || decoded.Closeout.GitPushOutput != "main -> origin/main" || decoded.Closeout.GitLogStatOutput == "" || !decoded.Closeout.RemoteSynced || decoded.Closeout.LocalSHA != "fedcba" || decoded.Closeout.RemoteSHA != "fedcba" || !decoded.Closeout.Complete {
		t.Fatalf("unexpected closeout payload: %+v", decoded.Closeout)
	}

	reportResponse := httptest.NewRecorder()
	handler.ServeHTTP(reportResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-contract-report/report?limit=20", nil))
	if reportResponse.Code != http.StatusOK {
		t.Fatalf("expected run report 200, got %d %s", reportResponse.Code, reportResponse.Body.String())
	}
	report := reportResponse.Body.String()
	for _, want := range []string{
		"# BigClaw Run Report",
		"## Closeout",
		"- Validation Evidence: go test ./internal/api",
		"- Validation Evidence: playback-smoke",
		"- Git Push Succeeded: true",
		"- SHA Pair: local=fedcba remote=fedcba",
		"## Audit",
		"Loop in @design before we publish the replay.",
		"## Timeline",
		"detail page ready",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in run report, got %s", want, report)
		}
	}
}
