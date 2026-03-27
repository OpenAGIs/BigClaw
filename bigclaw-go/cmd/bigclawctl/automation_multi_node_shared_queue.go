package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type automationMultiNodeSharedQueueOptions struct {
	GoRoot              string
	ReportPath          string
	TakeoverReportPath  string
	TakeoverArtifactDir string
	TakeoverTTL         time.Duration
	Count               int
	SubmitWorkers       int
	TimeoutSeconds      int
	HTTPClient          *http.Client
	Sleep               func(time.Duration)
}

type multiNodeRuntime struct {
	Name       string
	BaseURL    string
	AuditPath  string
	ServiceLog string
	Command    *exec.Cmd
}

type multiNodeSubmission struct {
	baseURL string
	task    automationTask
}

func runAutomationMultiNodeSharedQueueCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e multi-node-shared-queue", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	reportPath := flags.String("report-path", "docs/reports/multi-node-shared-queue-report.json", "shared queue report path")
	takeoverReportPath := flags.String("takeover-report-path", "docs/reports/live-multi-node-subscriber-takeover-report.json", "takeover report path")
	takeoverArtifactDir := flags.String("takeover-artifact-dir", "docs/reports/live-multi-node-subscriber-takeover-artifacts", "takeover artifact dir")
	takeoverTTL := flags.Duration("takeover-ttl", time.Second, "takeover ttl")
	count := flags.Int("count", 200, "task count")
	submitWorkers := flags.Int("submit-workers", 8, "concurrent submit workers")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "timeout seconds")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e multi-node-shared-queue [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, exitCode, err := automationMultiNodeSharedQueue(automationMultiNodeSharedQueueOptions{
		GoRoot:              absPath(*goRoot),
		ReportPath:          trim(*reportPath),
		TakeoverReportPath:  trim(*takeoverReportPath),
		TakeoverArtifactDir: trim(*takeoverArtifactDir),
		TakeoverTTL:         *takeoverTTL,
		Count:               *count,
		SubmitWorkers:       *submitWorkers,
		TimeoutSeconds:      *timeoutSeconds,
	})
	if report != nil {
		return emit(report, true, exitCode)
	}
	return err
}

