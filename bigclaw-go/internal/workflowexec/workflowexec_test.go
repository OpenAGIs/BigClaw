package workflowexec

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
	"bigclaw-go/internal/workflow"
)

func TestAcceptanceGateRejectsMissingEvidence(t *testing.T) {
	task := domain.Task{
		ID:                 "BIG-403",
		Source:             "linear",
		Title:              "Close acceptance gate",
		Description:        "need validation evidence",
		Priority:           int(domain.PriorityP0),
		AcceptanceCriteria: []string{"report-shared"},
		ValidationPlan:     []string{"pytest"},
	}

	decision := workflow.AcceptanceGate{}.Evaluate(task, workflow.ExecutionOutcome{Approved: true, Status: "approved"}, []string{"pytest"}, nil, "")
	if decision.Passed || decision.Status != "rejected" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if !reflect.DeepEqual(decision.MissingAcceptanceCriteria, []string{"report-shared"}) {
		t.Fatalf("unexpected missing acceptance criteria: %+v", decision.MissingAcceptanceCriteria)
	}
	if len(decision.MissingValidationSteps) != 0 {
		t.Fatalf("unexpected missing validation steps: %+v", decision.MissingValidationSteps)
	}
}

func TestEngineRecordsJournalAndAcceptsCompleteEvidence(t *testing.T) {
	dir := t.TempDir()
	ledger := Ledger{Path: filepath.Join(dir, "ledger.json")}
	task := domain.Task{
		ID:                 "BIG-402",
		Source:             "linear",
		Title:              "Record workflow journal",
		Description:        "capture execution closure",
		Priority:           int(domain.PriorityP0),
		AcceptanceCriteria: []string{"report-shared"},
		ValidationPlan:     []string{"pytest"},
		RequiredTools:      []string{"browser"},
	}

	result, err := Engine{}.Run(
		task,
		"run-wf-1",
		ledger,
		"",
		filepath.Join(dir, "journals", "run-wf-1.json"),
		[]string{"pytest", "report-shared"},
		nil,
		"",
		filepath.Join(dir, "reports", "run-wf-1-orchestration.md"),
		"",
		nil,
		"",
		true,
		"main -> origin/main",
		"commit 123abc\n 3 files changed, 12 insertions(+)",
	)
	if err != nil {
		t.Fatalf("run workflow: %v", err)
	}

	if result.Execution.Decision.Medium != "browser" {
		t.Fatalf("unexpected medium: %+v", result.Execution.Decision)
	}
	if !result.Acceptance.Passed || result.Acceptance.Status != "accepted" {
		t.Fatalf("unexpected acceptance: %+v", result.Acceptance)
	}
	if result.JournalPath == "" || result.OrchestrationReportPath == "" {
		t.Fatalf("expected journal and orchestration report paths, got %+v", result)
	}

	journal := readJSONMap(t, result.JournalPath)
	if got := journalSteps(journal); !reflect.DeepEqual(got, []string{"intake", "execution", "orchestration", "acceptance", "closeout"}) {
		t.Fatalf("unexpected journal steps: %#v", got)
	}
	entries := journal["entries"].([]any)
	if entries[3].(map[string]any)["status"] != "accepted" {
		t.Fatalf("unexpected acceptance journal entry: %+v", entries[3])
	}
	if entries[4].(map[string]any)["status"] != "complete" {
		t.Fatalf("unexpected closeout journal entry: %+v", entries[4])
	}

	loaded, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if loaded[0]["closeout"].(map[string]any)["git_push_succeeded"] != true {
		t.Fatalf("expected git push success in closeout: %+v", loaded[0]["closeout"])
	}
}

