package reporting

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ShadowCompareGenerator = "bigclaw-go/scripts/migration/shadow_compare/main.go"
	ShadowMatrixGenerator  = "bigclaw-go/scripts/migration/shadow_matrix/main.go"
)

type ShadowCompareOptions struct {
	PrimaryBaseURL string
	ShadowBaseURL  string
	TaskPath       string
	Timeout        time.Duration
	HealthTimeout  time.Duration
	HTTPClient     *http.Client
}

type ShadowMatrixOptions struct {
	PrimaryBaseURL     string
	ShadowBaseURL      string
	TaskFiles          []string
	CorpusManifestPath string
	ReplayCorpusSlices bool
	Timeout            time.Duration
	HealthTimeout      time.Duration
	HTTPClient         *http.Client
}

func RunShadowCompare(options ShadowCompareOptions) (map[string]any, error) {
	taskPath := strings.TrimSpace(options.TaskPath)
	if taskPath == "" {
		return nil, errors.New("task path is required")
	}
	task, err := loadShadowJSON(taskPath)
	if err != nil {
		return nil, err
	}
	return CompareShadowTask(options.PrimaryBaseURL, options.ShadowBaseURL, task, options.Timeout, options.HealthTimeout, options.HTTPClient)
}

func CompareShadowTask(primaryBaseURL string, shadowBaseURL string, task map[string]any, timeout time.Duration, healthTimeout time.Duration, client *http.Client) (map[string]any, error) {
	primaryBaseURL = strings.TrimRight(strings.TrimSpace(primaryBaseURL), "/")
	shadowBaseURL = strings.TrimRight(strings.TrimSpace(shadowBaseURL), "/")
	if primaryBaseURL == "" || shadowBaseURL == "" {
		return nil, errors.New("primary and shadow base URLs are required")
	}
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	if healthTimeout <= 0 {
		healthTimeout = 60 * time.Second
	}
	httpClient := client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	if err := waitShadowHealth(httpClient, primaryBaseURL, healthTimeout); err != nil {
		return nil, err
	}
	if err := waitShadowHealth(httpClient, shadowBaseURL, healthTimeout); err != nil {
		return nil, err
	}

	primaryTask := cloneShadowMap(task)
	shadowTask := cloneShadowMap(task)
	baseID := firstNonEmptyString(asString(task["id"]), fmt.Sprintf("shadow-%d", time.Now().Unix()))
	traceID := firstNonEmptyString(asString(task["trace_id"]), baseID)
	primaryTask["id"] = baseID + "-primary"
	shadowTask["id"] = baseID + "-shadow"
	primaryTask["trace_id"] = traceID
	shadowTask["trace_id"] = traceID

	if _, err := requestShadowJSON(httpClient, primaryBaseURL, http.MethodPost, "/tasks", primaryTask); err != nil {
		return nil, err
	}
	if _, err := requestShadowJSON(httpClient, shadowBaseURL, http.MethodPost, "/tasks", shadowTask); err != nil {
		return nil, err
	}

	primaryStatus, err := waitShadowTerminal(httpClient, primaryBaseURL, asString(primaryTask["id"]), timeout)
	if err != nil {
		return nil, err
	}
	shadowStatus, err := waitShadowTerminal(httpClient, shadowBaseURL, asString(shadowTask["id"]), timeout)
	if err != nil {
		return nil, err
	}
	primaryEventsResponse, err := requestShadowJSON(httpClient, primaryBaseURL, http.MethodGet, "/events?task_id="+url.QueryEscape(asString(primaryTask["id"]))+"&limit=100", nil)
	if err != nil {
		return nil, err
	}
	shadowEventsResponse, err := requestShadowJSON(httpClient, shadowBaseURL, http.MethodGet, "/events?task_id="+url.QueryEscape(asString(shadowTask["id"]))+"&limit=100", nil)
	if err != nil {
		return nil, err
	}
	primaryEvents := anyToMapSlice(primaryEventsResponse["events"])
	shadowEvents := anyToMapSlice(shadowEventsResponse["events"])
	return map[string]any{
		"trace_id": traceID,
		"primary": map[string]any{
			"task_id": asString(primaryTask["id"]),
			"status":  primaryStatus,
			"events":  primaryEvents,
		},
		"shadow": map[string]any{
			"task_id": asString(shadowTask["id"]),
			"status":  shadowStatus,
			"events":  shadowEvents,
		},
		"diff": map[string]any{
			"state_equal":              asString(primaryStatus["state"]) == asString(shadowStatus["state"]),
			"event_count_delta":        len(primaryEvents) - len(shadowEvents),
			"event_types_equal":        equalStringSlices(shadowEventTypes(primaryEvents), shadowEventTypes(shadowEvents)),
			"primary_event_types":      shadowEventTypes(primaryEvents),
			"shadow_event_types":       shadowEventTypes(shadowEvents),
			"primary_timeline_seconds": shadowTimelineSeconds(primaryEvents),
			"shadow_timeline_seconds":  shadowTimelineSeconds(shadowEvents),
		},
	}, nil
}