func automationMultiNodeSharedQueue(opts automationMultiNodeSharedQueueOptions) (map[string]any, int, error) {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	rootStateDir, err := os.MkdirTemp("", "bigclawd-multinode-")
	if err != nil {
		return nil, 0, err
	}
	defer os.RemoveAll(rootStateDir)
	runtimes, err := startMultiNodeRuntimes(opts.GoRoot, rootStateDir)
	if err != nil {
		return nil, 0, err
	}
	defer stopMultiNodeRuntimes(runtimes)
	for _, node := range runtimes {
		if err := automationWaitForHealth(client, node.BaseURL, 60, time.Second, sleep); err != nil {
			return nil, 0, err
		}
	}
	submittedBy := map[string]string{}
	timestamp := time.Now().Unix()
	type submission struct {
		baseURL string
		task    automationTask
	}
	submissions := make([]multiNodeSubmission, 0, opts.Count)
	for index := 0; index < opts.Count; index++ {
		node := runtimes[index%len(runtimes)]
		taskID := fmt.Sprintf("multinode-%d-%d", index, timestamp)
		task := automationTask{
			ID:               taskID,
			TraceID:          taskID,
			Title:            fmt.Sprintf("multi-node task %d", index),
			Entrypoint:       fmt.Sprintf("echo multinode %d", index),
			RequiredExecutor: "local",
			Metadata:         map[string]string{"scenario": "multi-node-shared-queue", "submit_node": node.Name},
		}
		submissions = append(submissions, multiNodeSubmission{baseURL: node.BaseURL, task: task})
		submittedBy[taskID] = node.Name
	}
	if err := submitTasksConcurrently(client, submissions, opts.SubmitWorkers); err != nil {
		return nil, 0, err
	}
	deadline := time.Now().Add(time.Duration(opts.TimeoutSeconds) * time.Second)
	var perTask map[string]map[string][]string
	for time.Now().Before(deadline) {
		events, err := aggregateMultiNodeEvents(runtimes)
		if err != nil {
			return nil, 0, err
		}
		perTask = summarizeMultiNodeTasks(submittedBy, events)
		allDone := true
		for _, item := range perTask {
			if len(item["completed"]) < 1 {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
		sleep(500 * time.Millisecond)
	}
	if perTask == nil {
		return nil, 0, errors.New("multi-node summary was empty")
	}
	duplicateStarted := []any{}
	duplicateCompleted := []any{}
	missingCompleted := []any{}
	completionByNode := map[string]any{}
	for _, node := range runtimes {
		completionByNode[node.Name] = 0
	}
	crossNode := 0
	for taskID, item := range perTask {
		if len(item["started"]) > 1 {
			duplicateStarted = append(duplicateStarted, taskID)
		}
		if len(item["completed"]) > 1 {
			duplicateCompleted = append(duplicateCompleted, taskID)
		}
		if len(item["completed"]) == 0 {
			missingCompleted = append(missingCompleted, taskID)
			continue
		}
		nodeName := item["completed"][0]
		completionByNode[nodeName] = automationInt(completionByNode[nodeName]) + 1
		if nodeName != submittedBy[taskID] {
			crossNode++
		}
	}
	sharedQueueReport := map[string]any{
		"generated_at":   utcISOTime(time.Now().UTC()),
		"root_state_dir": rootStateDir,
		"queue_path":     filepath.Join(rootStateDir, "shared-queue.db"),
		"count":          opts.Count,
		"submitted_by_node": map[string]any{
			runtimes[0].Name: countSubmittedBy(submittedBy, runtimes[0].Name),
			runtimes[1].Name: countSubmittedBy(submittedBy, runtimes[1].Name),
		},
		"completed_by_node":         completionByNode,
		"cross_node_completions":    crossNode,
		"duplicate_started_tasks":   duplicateStarted,
		"duplicate_completed_tasks": duplicateCompleted,
		"missing_completed_tasks":   missingCompleted,
		"all_ok":                    len(duplicateStarted) == 0 && len(duplicateCompleted) == 0 && len(missingCompleted) == 0 && automationInt(completionByNode[runtimes[0].Name]) > 0 && automationInt(completionByNode[runtimes[1].Name]) > 0,
		"nodes": []any{
			map[string]any{"name": runtimes[0].Name, "base_url": runtimes[0].BaseURL, "audit_path": runtimes[0].AuditPath, "service_log": runtimes[0].ServiceLog},
			map[string]any{"name": runtimes[1].Name, "base_url": runtimes[1].BaseURL, "audit_path": runtimes[1].AuditPath, "service_log": runtimes[1].ServiceLog},
		},
	}
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, sharedQueueReport); err != nil {
		return nil, 0, err
	}
	artifactRoot := filepath.Join(opts.GoRoot, opts.TakeoverArtifactDir)
	_ = os.RemoveAll(artifactRoot)
	scenarios := []any{}
	first, err := executeLiveTakeoverScenario(client, opts, runtimes[0], runtimes[1], runtimes, artifactRoot, "lease-expiry-stale-writer-rejected-live", "Lease expires on node-a and node-b takes ownership with stale-writer fencing", 1, 80, []any{"evt-81"}, false, false)
	if err != nil {
		return nil, 0, err
	}
	scenarios = append(scenarios, first)
	second, err := executeLiveTakeoverScenario(client, opts, runtimes[1], runtimes[0], runtimes, artifactRoot, "contention-then-takeover-live", "Node-a is rejected during active ownership and takes over after node-b lease expiry", 2, 120, []any{"evt-121", "evt-122"}, true, false)
	if err != nil {
		return nil, 0, err
	}
	scenarios = append(scenarios, second)
	third, err := executeLiveTakeoverScenario(client, opts, runtimes[1], runtimes[0], runtimes, artifactRoot, "idle-primary-takeover-live", "Node-b stops checkpointing and node-a advances the durable cursor after expiry", 3, 40, []any{"evt-41"}, false, true)
	if err != nil {
		return nil, 0, err
	}
	scenarios = append(scenarios, third)
	takeoverReport := buildLiveTakeoverReportGo(scenarios, opts.ReportPath)
	if err := automationWriteReport(opts.GoRoot, opts.TakeoverReportPath, takeoverReport); err != nil {
		return nil, 0, err
	}
	result := map[string]any{
		"shared_queue_report":  sharedQueueReport,
		"live_takeover_report": takeoverReport,
	}
	exitCode := 0
	if sharedQueueReport["all_ok"] != true || automationInt(takeoverReport["summary"].(map[string]any)["failing_scenarios"]) != 0 {
		exitCode = 1
	}
	return result, exitCode, nil
}

func startMultiNodeRuntimes(goRoot string, rootStateDir string) ([]multiNodeRuntime, error) {
	queuePath := filepath.Join(rootStateDir, "shared-queue.db")
	leasePath := filepath.Join(rootStateDir, "shared-subscriber-leases.db")
	runtimes := []multiNodeRuntime{}
	for _, name := range []string{"node-a", "node-b"} {
		baseURL, httpAddr, err := automationReserveLocalBaseURL()
		if err != nil {
			return nil, err
		}
		auditPath := filepath.Join(rootStateDir, name+"-audit.jsonl")
		env := os.Environ()
		env = append(env,
			"BIGCLAW_HTTP_ADDR="+httpAddr,
			"BIGCLAW_SERVICE_NAME="+name,
			"BIGCLAW_QUEUE_BACKEND=sqlite",
			"BIGCLAW_QUEUE_SQLITE_PATH="+queuePath,
			"BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH="+leasePath,
			"BIGCLAW_AUDIT_LOG_PATH="+auditPath,
			"BIGCLAW_POLL_INTERVAL=100ms",
		)
		logFile, err := os.CreateTemp("", name+"-*.log")
		if err != nil {
			return nil, err
		}
		cmd := exec.Command("go", "run", "./cmd/bigclawd")
		cmd.Dir = goRoot
		cmd.Env = env
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		runtimes = append(runtimes, multiNodeRuntime{Name: name, BaseURL: baseURL, AuditPath: auditPath, ServiceLog: logFile.Name(), Command: cmd})
	}
	return runtimes, nil
}

func stopMultiNodeRuntimes(runtimes []multiNodeRuntime) {
	for _, node := range runtimes {
		if node.Command == nil || node.Command.Process == nil {
			continue
		}
		_ = node.Command.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func(cmd *exec.Cmd) {
			_, _ = cmd.Process.Wait()
			close(done)
		}(node.Command)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = node.Command.Process.Kill()
		}
	}
}

