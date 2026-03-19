package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/bootstrap"
	"bigclaw-go/internal/githubsync"
	"bigclaw-go/internal/refill"
)

const (
	pollQuery = `
query RefillIssues($projectSlug: String!, $stateNames: [String!]!, $first: Int!, $after: String) {
  issues(filter: {project: {slugId: {eq: $projectSlug}}, state: {name: {in: $stateNames}}}, first: $first, after: $after) {
    nodes {
      id
      identifier
      title
      state {
        name
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}`
	promoteMutation = `
mutation PromoteIssue($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
    issue {
      identifier
      state {
        name
      }
    }
  }
}`
)

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type linearIssueNode struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      struct {
		Name string `json:"name"`
	} `json:"state"`
}

type refillResponse struct {
	Data struct {
		Issues struct {
			Nodes    []linearIssueNode `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"issues"`
	} `json:"data"`
	Errors any `json:"errors"`
}

type promoteResponse struct {
	Data struct {
		IssueUpdate struct {
			Success bool `json:"success"`
			Issue   struct {
				Identifier string `json:"identifier"`
				State      struct {
					Name string `json:"name"`
				} `json:"state"`
			} `json:"issue"`
		} `json:"issueUpdate"`
	} `json:"data"`
	Errors any `json:"errors"`
}

type refillClient interface {
	backend() string
	fetchIssueStates(projectSlug string, stateNames []string) ([]refill.LinearIssue, error)
	promoteIssue(issueID string, stateID string, stateName string) (bool, string, error)
}

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: bigclawctl <github-sync|workspace|refill> ...")
	}
	var err error
	switch os.Args[1] {
	case "github-sync":
		err = runGitHubSync(os.Args[2:])
	case "workspace":
		err = runWorkspace(os.Args[2:])
	case "refill":
		err = runRefill(os.Args[2:])
	case "local-issue":
		err = runLocalIssue(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}
	if err != nil {
		fatalf("%v", err)
	}
}

func runGitHubSync(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: bigclawctl github-sync <install|status|sync> [flags]")
	}
	command := args[0]
	flags := flag.NewFlagSet("github-sync "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repo := flags.String("repo", ".", "repository path")
	remote := flags.String("remote", "origin", "git remote")
	hooksPath := flags.String("hooks-path", ".githooks", "hooks path")
	allowDirty := flags.Bool("allow-dirty", false, "allow dirty")
	requireClean := flags.Bool("require-clean", false, "require clean")
	requireSynced := flags.Bool("require-synced", false, "require synced")
	asJSON := flags.Bool("json", false, "json")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	switch command {
	case "install":
		hooksDir, err := githubsync.InstallGitHooks(*repo, *hooksPath)
		if err != nil {
			return emit(map[string]any{"status": "error", "error": err.Error(), "repo": absPath(*repo)}, *asJSON, 1)
		}
		return emit(map[string]any{"status": "installed", "repo": absPath(*repo), "hooks_path": hooksDir}, *asJSON, 0)
	case "status":
		status, err := githubsync.InspectRepoSync(*repo, *remote)
		if err != nil {
			return emit(map[string]any{"status": "error", "error": err.Error(), "repo": absPath(*repo)}, *asJSON, 1)
		}
		payload := statusToMap("ok", status)
		code := 0
		if *requireClean && status.Dirty {
			code = 1
		}
		if *requireSynced && !status.Synced {
			code = 1
		}
		return emit(payload, *asJSON, code)
	case "sync":
		status, err := githubsync.EnsureRepoSync(*repo, *remote, true, *allowDirty)
		if err != nil {
			return emit(map[string]any{"status": "error", "error": err.Error(), "repo": absPath(*repo)}, *asJSON, 1)
		}
		payload := statusToMap("ok", status)
		code := 0
		if *requireClean && status.Dirty {
			code = 1
		}
		if *requireSynced && !status.Synced {
			code = 1
		}
		return emit(payload, *asJSON, code)
	default:
		return fmt.Errorf("unknown github-sync subcommand: %s", command)
	}
}