func TestEngineKeepsHighRiskTaskPendingManualApproval(t *testing.T) {
	ledger := Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	task := domain.Task{
		ID:                 "BIG-403-risk",
		Source:             "linear",
		Title:              "Approve prod change",
		Description:        "manual gate",
		RiskLevel:          domain.RiskHigh,
		AcceptanceCriteria: []string{"rollback-plan"},
		ValidationPlan:     []string{"integration-test"},
	}

	result, err := Engine{}.Run(task, "run-wf-2", ledger, "", "", []string{"rollback-plan", "integration-test"}, nil, "", "", "", nil, "", false, "", "")
	if err != nil {
		t.Fatalf("run workflow: %v", err)
	}
	if result.Execution.Run.Status != "needs-approval" {
		t.Fatalf("unexpected run status: %+v", result.Execution.Run)
	}
	if result.Acceptance.Passed || result.Acceptance.Status != "needs-approval" {
		t.Fatalf("unexpected acceptance: %+v", result.Acceptance)
	}
}

func TestEngineWritesPilotScorecardAndAcceptsPositiveROI(t *testing.T) {
	dir := t.TempDir()
	ledger := Ledger{Path: filepath.Join(dir, "ledger.json")}
	task := domain.Task{
		ID:                 "OPE-60",
		Source:             "linear",
		Title:              "Pilot closeout",
		Description:        "capture KPI and ROI evidence",
		Priority:           int(domain.PriorityP0),
		AcceptanceCriteria: []string{"pilot-scorecard", "report-shared"},
		ValidationPlan:     []string{"pytest"},
	}
	benchmarkPassed := true
	benchmarkScore := 98
	scorecard := &PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Design Partner A",
		Period:             "2026-Q2",
		MonthlyBenefit:     15000,
		MonthlyCost:        3000,
		ImplementationCost: 18000,
		BenchmarkScore:     &benchmarkScore,
		BenchmarkPassed:    &benchmarkPassed,
		Metrics: []PilotMetric{
			{Name: "Automation coverage", Baseline: 30, Current: 81, Target: 80, Unit: "%", HigherIsBetter: true},
			{Name: "Review cycle time", Baseline: 10, Current: 4, Target: 5, Unit: "h", HigherIsBetter: false},
		},
	}

	result, err := Engine{}.Run(
		task,
		"run-wf-pilot-1",
		ledger,
		"",
		filepath.Join(dir, "journals", "run-wf-pilot-1.json"),
		[]string{"pytest", "report-shared", "pilot-scorecard"},
		scorecard,
		filepath.Join(dir, "reports", "pilot-scorecard.md"),
		"",
		"",
		nil,
		"",
		true,
		"main -> origin/main",
		"commit 456def\n 2 files changed, 9 insertions(+)",
	)
	if err != nil {
		t.Fatalf("run workflow: %v", err)
	}
	if !result.Acceptance.Passed || result.Acceptance.Status != "accepted" {
		t.Fatalf("unexpected acceptance: %+v", result.Acceptance)
	}
	if result.PilotReportPath == "" {
		t.Fatalf("expected pilot report path")
	}
	reportBody, err := os.ReadFile(result.PilotReportPath)
	if err != nil {
		t.Fatalf("read pilot report: %v", err)
	}
	if !strings.Contains(string(reportBody), "Annualized ROI") {
		t.Fatalf("expected ROI in pilot report: %s", string(reportBody))
	}

	journal := readJSONMap(t, result.JournalPath)
	if got := journalSteps(journal); !reflect.DeepEqual(got, []string{"intake", "execution", "pilot-scorecard", "acceptance", "closeout"}) {
		t.Fatalf("unexpected journal steps: %#v", got)
	}
	entries := journal["entries"].([]any)
	if entries[2].(map[string]any)["status"] != "go" {
		t.Fatalf("unexpected pilot journal entry: %+v", entries[2])
	}
}