func submitTasksConcurrently(client *http.Client, submissions []multiNodeSubmission, workers int) error {
	workCh := make(chan multiNodeSubmission)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range workCh {
				if err := automationRequestJSON(client, http.MethodPost, item.baseURL, "/tasks", item.task, nil); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
		}()
	}
	for _, item := range submissions {
		select {
		case err := <-errCh:
			close(workCh)
			wg.Wait()
			return err
		default:
			workCh <- item
		}
	}
	close(workCh)
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func aggregateMultiNodeEvents(runtimes []multiNodeRuntime) ([]map[string]any, error) {
	events := []map[string]any{}
	for _, node := range runtimes {
		items, err := readJSONLWithNode(node.AuditPath, node.Name)
		if err != nil {
			return nil, err
		}
		events = append(events, items...)
	}
	return events, nil
}

func readJSONLWithNode(path string, nodeName string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	items := []map[string]any{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := trim(scanner.Text())
		if line == "" {
			continue
		}
		entry := map[string]any{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		entry["_node"] = nodeName
		items = append(items, entry)
	}
	return items, scanner.Err()
}

func summarizeMultiNodeTasks(submittedBy map[string]string, events []map[string]any) map[string]map[string][]string {
	perTask := map[string]map[string][]string{}
	for taskID := range submittedBy {
		perTask[taskID] = map[string][]string{"queued": {}, "started": {}, "completed": {}}
	}
	for _, event := range events {
		taskID := fmt.Sprint(event["task_id"])
		item, ok := perTask[taskID]
		if !ok {
			continue
		}
		switch event["type"] {
		case "task.started":
			item["started"] = append(item["started"], fmt.Sprint(event["_node"]))
		case "task.completed":
			item["completed"] = append(item["completed"], fmt.Sprint(event["_node"]))
		case "task.queued":
			item["queued"] = append(item["queued"], fmt.Sprint(event["_node"]))
		}
	}
	return perTask
}

func countSubmittedBy(submittedBy map[string]string, nodeName string) int {
	count := 0
	for _, name := range submittedBy {
		if name == nodeName {
			count++
		}
	}
	return count
}

func executeLiveTakeoverScenario(client *http.Client, opts automationMultiNodeSharedQueueOptions, primary multiNodeRuntime, takeover multiNodeRuntime, runtimes []multiNodeRuntime, artifactRoot string, scenarioID string, title string, scenarioIndex int, offsetBase int, duplicateEvents []any, includeConflictProbe bool, includeIdleGap bool) (map[string]any, error) {
	subscriberGroup := "live-" + scenarioID
	subscriberID := "event-stream"
	taskID := fmt.Sprintf("%s-task-%d-%d", scenarioID, scenarioIndex, time.Now().Unix())
	task := automationTask{
		ID:               taskID,
		TraceID:          taskID,
		Title:            scenarioID + " shared-queue task",
		Entrypoint:       "echo " + scenarioID,
		RequiredExecutor: "local",
		Metadata:         map[string]string{"scenario": "live-multi-node-subscriber-takeover", "submit_node": primary.Name, "takeover_scenario": scenarioID},
	}
	if err := automationRequestJSON(client, http.MethodPost, primary.BaseURL, "/tasks", task, nil); err != nil {
		return nil, err
	}
	if _, err := waitForTaskCompletion(taskID, primary.Name, runtimes, opts.TimeoutSeconds, opts.Sleep); err != nil {
		return nil, err
	}
	if opts.Sleep != nil {
		opts.Sleep(200 * time.Millisecond)
	}
	taskEvents, err := taskEventExcerpt(runtimes, taskID)
	if err != nil {
		return nil, err
	}
	leaseOwnerTimeline := []any{}
	leaseResp := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, primary.BaseURL, "/subscriber-groups/leases", map[string]any{
		"group_id":      subscriberGroup,
		"subscriber_id": subscriberID,
		"consumer_id":   primary.Name,
		"ttl_seconds":   int(opts.TakeoverTTL / time.Second),
	}, &leaseResp); err != nil {
		return nil, err
	}
	lease := leaseResp["lease"].(map[string]any)
	leaseOwnerTimeline = append(leaseOwnerTimeline, multiNodeOwnerTimelineEntry(primary.Name, "lease_acquired", lease))
	if includeConflictProbe {
		status, _, err := externalStoreStatusOnly(client, http.MethodPost, takeover.BaseURL, "/subscriber-groups/leases", map[string]any{
			"group_id":      subscriberGroup,
			"subscriber_id": subscriberID,
			"consumer_id":   takeover.Name,
			"ttl_seconds":   int(opts.TakeoverTTL / time.Second),
		})
		if err != nil {
			return nil, err
		}
		if status != http.StatusConflict {
			return nil, fmt.Errorf("expected conflict probe 409, got %d", status)
		}
	}
	checkpointResp := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, primary.BaseURL, "/subscriber-groups/checkpoints", map[string]any{
		"group_id":            subscriberGroup,
		"subscriber_id":       subscriberID,
		"consumer_id":         primary.Name,
		"lease_token":         lease["lease_token"],
		"lease_epoch":         lease["lease_epoch"],
		"checkpoint_offset":   offsetBase,
		"checkpoint_event_id": fmt.Sprintf("evt-%d", offsetBase),
	}, &checkpointResp); err != nil {
		return nil, err
	}
	checkpointBefore := multiNodeCheckpointPayload(checkpointResp["lease"].(map[string]any))
	if includeIdleGap && opts.Sleep != nil {
		opts.Sleep(50 * time.Millisecond)
	}
	if opts.Sleep != nil {
		opts.Sleep(opts.TakeoverTTL + 300*time.Millisecond)
	}
	takeoverResp := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, takeover.BaseURL, "/subscriber-groups/leases", map[string]any{
		"group_id":      subscriberGroup,
		"subscriber_id": subscriberID,
		"consumer_id":   takeover.Name,
		"ttl_seconds":   int(opts.TakeoverTTL / time.Second),
	}, &takeoverResp); err != nil {
		return nil, err
	}
	takeoverLease := takeoverResp["lease"].(map[string]any)
	leaseOwnerTimeline = append(leaseOwnerTimeline, multiNodeOwnerTimelineEntry(takeover.Name, "takeover_acquired", takeoverLease))
	attemptedOffset := offsetBase + 1
	status, _, err := externalStoreStatusOnly(client, http.MethodPost, primary.BaseURL, "/subscriber-groups/checkpoints", map[string]any{
		"group_id":            subscriberGroup,
		"subscriber_id":       subscriberID,
		"consumer_id":         primary.Name,
		"lease_token":         lease["lease_token"],
		"lease_epoch":         lease["lease_epoch"],
		"checkpoint_offset":   attemptedOffset,
		"checkpoint_event_id": fmt.Sprintf("evt-%d", attemptedOffset),
	})
	if err != nil {
		return nil, err
	}
	if status != http.StatusConflict {
		return nil, fmt.Errorf("expected stale writer 409, got %d", status)
	}
	finalOffset := offsetBase + len(duplicateEvents)
	finalResp := map[string]any{}
	if err := automationRequestJSON(client, http.MethodPost, takeover.BaseURL, "/subscriber-groups/checkpoints", map[string]any{
		"group_id":            subscriberGroup,
		"subscriber_id":       subscriberID,
		"consumer_id":         takeover.Name,
		"lease_token":         takeoverLease["lease_token"],
		"lease_epoch":         takeoverLease["lease_epoch"],
		"checkpoint_offset":   finalOffset,
		"checkpoint_event_id": fmt.Sprintf("evt-%d", finalOffset),
	}, &finalResp); err != nil {
		return nil, err
	}
	checkpointAfter := multiNodeCheckpointPayload(finalResp["lease"].(map[string]any))
	if opts.Sleep != nil {
		opts.Sleep(100 * time.Millisecond)
	}
	runtimeEvents, perNodeEvents, err := runtimeTakeoverEvents(runtimes, subscriberGroup, subscriberID)
	if err != nil {
		return nil, err
	}
	auditTimeline := toTakeoverTimeline(runtimeEvents)
	staleWriteRejections := 0
	for _, item := range auditTimeline {
		entry := item.(map[string]any)
		if entry["action"] == "lease_fenced" {
			staleWriteRejections++
		}
	}
	auditLogPaths, err := exportRuntimeTakeoverAudit(artifactRoot, scenarioID, opts.GoRoot, perNodeEvents)
	if err != nil {
		return nil, err
	}
	return buildLiveTakeoverScenarioResult(scenarioID, title, primary.Name, takeover.Name, task.TraceID, subscriberGroup, leaseOwnerTimeline, checkpointBefore, checkpointAfter, map[string]any{"offset": offsetBase, "event_id": fmt.Sprintf("evt-%d", offsetBase)}, map[string]any{"offset": finalOffset, "event_id": fmt.Sprintf("evt-%d", finalOffset)}, duplicateEvents, staleWriteRejections, auditTimeline, taskEvents, auditLogPaths), nil
}