func RunShadowMatrix(options ShadowMatrixOptions) (map[string]any, error) {
	if len(options.TaskFiles) == 0 && strings.TrimSpace(options.CorpusManifestPath) == "" {
		return nil, errors.New("at least one task file or corpus manifest is required")
	}
	fixtureEntries, err := loadShadowFixtureEntries(options.TaskFiles)
	if err != nil {
		return nil, err
	}
	var manifestMeta map[string]any
	var replayEntries []map[string]any
	var corpusSlices []map[string]any
	if strings.TrimSpace(options.CorpusManifestPath) != "" {
		manifestMeta, replayEntries, corpusSlices, err = loadShadowCorpusManifestEntries(options.CorpusManifestPath, options.ReplayCorpusSlices)
		if err != nil {
			return nil, err
		}
	}
	executionEntries := append(append([]map[string]any{}, fixtureEntries...), replayEntries...)
	results := make([]map[string]any, 0, len(executionEntries))
	for idx, entry := range executionEntries {
		task := cloneShadowMap(asMap(entry["task"]))
		baseID := firstNonEmptyString(asString(task["id"]), fmt.Sprintf("matrix-task-%d", idx+1))
		task["id"] = fmt.Sprintf("%s-m%d", baseID, idx+1)
		result, err := CompareShadowTask(options.PrimaryBaseURL, options.ShadowBaseURL, task, options.Timeout, options.HealthTimeout, options.HTTPClient)
		if err != nil {
			return nil, err
		}
		result["source_file"] = entry["source_file"]
		result["source_kind"] = entry["source_kind"]
		result["task_shape"] = entry["task_shape"]
		if corpusSlice := asMap(entry["corpus_slice"]); len(corpusSlice) > 0 {
			result["corpus_slice"] = corpusSlice
		}
		results = append(results, result)
	}
	return buildShadowMatrixReport(results, fixtureEntries, corpusSlices, manifestMeta), nil
}

func loadShadowFixtureEntries(taskFiles []string) ([]map[string]any, error) {
	entries := make([]map[string]any, 0, len(taskFiles))
	for _, taskFile := range taskFiles {
		task, err := loadShadowJSON(taskFile)
		if err != nil {
			return nil, err
		}
		entries = append(entries, makeShadowExecutionEntry(task, "fixture", taskFile, deriveShadowTaskShape(task), nil))
	}
	return entries, nil
}