func TestAcceptanceGateRejectsHoldPilotScorecard(t *testing.T) {
	task := domain.Task{
		ID:                 "OPE-60-hold",
		Source:             "linear",
		Title:              "Pilot hold decision",
		Description:        "scorecard blocks closure",
		AcceptanceCriteria: []string{"pilot-scorecard"},
		ValidationPlan:     []string{"pytest"},
	}
	benchmarkPassed := false
	scorecard := PilotScorecard{
		IssueID:            "OPE-60",
		Customer:           "Design Partner B",
		Period:             "2026-Q2",
		MonthlyBenefit:     1000,
		MonthlyCost:        2500,
		ImplementationCost: 8000,
		BenchmarkPassed:    &benchmarkPassed,
		Metrics: []PilotMetric{
			{Name: "Backlog aging", Baseline: 4, Current: 6, Target: 4, Unit: "d", HigherIsBetter: false},
		},
	}

	decision := workflow.AcceptanceGate{}.Evaluate(task, workflow.ExecutionOutcome{Approved: true, Status: "approved"}, []string{"pytest", "pilot-scorecard"}, nil, scorecard.Recommendation())
	if decision.Passed || decision.Status != "rejected" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if decision.Summary != "pilot scorecard indicates insufficient ROI or KPI progress" {
		t.Fatalf("unexpected summary: %+v", decision)
	}
}

func TestEngineWritesOrchestrationReportWithoutDuplicatingLedgerEntries(t *testing.T) {
	dir := t.TempDir()
	ledger := Ledger{Path: filepath.Join(dir, "ledger.json")}
	task := domain.Task{
		ID:                 "OPE-66-workflow",
		Source:             "linear",
		Title:              "Coordinate customer rollout",
		Description:        "Need browser and analytics support",
		Labels:             []string{"customer", "data"},
		Priority:           int(domain.PriorityP0),
		RequiredTools:      []string{"browser", "sql"},
		AcceptanceCriteria: []string{"report-shared"},
		ValidationPlan:     []string{"pytest"},
	}

	result, err := Engine{}.Run(
		task,
		"run-wf-ope-66",
		ledger,
		"",
		filepath.Join(dir, "journals", "run-wf-ope-66.json"),
		[]string{"pytest", "report-shared"},
		nil,
		"",
		filepath.Join(dir, "reports", "run-wf-ope-66-orchestration.md"),
		filepath.Join(dir, "reports", "run-wf-ope-66-canvas.md"),
		nil,
		"",
		true,
		"main -> origin/main",
		"commit 789fed\n 4 files changed, 16 insertions(+)",
	)
	if err != nil {
		t.Fatalf("run workflow: %v", err)
	}
	reportBody, err := os.ReadFile(result.OrchestrationReportPath)
	if err != nil {
		t.Fatalf("read orchestration report: %v", err)
	}
	if strings.Contains(string(reportBody), "- customer-success:") {
		t.Fatalf("expected tier-limited plan to omit blocked department handoff: %s", string(reportBody))
	}
	if !strings.Contains(string(reportBody), "Upgrade Required: true") {
		t.Fatalf("expected upgrade requirement in report: %s", string(reportBody))
	}
	if !strings.Contains(string(reportBody), "Human Handoff Team: operations") {
		t.Fatalf("expected operations handoff in report: %s", string(reportBody))
	}
	canvasBody, err := os.ReadFile(result.OrchestrationCanvasPath)
	if err != nil {
		t.Fatalf("read orchestration canvas: %v", err)
	}
	if !strings.Contains(string(canvasBody), "# Orchestration Canvas") || !strings.Contains(string(canvasBody), "Recommendation: resolve-entitlement-gap") {
		t.Fatalf("unexpected canvas: %s", string(canvasBody))
	}

	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", len(entries))
	}
	artifacts := entries[0]["artifacts"].([]any)
	if artifacts[0].(map[string]any)["name"] != "cross-department-orchestration" || artifacts[1].(map[string]any)["name"] != "orchestration-canvas" {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
	journal := readJSONMap(t, result.JournalPath)
	if got := journalSteps(journal); !reflect.DeepEqual(got, []string{"intake", "execution", "orchestration", "acceptance", "closeout"}) {
		t.Fatalf("unexpected journal steps: %#v", got)
	}
}