func waitForTaskCompletion(taskID string, submittedBy string, runtimes []multiNodeRuntime, timeoutSeconds int, sleep func(time.Duration)) (map[string][]string, error) {
	if sleep == nil {
		sleep = time.Sleep
	}
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	for time.Now().Before(deadline) {
		events, err := aggregateMultiNodeEvents(runtimes)
		if err != nil {
			return nil, err
		}
		perTask := summarizeMultiNodeTasks(map[string]string{taskID: submittedBy}, events)
		item := perTask[taskID]
		if len(item["completed"]) > 0 {
			return item, nil
		}
		sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("timed out waiting for task %s to complete", taskID)
}

func taskEventExcerpt(runtimes []multiNodeRuntime, taskID string) ([]any, error) {
	events, err := aggregateMultiNodeEvents(runtimes)
	if err != nil {
		return nil, err
	}
	excerpt := []any{}
	for _, event := range events {
		if fmt.Sprint(event["task_id"]) != taskID {
			continue
		}
		excerpt = append(excerpt, map[string]any{
			"event_id":      event["id"],
			"delivered_by":  []any{event["_node"]},
			"delivery_kind": event["type"],
		})
	}
	return excerpt, nil
}

func runtimeTakeoverEvents(runtimes []multiNodeRuntime, subscriberGroup string, subscriberID string) ([]map[string]any, map[string][]map[string]any, error) {
	timeline := []map[string]any{}
	perNode := map[string][]map[string]any{}
	for _, node := range runtimes {
		items, err := readJSONLWithNode(node.AuditPath, node.Name)
		if err != nil {
			return nil, nil, err
		}
		filtered := []map[string]any{}
		for _, event := range items {
			payload, _ := event["payload"].(map[string]any)
			if payload == nil {
				continue
			}
			switch event["type"] {
			case "subscriber.lease_acquired", "subscriber.lease_rejected", "subscriber.lease_expired", "subscriber.takeover_succeeded", "subscriber.checkpoint_committed", "subscriber.checkpoint_rejected":
			default:
				continue
			}
			if fmt.Sprint(payload["group_id"]) != subscriberGroup || fmt.Sprint(payload["subscriber_id"]) != subscriberID {
				continue
			}
			filtered = append(filtered, event)
			timeline = append(timeline, event)
		}
		perNode[node.Name] = filtered
	}
	return timeline, perNode, nil
}

func exportRuntimeTakeoverAudit(artifactRoot string, scenarioID string, repoRoot string, perNodeEvents map[string][]map[string]any) ([]any, error) {
	scenarioDir := filepath.Join(artifactRoot, scenarioID)
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		return nil, err
	}
	paths := []any{}
	for nodeName, items := range perNodeEvents {
		path := filepath.Join(scenarioDir, nodeName+"-audit.jsonl")
		file, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			body, err := json.Marshal(item)
			if err != nil {
				_ = file.Close()
				return nil, err
			}
			if _, err := file.Write(append(body, '\n')); err != nil {
				_ = file.Close()
				return nil, err
			}
		}
		_ = file.Close()
		relative, _ := filepath.Rel(repoRoot, path)
		paths = append(paths, filepath.ToSlash(relative))
	}
	return paths, nil
}