func loadShadowCorpusManifestEntries(manifestPath string, replayCorpusSlices bool) (map[string]any, []map[string]any, []map[string]any, error) {
	manifest, err := loadShadowJSON(manifestPath)
	if err != nil {
		return nil, nil, nil, err
	}
	slices := asSlice(manifest["slices"])
	if slices == nil {
		return nil, nil, nil, errors.New("corpus manifest must contain a top-level slices array")
	}
	coverageSlices := make([]map[string]any, 0, len(slices))
	replayEntries := make([]map[string]any, 0)
	for idx, rawSlice := range slices {
		sliceData, err := normalizeShadowCorpusSlice(asMap(rawSlice), idx+1, manifestPath)
		if err != nil {
			return nil, nil, nil, err
		}
		coverageSlices = append(coverageSlices, sliceData)
		if replayCorpusSlices && len(asMap(sliceData["task"])) > 0 {
			replayEntries = append(replayEntries, makeShadowExecutionEntry(
				asMap(sliceData["task"]),
				"corpus",
				asString(sliceData["source_file"]),
				asString(sliceData["task_shape"]),
				sliceData,
			))
		}
	}
	manifestMeta := map[string]any{
		"name":         firstNonEmptyString(asString(manifest["name"]), strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))),
		"generated_at": manifest["generated_at"],
		"source_file":  manifestPath,
	}
	return manifestMeta, replayEntries, coverageSlices, nil
}

func buildShadowCorpusCoverage(fixtureEntries []map[string]any, corpusSlices []map[string]any, manifestMeta map[string]any) map[string]any {
	fixtureByShape := map[string][]map[string]any{}
	for _, entry := range fixtureEntries {
		shape := asString(entry["task_shape"])
		fixtureByShape[shape] = append(fixtureByShape[shape], entry)
	}

	type corpusAggregate struct {
		SliceCount      int
		ReplayableCount int
		CorpusWeight    int
		SliceIDs        []string
		Titles          []string
	}
	corpusByShape := map[string]*corpusAggregate{}
	for _, sliceData := range corpusSlices {
		shape := asString(sliceData["task_shape"])
		aggregate := corpusByShape[shape]
		if aggregate == nil {
			aggregate = &corpusAggregate{}
			corpusByShape[shape] = aggregate
		}
		aggregate.SliceCount++
		if len(asMap(sliceData["task"])) > 0 {
			aggregate.ReplayableCount++
		}
		aggregate.CorpusWeight += asInt(sliceData["weight"])
		aggregate.SliceIDs = append(aggregate.SliceIDs, asString(sliceData["slice_id"]))
		aggregate.Titles = append(aggregate.Titles, asString(sliceData["title"]))
	}

	shapes := make([]string, 0, len(corpusByShape))
	for shape := range corpusByShape {
		shapes = append(shapes, shape)
	}
	sort.Slice(shapes, func(i, j int) bool {
		left := corpusByShape[shapes[i]]
		right := corpusByShape[shapes[j]]
		if left.CorpusWeight == right.CorpusWeight {
			return shapes[i] < shapes[j]
		}
		return left.CorpusWeight > right.CorpusWeight
	})

	shapeScorecard := make([]map[string]any, 0, len(shapes))
	for _, shape := range shapes {
		aggregate := corpusByShape[shape]
		fixtures := fixtureByShape[shape]
		fixtureSources := make([]string, 0, len(fixtures))
		for _, entry := range fixtures {
			fixtureSources = append(fixtureSources, asString(entry["source_file"]))
		}
		shapeScorecard = append(shapeScorecard, map[string]any{
			"task_shape":             shape,
			"fixture_task_count":     len(fixtures),
			"fixture_sources":        fixtureSources,
			"corpus_slice_count":     aggregate.SliceCount,
			"replayable_slice_count": aggregate.ReplayableCount,
			"corpus_weight":          aggregate.CorpusWeight,
			"corpus_slice_ids":       aggregate.SliceIDs,
			"corpus_titles":          aggregate.Titles,
			"covered_by_fixture":     len(fixtures) > 0,
		})
	}

	uncoveredSlices := make([]map[string]any, 0)
	for _, sliceData := range corpusSlices {
		if len(fixtureByShape[asString(sliceData["task_shape"])]) > 0 {
			continue
		}
		uncoveredSlices = append(uncoveredSlices, map[string]any{
			"slice_id":    sliceData["slice_id"],
			"title":       sliceData["title"],
			"task_shape":  sliceData["task_shape"],
			"weight":      sliceData["weight"],
			"replayable":  len(asMap(sliceData["task"])) > 0,
			"source_file": sliceData["source_file"],
			"tags":        sliceData["tags"],
			"notes":       firstNonEmptyString(asString(sliceData["notes"])),
		})
	}

	replayableSliceCount := 0
	for _, sliceData := range corpusSlices {
		if len(asMap(sliceData["task"])) > 0 {
			replayableSliceCount++
		}
	}
	return map[string]any{
		"manifest_name":                 asString(manifestMeta["name"]),
		"manifest_source_file":          asString(manifestMeta["source_file"]),
		"generated_at":                  manifestMeta["generated_at"],
		"fixture_task_count":            len(fixtureEntries),
		"corpus_slice_count":            len(corpusSlices),
		"corpus_replayable_slice_count": replayableSliceCount,
		"covered_corpus_slice_count":    len(corpusSlices) - len(uncoveredSlices),
		"uncovered_corpus_slice_count":  len(uncoveredSlices),
		"shape_scorecard":               shapeScorecard,
		"uncovered_slices":              uncoveredSlices,
	}
}