func TestEngineWritesRepoSyncAuditReportAndRecordsFailureCategories(t *testing.T) {
	dir := t.TempDir()
	ledger := Ledger{Path: filepath.Join(dir, "ledger.json")}
	task := domain.Task{
		ID:                 "OPE-219",
		Source:             "linear",
		Title:              "Audit repo sync",
		Description:        "capture sync failures and pr freshness",
		Priority:           int(domain.PriorityP1),
		AcceptanceCriteria: []string{"repo-sync-audit", "report-shared"},
		ValidationPlan:     []string{"pytest"},
	}
	prNumber := 219
	repoSyncAudit := &RepoSyncAudit{
		Sync: GitSyncTelemetry{
			Status:          "failed",
			FailureCategory: "divergence",
			Summary:         "branch diverged from remote",
			Branch:          "dcjcloud/ope-219",
			RemoteRef:       "origin/dcjcloud/ope-219",
			AheadBy:         2,
			BehindBy:        1,
		},
		PullRequest: PullRequestFreshness{
			PRNumber:           &prNumber,
			PRURL:              "https://github.com/OpenAGIs/BigClaw/pull/219",
			BranchState:        "out-of-sync",
			BodyState:          "drifted",
			BranchHeadSHA:      "abc123",
			PRHeadSHA:          "def456",
			ExpectedBodyDigest: "expected",
			ActualBodyDigest:   "actual",
		},
	}

	result, err := Engine{}.Run(
		task,
		"run-wf-ope-219",
		ledger,
		"",
		filepath.Join(dir, "journals", "run-wf-ope-219.json"),
		[]string{"pytest", "report-shared", "repo-sync-audit"},
		nil,
		"",
		"",
		"",
		repoSyncAudit,
		filepath.Join(dir, "reports", "run-wf-ope-219-repo-sync.md"),
		true,
		"feature/OPE-219 -> origin/feature/OPE-219",
		"commit abc123\n 3 files changed, 18 insertions(+)",
	)
	if err != nil {
		t.Fatalf("run workflow: %v", err)
	}
	if !result.Acceptance.Passed || result.RepoSyncReportPath == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
	reportBody, err := os.ReadFile(result.RepoSyncReportPath)
	if err != nil {
		t.Fatalf("read repo sync report: %v", err)
	}
	if !strings.Contains(string(reportBody), "Failure Category: divergence") || !strings.Contains(string(reportBody), "Body State: drifted") {
		t.Fatalf("unexpected repo sync report: %s", string(reportBody))
	}
	journal := readJSONMap(t, result.JournalPath)
	if got := journalSteps(journal); !reflect.DeepEqual(got, []string{"intake", "execution", "repo-sync", "acceptance", "closeout"}) {
		t.Fatalf("unexpected journal steps: %#v", got)
	}
	entries := journal["entries"].([]any)
	if entries[2].(map[string]any)["details"].(map[string]any)["failure_category"] != "divergence" {
		t.Fatalf("unexpected repo sync journal details: %+v", entries[2])
	}
	loaded, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	auditActions := make([]string, 0)
	for _, audit := range loaded[0]["audits"].([]any) {
		auditActions = append(auditActions, audit.(map[string]any)["action"].(string))
	}
	if !contains(auditActions, "repo.sync") || !contains(auditActions, "repo.pr-freshness") {
		t.Fatalf("unexpected audit actions: %+v", auditActions)
	}
	artifacts := loaded[0]["artifacts"].([]any)
	if artifacts[0].(map[string]any)["name"] != "repo-sync-audit" {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
	closeout := loaded[0]["closeout"].(map[string]any)
	sync := closeout["repo_sync_audit"].(map[string]any)["sync"].(map[string]any)
	if sync["failure_category"] != "divergence" {
		t.Fatalf("unexpected closeout repo sync audit: %+v", closeout)
	}
}

func TestTaskRunCapturesArtifactsAuditsCloseoutAndLedgerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "validation.md")
	if err := os.WriteFile(artifactPath, []byte("validation ok"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	sum := sha256.Sum256([]byte("validation ok"))
	expectedDigest := hex.EncodeToString(sum[:])

	run := NewTaskRun(domain.Task{
		ID:       "BIG-502",
		Source:   "linear",
		Title:    "Add observability",
		Priority: int(domain.PriorityP0),
	}, "run-1", "docker")
	run.Log("info", "task accepted", map[string]any{"queue": "primary"})
	run.Trace("scheduler.decide", "ok", map[string]any{"approved": true})
	if err := run.RegisterArtifact("validation-report", "report", artifactPath, map[string]any{"environment": "sandbox"}); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	run.Audit("scheduler.approved", "system", "success", map[string]any{"reason": "default low risk path"})
	run.RecordCloseout([]string{"pytest", "validation-report"}, true, "Everything up-to-date", "commit abc123\n 1 file changed, 2 insertions(+)", nil)
	run.Finalize("succeeded", "validation passed")

	ledger := Ledger{Path: filepath.Join(dir, "observability.json")}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("upsert ledger: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 ledger entry, got %+v", entries)
	}
	if entries[0]["status"] != "succeeded" {
		t.Fatalf("unexpected status: %+v", entries[0])
	}
	if entries[0]["logs"].([]any)[0].(map[string]any)["context"].(map[string]any)["queue"] != "primary" {
		t.Fatalf("unexpected logs: %+v", entries[0]["logs"])
	}
	if entries[0]["traces"].([]any)[0].(map[string]any)["attributes"].(map[string]any)["approved"] != true {
		t.Fatalf("unexpected traces: %+v", entries[0]["traces"])
	}
	if entries[0]["artifacts"].([]any)[0].(map[string]any)["sha256"] != expectedDigest {
		t.Fatalf("unexpected artifacts: %+v", entries[0]["artifacts"])
	}
	actions := []string{}
	for _, item := range entries[0]["audits"].([]any) {
		actions = append(actions, item.(map[string]any)["action"].(string))
	}
	for _, want := range []string{"artifact.registered", "closeout.recorded", "scheduler.approved"} {
		if !contains(actions, want) {
			t.Fatalf("missing audit action %q in %+v", want, actions)
		}
	}
	if entries[0]["closeout"].(map[string]any)["complete"] != true {
		t.Fatalf("unexpected closeout: %+v", entries[0]["closeout"])
	}
}