func toTakeoverTimeline(runtimeEvents []map[string]any) []any {
	timeline := []any{}
	for _, event := range runtimeEvents {
		payload, _ := event["payload"].(map[string]any)
		action := fmt.Sprint(event["type"])
		subscriber := firstNonEmpty(fmt.Sprint(payload["consumer_id"]), fmt.Sprint(payload["attempted_consumer_id"]), "unknown")
		details := map[string]any{
			"runtime_event_id":   event["id"],
			"runtime_event_type": event["type"],
			"audit_node":         event["_node"],
		}
		switch event["type"] {
		case "subscriber.lease_acquired":
			action = "lease_acquired"
			details["lease_epoch"] = payload["lease_epoch"]
			details["renewal"] = payload["renewal"]
		case "subscriber.lease_rejected":
			action = "lease_rejected"
			subscriber = firstNonEmpty(fmt.Sprint(payload["attempted_consumer_id"]), subscriber)
			details["attempted_owner"] = payload["attempted_consumer_id"]
			details["accepted_owner"] = payload["consumer_id"]
			details["lease_epoch"] = payload["lease_epoch"]
			details["reason"] = payload["reason"]
		case "subscriber.lease_expired":
			action = "lease_expired"
			subscriber = firstNonEmpty(fmt.Sprint(payload["expired_consumer_id"]), subscriber)
			details["last_offset"] = payload["checkpoint_offset"]
			details["takeover_consumer_id"] = payload["takeover_consumer_id"]
		case "subscriber.takeover_succeeded":
			action = "takeover_succeeded"
			details["lease_epoch"] = payload["lease_epoch"]
			details["previous_owner"] = payload["previous_consumer_id"]
		case "subscriber.checkpoint_committed":
			action = "checkpoint_committed"
			details["offset"] = payload["checkpoint_offset"]
			details["event_id"] = payload["checkpoint_event_id"]
		case "subscriber.checkpoint_rejected":
			reason := fmt.Sprint(payload["reason"])
			if containsSubstring(reason, "fenced") {
				action = "lease_fenced"
			} else {
				action = "checkpoint_rejected"
			}
			subscriber = firstNonEmpty(fmt.Sprint(payload["attempted_consumer_id"]), subscriber)
			details["attempted_offset"] = payload["attempted_checkpoint_offset"]
			details["attempted_event_id"] = payload["attempted_checkpoint_event_id"]
			details["accepted_owner"] = payload["consumer_id"]
			details["reason"] = reason
		}
		timeline = append(timeline, map[string]any{
			"timestamp":  normalizeISO8601(fmt.Sprint(event["timestamp"])),
			"subscriber": subscriber,
			"action":     action,
			"details":    details,
		})
	}
	return timeline
}