func buildShadowMatrixReport(results []map[string]any, fixtureEntries []map[string]any, corpusSlices []map[string]any, manifestMeta map[string]any) map[string]any {
	matched := 0
	for _, item := range results {
		diff := asMap(item["diff"])
		if asBool(diff["state_equal"]) && asBool(diff["event_types_equal"]) {
			matched++
		}
	}
	report := map[string]any{
		"total":      len(results),
		"matched":    matched,
		"mismatched": len(results) - matched,
		"inputs": map[string]any{
			"fixture_task_count": len(fixtureEntries),
			"corpus_slice_count": len(corpusSlices),
			"manifest_name":      mapOrNilString(manifestMeta, "name"),
		},
		"results": results,
	}
	if len(corpusSlices) > 0 && len(manifestMeta) > 0 {
		report["corpus_coverage"] = buildShadowCorpusCoverage(fixtureEntries, corpusSlices, manifestMeta)
	}
	return report
}

func matchedAllShadowResults(results []map[string]any) bool {
	for _, item := range results {
		diff := asMap(item["diff"])
		if !asBool(diff["state_equal"]) || !asBool(diff["event_types_equal"]) {
			return false
		}
	}
	return true
}

func deriveShadowTaskShape(task map[string]any) string {
	features := []string{}
	executor := firstNonEmptyString(asString(task["required_executor"]), asString(task["executor"]), "default")
	features = append(features, "executor:"+executor)

	labels := stringSliceFromAny(task["labels"])
	sort.Strings(labels)
	if len(labels) > 0 {
		features = append(features, "labels:"+strings.Join(labels, ","))
	}
	if task["budget_cents"] != nil {
		features = append(features, "budgeted")
	}
	if len(asSlice(task["acceptance_criteria"])) > 0 {
		features = append(features, "acceptance")
	}
	if len(asSlice(task["validation_plan"])) > 0 {
		features = append(features, "validation-plan")
	}
	if scenario := asString(asMap(task["metadata"])["scenario"]); scenario != "" {
		features = append(features, "scenario:"+scenario)
	}
	return strings.Join(features, "|")
}

func makeShadowExecutionEntry(task map[string]any, sourceKind string, sourceFile string, taskShape string, sliceData map[string]any) map[string]any {
	entry := map[string]any{
		"task":        cloneShadowMap(task),
		"source_kind": sourceKind,
		"source_file": sourceFile,
		"task_shape":  firstNonEmptyString(taskShape, deriveShadowTaskShape(task)),
	}
	taskCopy := asMap(entry["task"])
	taskCopy["_source_file"] = sourceFile
	entry["task"] = taskCopy
	if len(sliceData) > 0 {
		entry["corpus_slice"] = map[string]any{
			"id":     sliceData["slice_id"],
			"title":  sliceData["title"],
			"weight": sliceData["weight"],
			"tags":   sliceData["tags"],
		}
	}
	return entry
}