func TestTaskRunCloseoutSerializesRepoSyncAudit(t *testing.T) {
	dir := t.TempDir()
	prNumber := 219
	run := NewTaskRun(domain.Task{ID: "BIG-sync", Source: "linear", Title: "Repo sync closeout"}, "run-sync", "docker")
	repoSyncAudit := &RepoSyncAudit{
		Sync: GitSyncTelemetry{
			Status:          "failed",
			FailureCategory: "dirty",
			Summary:         "worktree has local changes",
			Branch:          "feature/OPE-219",
			RemoteRef:       "origin/feature/OPE-219",
			DirtyPaths:      []string{"src/bigclaw/workflow.py"},
		},
		PullRequest: PullRequestFreshness{
			PRNumber:           &prNumber,
			PRURL:              "https://github.com/OpenAGIs/BigClaw/pull/219",
			BranchState:        "out-of-sync",
			BodyState:          "drifted",
			BranchHeadSHA:      "abc123",
			PRHeadSHA:          "def456",
			ExpectedBodyDigest: "body-expected",
			ActualBodyDigest:   "body-actual",
		},
	}
	run.RecordCloseout([]string{"pytest"}, false, "push rejected", "commit abc123\n 1 file changed, 2 insertions(+)", repoSyncAudit)

	ledger := Ledger{Path: filepath.Join(dir, "observability.json")}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("upsert ledger: %v", err)
	}
	loadedRuns, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if loadedRuns[0].Closeout.RepoSyncAudit == nil {
		t.Fatalf("expected repo sync audit in closeout: %+v", loadedRuns[0].Closeout)
	}
	if loadedRuns[0].Closeout.RepoSyncAudit.Sync.FailureCategory != "dirty" || loadedRuns[0].Closeout.RepoSyncAudit.PullRequest.BodyState != "drifted" {
		t.Fatalf("unexpected repo sync audit: %+v", loadedRuns[0].Closeout.RepoSyncAudit)
	}
}