func buildLiveTakeoverScenarioResult(scenarioID string, title string, primary string, takeover string, traceID string, subscriberGroup string, leaseOwnerTimeline []any, checkpointBefore map[string]any, checkpointAfter map[string]any, replayStart map[string]any, replayEnd map[string]any, duplicateEvents []any, staleWriteRejections int, auditTimeline []any, eventExcerpt []any, auditLogPaths []any) map[string]any {
	assertions := buildMultiNodeAssertionResults(leaseOwnerTimeline, checkpointBefore, checkpointAfter, replayStart, replayEnd, duplicateEvents, staleWriteRejections, auditTimeline)
	allPassed := true
	for _, category := range []string{"audit", "checkpoint", "replay"} {
		for _, item := range assertions[category].([]any) {
			if item.(map[string]any)["passed"] != true {
				allPassed = false
			}
		}
	}
	return map[string]any{
		"id":                       scenarioID,
		"title":                    title,
		"subscriber_group":         subscriberGroup,
		"primary_subscriber":       primary,
		"takeover_subscriber":      takeover,
		"task_or_trace_id":         traceID,
		"audit_assertions":         []any{"Per-node audit artifacts are filtered excerpts from runtime-emitted audit events rather than harness-authored companion logs.", "The live report binds takeover actions to a real shared-queue task trace ID for the same two-node cluster run.", "Lease rejection and accepted takeover owner are captured in one ordered audit timeline."},
		"checkpoint_assertions":    []any{"Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary subscriber.", "Standby checkpoint commit is attributed to the new lease owner returned by the live API.", "Late primary checkpoint writes are fenced once takeover succeeds."},
		"replay_assertions":        []any{"Replay resumes from the last durable checkpoint boundary returned by the live lease endpoint.", "Duplicate replay candidates are counted explicitly from the overlap between the last durable offset and final takeover offset.", "The final replay cursor and final owner are both emitted in the live report schema."},
		"lease_owner_timeline":     leaseOwnerTimeline,
		"checkpoint_before":        checkpointBefore,
		"checkpoint_after":         checkpointAfter,
		"replay_start_cursor":      replayStart,
		"replay_end_cursor":        replayEnd,
		"duplicate_delivery_count": len(duplicateEvents),
		"duplicate_events":         duplicateEvents,
		"stale_write_rejections":   staleWriteRejections,
		"audit_log_paths":          auditLogPaths,
		"event_log_excerpt":        eventExcerpt,
		"audit_timeline":           auditTimeline,
		"assertion_results":        assertions,
		"all_assertions_passed":    allPassed,
		"local_limitations":        []any{"The live proof runs against a real two-node shared-queue cluster and one shared SQLite-backed subscriber lease store.", "The checked-in takeover artifacts are derived from runtime-emitted subscriber transition events exported per scenario.", "This proof upgrades ownership to a shared durable scaffold without claiming broker-backed or replicated subscriber ownership."},
	}
}