func runLocalIssue(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: bigclawctl local-issue <update|closeout> [flags]")
	}
	command := args[0]
	flags := flag.NewFlagSet("local-issue "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	localIssuesPath := flags.String("local-issues", "", "local issue store path")
	issueRef := flags.String("issue", "", "issue identifier or id")
	stateName := flags.String("state", "Done", "state name")
	commentText := flags.String("comment", "", "progress comment")
	summary := flags.String("summary", "", "what changed")
	validation := flags.String("validation", "", "validation commands/results")
	commitSHA := flags.String("commit", "", "commit SHA")
	prURL := flags.String("pr-url", "", "PR URL")
	asJSON := flags.Bool("json", false, "json")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	switch command {
	case "update":
		if trim(*issueRef) == "" {
			return errors.New("local-issue update requires --issue")
		}
		storePath := resolvedLocalIssueStorePath(*localIssuesPath)
		if storePath == "" {
			return errors.New("local issue store not found")
		}
		store, err := refill.LoadLocalIssueStore(storePath)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		comment := refill.LocalIssueComment{
			Body:      trim(*commentText),
			CreatedAt: now,
			Metadata: map[string]any{
				"commit_sha": trim(*commitSHA),
				"pr_url":     trim(*prURL),
			},
		}
		nextState := trim(*stateName)
		if !hasCLIFlag(args[1:], "--state") {
			nextState = ""
		}
		updatedState, err := store.UpdateIssue(*issueRef, nextState, comment, now)
		if err != nil {
			return err
		}
		return emit(map[string]any{
			"status":       "ok",
			"backend":      "local",
			"issue":        trim(*issueRef),
			"state":        updatedState,
			"local_issues": absPath(storePath),
			"commented":    trim(*commentText) != "",
			"commit_sha":   trim(*commitSHA),
			"pr_url":       trim(*prURL),
		}, *asJSON, 0)
	case "closeout":
		if trim(*issueRef) == "" {
			return errors.New("local-issue closeout requires --issue")
		}
		storePath := resolvedLocalIssueStorePath(*localIssuesPath)
		if storePath == "" {
			return errors.New("local issue store not found")
		}
		store, err := refill.LoadLocalIssueStore(storePath)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		commentBody := buildCloseoutComment(*summary, *validation, *commitSHA, *prURL)
		comment := refill.LocalIssueComment{
			Body:      commentBody,
			CreatedAt: now,
			Metadata: map[string]any{
				"commit_sha": trim(*commitSHA),
				"pr_url":     trim(*prURL),
			},
		}
		updatedState, err := store.CloseIssue(*issueRef, *stateName, comment, now)
		if err != nil {
			return err
		}
		return emit(map[string]any{
			"status":       "ok",
			"backend":      "local",
			"issue":        trim(*issueRef),
			"state":        updatedState,
			"local_issues": absPath(storePath),
			"commit_sha":   trim(*commitSHA),
			"pr_url":       trim(*prURL),
		}, *asJSON, 0)
	default:
		return fmt.Errorf("unknown local-issue subcommand: %s", command)
	}
}