func TestRenderTaskRunReportAndDetailPage(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "artifact.txt")
	if err := os.WriteFile(artifactPath, []byte("audit trail"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	run := NewTaskRun(domain.Task{ID: "BIG-502", Source: "linear", Title: "Observe execution"}, "run-3", "browser")
	run.Log("info", "opened detail page", nil)
	run.Trace("playback.render", "ok", nil)
	if err := run.RegisterArtifact("approval-note", "note", artifactPath, nil); err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	run.Audit("playback.render", "reviewer", "success", nil)
	run.AddComment("pm", "Loop in @design before we publish the replay.", []string{"design"}, "overview")
	run.AddDecisionNote("design", "Replay copy approved for external review.", "approved", []string{"pm"}, nil, "")
	run.RecordCloseout([]string{"pytest", "playback-smoke"}, true, "main -> origin/main", "commit fedcba\n 1 file changed, 1 insertion(+)", nil)
	run.Closeout.RunCommitLinks = []repo.RunCommitLink{
		{RunID: "run-3", CommitHash: "abc111", Role: "candidate", RepoSpaceID: "space-1"},
		{RunID: "run-3", CommitHash: "fedcba", Role: "accepted", RepoSpaceID: "space-1"},
	}
	run.Finalize("approved", "detail page ready")

	report := RenderTaskRunReport(run)
	for _, fragment := range []string{
		"Run ID: run-3",
		"## Logs",
		"## Trace",
		"## Artifacts",
		"## Audit",
		"## Closeout",
		"Git Push Succeeded: true",
		"## Actions",
		"Retry [retry] state=disabled target=run-3 reason=retry is available for failed or approval-blocked runs",
		"## Collaboration",
		"Loop in @design before we publish the replay.",
		"Replay copy approved for external review.",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report:\n%s", fragment, report)
		}
	}

	page := RenderTaskRunDetailPage(run)
	for _, fragment := range []string{
		"<title>Task Run Detail",
		"Timeline / Log Sync",
		"data-detail=\"title\"",
		"Reports",
		"opened detail page",
		"playback.render",
		artifactPath,
		"detail page ready",
		"Closeout",
		"complete",
		"Repo Evidence",
		"fedcba",
		"Actions",
		"Pause [pause] state=disabled target=run-3 reason=completed or failed runs cannot be paused",
		"Collaboration",
		"Loop in @design before we publish the replay.",
		"Replay copy approved for external review.",
	} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("expected %q in detail page:\n%s", fragment, page)
		}
	}
}

func TestRenderTaskRunDetailPageEscapesScriptBreakoutAndSynthesizesCollaborationThread(t *testing.T) {
	dir := t.TempDir()
	ledger := Ledger{Path: filepath.Join(dir, "observability.json")}
	run := NewTaskRun(domain.Task{ID: "BIG-502-roundtrip", Source: "linear", Title: "Round trip"}, "run-roundtrip", "docker")
	run.Log("info", "contains </script> marker", nil)
	run.Trace("scheduler.decide", "ok", nil)
	run.Audit("scheduler.decision", "scheduler", "approved", map[string]any{"reason": "default low risk path"})
	run.AddComment("ops", "Need @eng confirmation on the retry plan.", []string{"eng"}, "timeline")
	run.Finalize("approved", "default low risk path")
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("upsert ledger: %v", err)
	}

	page := RenderTaskRunDetailPage(run)
	if !strings.Contains(page, "contains &lt;\\/script&gt; marker") {
		t.Fatalf("expected escaped script marker in page:\n%s", page)
	}

	loadedRuns, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	thread := BuildCollaborationThreadFromAudits(loadedRuns[0].Audits, "run", loadedRuns[0].RunID)
	if thread == nil || thread.MentionCount != 1 {
		t.Fatalf("unexpected collaboration thread: %+v", thread)
	}
	if len(thread.Comments) != 1 || thread.Comments[0].Body != "Need @eng confirmation on the retry plan." {
		t.Fatalf("unexpected collaboration comments: %+v", thread)
	}
}

func readJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return payload
}

func journalSteps(journal map[string]any) []string {
	entries := journal["entries"].([]any)
	steps := make([]string, 0, len(entries))
	for _, entry := range entries {
		steps = append(steps, entry.(map[string]any)["step"].(string))
	}
	return steps
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