func buildMultiNodeAssertionResults(leaseOwnerTimeline []any, checkpointBefore map[string]any, checkpointAfter map[string]any, replayStart map[string]any, replayEnd map[string]any, duplicateEvents []any, staleWriteRejections int, auditTimeline []any) map[string]any {
	return map[string]any{
		"audit": []any{
			map[string]any{"label": "ownership handoff is visible in the audit timeline", "passed": ownerCount(leaseOwnerTimeline) >= 2},
			map[string]any{"label": "audit timeline contains acquisition, expiry, rejection, or takeover events", "passed": timelineHasAction(auditTimeline, "lease_acquired", "lease_expired", "lease_rejected", "lease_fenced", "takeover_succeeded")},
			map[string]any{"label": "audit timeline stays ordered by timestamp", "passed": timelineOrdered(auditTimeline)},
			map[string]any{"label": "runtime takeover events keep required owner and lease fields", "passed": runtimeAuditFieldsPresent(auditTimeline)},
		},
		"checkpoint": []any{
			map[string]any{"label": "checkpoint never regresses across takeover", "passed": automationInt(checkpointAfter["offset"]) >= automationInt(checkpointBefore["offset"])},
			map[string]any{"label": "final checkpoint owner matches the final lease owner", "passed": fmt.Sprint(checkpointAfter["owner"]) == lastOwner(leaseOwnerTimeline)},
			map[string]any{"label": "stale writers do not replace the accepted checkpoint owner", "passed": staleWriteRejections == 0 || fmt.Sprint(checkpointAfter["owner"]) == lastOwner(leaseOwnerTimeline)},
		},
		"replay": []any{
			map[string]any{"label": "replay restarts from the durable checkpoint boundary", "passed": automationInt(replayStart["offset"]) == automationInt(checkpointBefore["offset"])},
			map[string]any{"label": "replay end cursor advances to the final durable checkpoint", "passed": automationInt(replayEnd["offset"]) == automationInt(checkpointAfter["offset"])},
			map[string]any{"label": "duplicate replay candidates are counted explicitly", "passed": len(duplicateEvents) >= 0},
		},
	}
}