func normalizeShadowCorpusSlice(sliceData map[string]any, index int, manifestPath string) (map[string]any, error) {
	sliceID := firstNonEmptyString(asString(sliceData["slice_id"]), fmt.Sprintf("corpus-slice-%d", index))
	title := firstNonEmptyString(asString(sliceData["title"]), sliceID)
	weight := asIntWithFallback(sliceData["weight"], 1)
	tags := stringSliceFromAny(sliceData["tags"])

	var task map[string]any
	var sourceFile string
	if taskFile := strings.TrimSpace(asString(sliceData["task_file"])); taskFile != "" {
		sourceFile = taskFile
		var err error
		task, err = loadShadowJSON(resolveShadowManifestTaskFile(manifestPath, taskFile))
		if err != nil {
			return nil, err
		}
	} else if rawTask := asMap(sliceData["task"]); len(rawTask) > 0 {
		task = cloneShadowMap(rawTask)
		sourceFile = fmt.Sprintf("%s#%s", filepath.Base(manifestPath), sliceID)
	}

	taskShape := firstNonEmptyString(asString(sliceData["task_shape"]))
	if taskShape == "" && len(task) > 0 {
		taskShape = deriveShadowTaskShape(task)
	}
	if taskShape == "" {
		return nil, fmt.Errorf("corpus slice %s must define task_shape or provide task/task_file", sliceID)
	}
	return map[string]any{
		"slice_id":    sliceID,
		"title":       title,
		"weight":      weight,
		"tags":        tags,
		"task_shape":  taskShape,
		"task":        task,
		"source_file": sourceFile,
		"replay":      asBool(sliceData["replay"]),
		"notes":       firstNonEmptyString(asString(sliceData["notes"])),
	}, nil
}

func resolveShadowManifestTaskFile(manifestPath string, taskFile string) string {
	if filepath.IsAbs(taskFile) {
		return taskFile
	}
	return filepath.Join(filepath.Dir(manifestPath), taskFile)
}

func waitShadowHealth(client *http.Client, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		payload, err := requestShadowJSON(client, baseURL, http.MethodGet, "/healthz", nil)
		if err == nil && asBool(payload["ok"]) {
			return nil
		}
		lastErr = err
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for health on %s: %v", baseURL, lastErr)
}

func waitShadowTerminal(client *http.Client, baseURL string, taskID string, timeout time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := requestShadowJSON(client, baseURL, http.MethodGet, "/tasks/"+taskID, nil)
		if err != nil {
			return nil, err
		}
		switch asString(status["state"]) {
		case "succeeded", "dead_letter", "cancelled", "failed":
			return status, nil
		}
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("timeout waiting for %s on %s", taskID, baseURL)
}

func requestShadowJSON(client *http.Client, baseURL string, method string, path string, payload any) (map[string]any, error) {
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
	req, err := http.NewRequest(method, baseURL+path, body)
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
		return nil, fmt.Errorf("request %s %s failed with status %d", method, baseURL+path, resp.StatusCode)
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func shadowEventTypes(events []map[string]any) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, asString(event["type"]))
	}
	return out
}

func shadowTimelineSeconds(events []map[string]any) float64 {
	if len(events) < 2 {
		return 0
	}
	start := asString(events[0]["timestamp"])
	end := asString(events[len(events)-1]["timestamp"])
	if start == "" || end == "" {
		return 0
	}
	startTS, err := parseFlexibleTime(start)
	if err != nil {
		return 0
	}
	endTS, err := parseFlexibleTime(end)
	if err != nil {
		return 0
	}
	seconds := endTS.Sub(startTS).Seconds()
	if seconds < 0 {
		return 0
	}
	return seconds
}

func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func loadShadowJSON(path string) (map[string]any, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(contents, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func cloneShadowMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	contents, _ := json.Marshal(input)
	var out map[string]any
	_ = json.Unmarshal(contents, &out)
	return out
}

func mapOrNilString(input map[string]any, key string) any {
	if len(input) == 0 {
		return nil
	}
	return input[key]
}