func runWorkspace(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]")
	}
	command := args[0]
	flags := flag.NewFlagSet("workspace "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	workspace := flags.String("workspace", ".", "workspace")
	workspaceRoot := flags.String("workspace-root", ".", "workspace root")
	issue := flags.String("issue", "", "issue identifier")
	repoURL := flags.String("repo-url", "", "repo url")
	defaultBranch := flags.String("default-branch", "main", "default branch")
	cacheRoot := flags.String("cache-root", "", "cache root")
	cacheBase := flags.String("cache-base", "~/.cache/symphony/repos", "cache base")
	cacheKey := flags.String("cache-key", "", "cache key")
	asJSON := flags.Bool("json", false, "json")
	issuesCSV := flags.String("issues", "", "comma-separated issues")
	reportPath := flags.String("report", "", "report path")
	cleanup := flags.Bool("cleanup", true, "cleanup")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	switch command {
	case "bootstrap":
		status, err := bootstrap.BootstrapWorkspace(*workspace, *issue, *repoURL, *defaultBranch, *cacheRoot, *cacheBase, *cacheKey)
		if err != nil {
			return emit(map[string]any{"status": "error", "workspace": absPath(*workspace), "error": err.Error()}, *asJSON, 1)
		}
		return emit(mergeMap(map[string]any{"status": "ok"}, structToMap(status)), *asJSON, 0)
	case "cleanup":
		status, err := bootstrap.CleanupWorkspace(*workspace, *issue, *repoURL, *defaultBranch, *cacheRoot, *cacheBase, *cacheKey)
		if err != nil {
			return emit(map[string]any{"status": "error", "workspace": absPath(*workspace), "error": err.Error()}, *asJSON, 1)
		}
		return emit(mergeMap(map[string]any{"status": "ok"}, structToMap(status)), *asJSON, 0)
	case "validate":
		issues := splitCSV(*issuesCSV)
		report, err := bootstrap.BuildValidationReport(*repoURL, *workspaceRoot, issues, *defaultBranch, *cacheRoot, *cacheBase, *cacheKey, *cleanup)
		if err != nil {
			return err
		}
		if *reportPath != "" {
			if _, err := bootstrap.WriteValidationReport(report, *reportPath); err != nil {
				return err
			}
		}
		if *asJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(report)
		}
		_, err = os.Stdout.WriteString(bootstrap.RenderValidationMarkdown(report))
		return err
	default:
		return fmt.Errorf("unknown workspace subcommand: %s", command)
	}
}