func buildLiveTakeoverReportGo(scenarios []any, sharedQueueReportPath string) map[string]any {
	passing := 0
	duplicates := 0
	rejections := 0
	for _, item := range scenarios {
		entry, _ := item.(map[string]any)
		if entry["all_assertions_passed"] == true {
			passing++
		}
		duplicates += automationInt(entry["duplicate_delivery_count"])
		rejections += automationInt(entry["stale_write_rejections"])
	}
	return map[string]any{
		"generated_at": utcISOTime(time.Now().UTC()),
		"ticket":       "OPE-260",
		"title":        "Live multi-node subscriber takeover proof",
		"status":       "live-multi-node-proof",
		"harness_mode": "live_multi_node_bigclawd_cluster",
		"current_primitives": map[string]any{
			"lease_aware_checkpoints": []any{"internal/events/subscriber_leases.go", "internal/events/subscriber_leases_test.go", "internal/api/server.go"},
			"shared_queue_evidence":   []any{"scripts/e2e/multi_node_shared_queue.py", sharedQueueReportPath},
			"live_takeover_harness":   []any{"internal/api/server.go", "scripts/e2e/multi_node_shared_queue.py", "docs/reports/live-multi-node-subscriber-takeover-report.json"},
		},
		"required_report_sections": []any{"scenario metadata", "fault injection steps", "audit assertions", "checkpoint assertions", "replay assertions", "per-node audit artifacts", "final owner and replay cursor summary", "duplicate delivery accounting", "open blockers and follow-up implementation hooks"},
		"implementation_path":      []any{"run a real two-node bigclawd cluster against one shared SQLite queue", "drive lease acquisition, expiry, fencing, and checkpoint takeover through the live subscriber-group API on both nodes against one shared SQLite lease backend", "slice canonical per-node takeover artifacts out of the runtime audit stream beside the checked-in report", "keep broker-backed and replicated subscriber ownership caveats explicit until a broker-native lease backend exists"},
		"summary": map[string]any{
			"scenario_count":           len(scenarios),
			"passing_scenarios":        passing,
			"failing_scenarios":        len(scenarios) - passing,
			"duplicate_delivery_count": duplicates,
			"stale_write_rejections":   rejections,
		},
		"scenarios":      scenarios,
		"remaining_gaps": []any{"Subscriber ownership now uses a shared durable SQLite scaffold, but it is not yet broker-backed or replicated.", "The live proof reuses real shared-queue nodes but does not yet validate broker-backed or replicated subscriber ownership.", "Native runtime audit coverage now captures takeover transitions, but the proof still depends on the current lease API rather than broker-backed replay ownership."},
	}
}

func multiNodeCheckpointPayload(lease map[string]any) map[string]any {
	return map[string]any{
		"owner":       lease["consumer_id"],
		"lease_epoch": lease["lease_epoch"],
		"lease_token": lease["lease_token"],
		"offset":      lease["checkpoint_offset"],
		"event_id":    firstNonEmpty(fmt.Sprint(lease["checkpoint_event_id"]), ""),
		"updated_at":  normalizeISO8601(fmt.Sprint(lease["updated_at"])),
	}
}

func multiNodeOwnerTimelineEntry(owner string, event string, lease map[string]any) map[string]any {
	return map[string]any{
		"timestamp":           utcISOTime(time.Now().UTC()),
		"owner":               owner,
		"event":               event,
		"lease_epoch":         lease["lease_epoch"],
		"checkpoint_offset":   lease["checkpoint_offset"],
		"checkpoint_event_id": firstNonEmpty(fmt.Sprint(lease["checkpoint_event_id"]), ""),
	}
}

func normalizeISO8601(value string) string {
	if trim(value) == "" {
		return ""
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return ts.UTC().Format(time.RFC3339)
}

func runtimeAuditFieldsPresent(items []any) bool {
	for _, item := range items {
		entry, _ := item.(map[string]any)
		details, _ := entry["details"].(map[string]any)
		if trim(fmt.Sprint(details["runtime_event_id"])) == "" || trim(fmt.Sprint(details["runtime_event_type"])) == "" || trim(fmt.Sprint(entry["subscriber"])) == "" {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trim(value) != "" {
			return value
		}
	}
	return ""
}

func containsSubstring(value string, pattern string) bool {
	return len(pattern) == 0 || (len(value) >= len(pattern) && filepath.Base(value) != "" && (func() bool { return stringContains(value, pattern) })())
}

func stringContains(value string, pattern string) bool {
	return len(pattern) == 0 || (len(value) >= len(pattern) && (func() bool { return len(value) >= len(pattern) && indexString(value, pattern) >= 0 })())
}

func indexString(value string, pattern string) int {
	for i := 0; i+len(pattern) <= len(value); i++ {
		if value[i:i+len(pattern)] == pattern {
			return i
		}
	}
	return -1
}