func runRefill(args []string) error {
	flags := flag.NewFlagSet("refill", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	queuePath := flags.String("queue", "docs/parallel-refill-queue.json", "queue path")
	localIssuesPath := flags.String("local-issues", "", "local issue store path")
	targetInProgress := flags.Int("target-in-progress", -1, "override target")
	watch := flags.Bool("watch", false, "watch")
	interval := flags.Int("interval", 20, "interval")
	apply := flags.Bool("apply", false, "apply")
	refreshURL := flags.String("refresh-url", "", "refresh url")
	if err := flags.Parse(args); err != nil {
		return err
	}
	queue, err := refill.LoadQueue(resolveRepoRelativePath(*queuePath))
	if err != nil {
		return err
	}
	client, err := refillClientFromFlags(*localIssuesPath)
	if err != nil {
		return err
	}
	runOnce := func() error {
		var override *int
		if *targetInProgress >= 0 {
			override = targetInProgress
		}
		return runRefillOnce(queue, client, *apply, *refreshURL, override)
	}
	if !*watch {
		return runOnce()
	}
	for {
		if err := runOnce(); err != nil {
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

type linearClient struct {
	apiKey     string
	endpoint   string
	httpClient *http.Client
}

func (c *linearClient) backend() string {
	return "linear"
}

func (c *linearClient) graphqlEndpoint() string {
	if trim(c.endpoint) != "" {
		return c.endpoint
	}
	return "https://api.linear.app/graphql"
}

func (c *linearClient) client() *http.Client {
	if c.httpClient != nil {
		return c.httpClient
	}
	return http.DefaultClient
}

func (c *linearClient) graphql(query string, variables map[string]any, target any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, c.graphqlEndpoint(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", c.apiKey)
	response, err := c.client().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode >= 400 {
		return fmt.Errorf("Linear HTTP %d: %s", response.StatusCode, string(responseBody))
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return err
	}
	return nil
}

func (c *linearClient) fetchIssueStates(projectSlug string, stateNames []string) ([]refill.LinearIssue, error) {
	issues := []refill.LinearIssue{}
	cursor := ""
	for {
		response := refillResponse{}
		variables := map[string]any{
			"projectSlug": projectSlug,
			"stateNames":  stateNames,
			"first":       50,
			"after":       nil,
		}
		if cursor != "" {
			variables["after"] = cursor
		}
		if err := c.graphql(pollQuery, variables, &response); err != nil {
			return nil, err
		}
		if response.Errors != nil {
			return nil, fmt.Errorf("%v", response.Errors)
		}
		for _, node := range response.Data.Issues.Nodes {
			issues = append(issues, refill.LinearIssue{ID: node.ID, Identifier: node.Identifier, StateName: node.State.Name})
		}
		if !response.Data.Issues.PageInfo.HasNextPage {
			return issues, nil
		}
		cursor = response.Data.Issues.PageInfo.EndCursor
	}
}

func (c *linearClient) promoteIssue(identifier string, stateID string, _ string) (bool, string, error) {
	response := promoteResponse{}
	if err := c.graphql(promoteMutation, map[string]any{"id": identifier, "input": map[string]any{"stateId": stateID}}, &response); err != nil {
		return false, "", err
	}
	if response.Errors != nil {
		return false, "", fmt.Errorf("%v", response.Errors)
	}
	return response.Data.IssueUpdate.Success, response.Data.IssueUpdate.Issue.State.Name, nil
}

type localIssueClient struct {
	store *refill.LocalIssueStore
	now   func() time.Time
}

func (c *localIssueClient) backend() string {
	return "local"
}

func (c *localIssueClient) fetchIssueStates(_ string, stateNames []string) ([]refill.LinearIssue, error) {
	return c.store.IssueStates(stateNames), nil
}

func (c *localIssueClient) promoteIssue(issueID string, _ string, stateName string) (bool, string, error) {
	if trim(stateName) == "" {
		return false, "", errors.New("missing activate state name")
	}
	now := time.Now
	if c.now != nil {
		now = c.now
	}
	updatedState, err := c.store.UpdateIssueState(issueID, stateName, now())
	if err != nil {
		return false, "", err
	}
	return true, updatedState, nil
}

func refillClientFromFlags(localIssuesPath string) (refillClient, error) {
	if path := resolvedLocalIssueStorePath(localIssuesPath); path != "" {
		store, err := refill.LoadLocalIssueStore(path)
		if err != nil {
			return nil, err
		}
		return &localIssueClient{store: store}, nil
	}
	client := &linearClient{apiKey: os.Getenv("LINEAR_API_KEY")}
	if trim(client.apiKey) == "" {
		return nil, errors.New("LINEAR_API_KEY is required when no local issue store is configured")
	}
	return client, nil
}

func resolvedLocalIssueStorePath(path string) string {
	if trim(path) != "" {
		return resolveRepoRelativePath(path)
	}
	for _, candidate := range []string{"local-issues.json", "../local-issues.json"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func resolveRepoRelativePath(path string) string {
	if trim(path) == "" || filepath.IsAbs(path) {
		return path
	}
	candidates := []string{path, filepath.Join("..", path)}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return path
}

func hasCLIFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
}

func runRefillOnce(queue *refill.ParallelIssueQueue, client refillClient, apply bool, refreshURL string, targetOverride *int) error {
	refillStates := make([]string, 0, len(queue.RefillStates()))
	for state := range queue.RefillStates() {
		refillStates = append(refillStates, state)
	}
	issues, err := client.fetchIssueStates(queue.ProjectSlug(), append([]string{"In Progress"}, refillStates...))
	if err != nil {
		return err
	}
	stateMap := refill.IssueStateMap(issues)
	active := map[string]struct{}{}
	issueIDs := map[string]string{}
	for _, issue := range issues {
		if issue.Identifier != "" && issue.ID != "" {
			issueIDs[issue.Identifier] = issue.ID
		}
		if issue.StateName == "In Progress" {
			active[issue.Identifier] = struct{}{}
		}
	}
	candidates := queue.SelectCandidates(active, stateMap, targetOverride)
	target := queue.TargetInProgress()
	if targetOverride != nil {
		target = *targetOverride
	}
	payload := map[string]any{
		"active_in_progress": refill.SortedActive(issues),
		"backend":            client.backend(),
		"target_in_progress": target,
		"candidates":         candidates,
		"mode":               map[bool]string{true: "apply", false: "dry-run"}[apply],
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return err
	}
	if !apply {
		return nil
	}
	for _, identifier := range candidates {
		issueID, ok := issueIDs[identifier]
		if !ok {
			return fmt.Errorf("missing issue ID for candidate %s", identifier)
		}
		success, stateName, err := client.promoteIssue(issueID, queue.ActivateStateID(), queue.ActivateStateName())
		if err != nil {
			return err
		}
		if success {
			fmt.Printf("promoted %s -> %s\n", identifier, stateName)
			if err := triggerRefresh(refreshURL); err != nil {
				return err
			}
		}
	}
	return nil
}

func triggerRefresh(refreshURL string) error {
	if trim(refreshURL) == "" {
		return nil
	}
	request, err := http.NewRequest(http.MethodPost, refreshURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("refresh HTTP %d: %s", response.StatusCode, trim(string(body)))
	}
	return nil
}

func emit(payload map[string]any, asJSON bool, exitCode int) error {
	if asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(payload); err != nil {
			return err
		}
	} else {
		for key, value := range payload {
			fmt.Printf("%s=%v\n", key, value)
		}
	}
	if exitCode != 0 {
		return exitError(exitCode)
	}
	return nil
}

func buildCloseoutComment(summary string, validation string, commitSHA string, prURL string) string {
	lines := []string{}
	appendSection := func(title string, value string) {
		value = trim(value)
		if value == "" {
			return
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, title+":")
		lines = append(lines, value)
	}
	appendSection("What changed", summary)
	appendSection("Validation", validation)
	if trim(commitSHA) != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Commit SHA: "+trim(commitSHA))
	}
	if trim(prURL) != "" {
		lines = append(lines, "PR URL: "+trim(prURL))
	}
	return trim(joinLines(lines))
}

type exitError int

func (e exitError) Error() string {
	return "command failed"
}

func statusToMap(status string, repo githubsync.RepoSyncStatus) map[string]any {
	return map[string]any{
		"status":        status,
		"branch":        repo.Branch,
		"local_sha":     repo.LocalSHA,
		"remote_sha":    repo.RemoteSHA,
		"dirty":         repo.Dirty,
		"remote_exists": repo.RemoteExists,
		"synced":        repo.Synced,
		"pushed":        repo.Pushed,
	}
}

func structToMap(value any) map[string]any {
	body, _ := json.Marshal(value)
	result := map[string]any{}
	_ = json.Unmarshal(body, &result)
	return result
}

func mergeMap(left map[string]any, right map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range left {
		result[key] = value
	}
	for key, value := range right {
		result[key] = value
	}
	return result
}

func splitCSV(value string) []string {
	items := []string{}
	current := ""
	for _, r := range value {
		if r == ',' {
			current = trim(current)
			if current != "" {
				items = append(items, current)
			}
			current = ""
			continue
		}
		current += string(r)
	}
	current = trim(current)
	if current != "" {
		items = append(items, current)
	}
	return items
}

func absPath(path string) string {
	if path == "" {
		path = "."
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}

func trim(value string) string {
	return string(bytes.TrimSpace([]byte(value)))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func atoiPointer(value string) *int {
	if trim(value) == "" {
		return nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &number
}

func joinLines(lines []string) string {
	return string(bytes.Join(func() [][]byte {
		parts := make([][]byte, 0, len(lines))
		for _, line := range lines {
			parts = append(parts, []byte(line))
		}
		return parts
	}(), []byte("\n")))
}
