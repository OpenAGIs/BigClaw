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
	"time"

	"bigclaw-go/internal/bootstrap"
	"bigclaw-go/internal/githubsync"
	"bigclaw-go/internal/legacyshim"
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
	fetchIssueStates(projectSlug string, stateNames []string) ([]refill.TrackedIssue, error)
	promoteIssue(issueID string, stateID string, stateName string) (bool, string, error)
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		printRootUsage(os.Stdout)
		return 0
	}
	var err error
	switch args[0] {
	case "github-sync":
		err = runGitHubSync(args[1:])
	case "workspace":
		err = runWorkspace(args[1:])
	case "refill":
		err = runRefill(args[1:])
	case "local-issues":
		err = runLocalIssues(args[1:])
	case "legacy-python":
		err = runLegacyPython(args[1:])
	default:
		err = fmt.Errorf("unknown command: %s", args[0])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runLegacyPython(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl legacy-python <compile-check> [flags]\n")
		return nil
	}
	if len(args) == 0 {
		return errors.New("usage: bigclawctl legacy-python <compile-check> [flags]")
	}
	command := args[0]
	flags := flag.NewFlagSet("legacy-python "+command, flag.ContinueOnError)
	repoRoot := flags.String("repo", "..", "repo root")
	pythonBin := flags.String("python", "python3", "python executable")
	asJSON := flags.Bool("json", false, "json")
	if helpText, err := parseFlagsWithHelp(flags, fmt.Sprintf("usage: bigclawctl legacy-python %s [flags]", command), args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	switch command {
	case "compile-check":
		result, err := legacyshim.CompileCheck(absPath(*repoRoot), *pythonBin)
		if err != nil {
			payload := map[string]any{
				"status": "error",
				"repo":   absPath(*repoRoot),
				"python": *pythonBin,
				"files":  result.Files,
				"error":  err.Error(),
			}
			if result.Output != "" {
				payload["output"] = result.Output
			}
			return emit(payload, *asJSON, 1)
		}
		payload := map[string]any{
			"status": "ok",
			"repo":   absPath(*repoRoot),
			"python": result.Python,
			"files":  result.Files,
		}
		if result.Output != "" {
			payload["output"] = result.Output
		}
		return emit(payload, *asJSON, 0)
	default:
		return fmt.Errorf("unknown legacy-python subcommand: %s", command)
	}
}

func runGitHubSync(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl github-sync <install|status|sync> [flags]\n")
		return nil
	}
	if len(args) == 0 {
		return errors.New("usage: bigclawctl github-sync <install|status|sync> [flags]")
	}
	command := args[0]
	flags := flag.NewFlagSet("github-sync "+command, flag.ContinueOnError)
	repo := flags.String("repo", ".", "repository path")
	remote := flags.String("remote", "origin", "git remote")
	hooksPath := flags.String("hooks-path", ".githooks", "hooks path")
	allowDirty := flags.Bool("allow-dirty", false, "allow dirty")
	requireClean := flags.Bool("require-clean", false, "require clean")
	requireSynced := flags.Bool("require-synced", false, "require synced")
	asJSON := flags.Bool("json", false, "json")
	if helpText, err := parseFlagsWithHelp(flags, fmt.Sprintf("usage: bigclawctl github-sync %s [flags]", command), args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
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

func runWorkspace(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]\n")
		return nil
	}
	if len(args) == 0 {
		return errors.New("usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]")
	}
	command := args[0]
	flags := flag.NewFlagSet("workspace "+command, flag.ContinueOnError)
	repoRoot := flags.String("repo", "..", "repo root")
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
	if helpText, err := parseFlagsWithHelp(flags, fmt.Sprintf("usage: bigclawctl workspace %s [flags]", command), args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	resolvedRepoRoot := absPath(*repoRoot)
	resolvedWorkspace := resolvePathAgainstRepoRoot(resolvedRepoRoot, *workspace)
	resolvedWorkspaceRoot := resolvePathAgainstRepoRoot(resolvedRepoRoot, *workspaceRoot)
	resolvedReportPath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *reportPath)
	switch command {
	case "bootstrap":
		status, err := bootstrap.BootstrapWorkspace(resolvedWorkspace, *issue, *repoURL, *defaultBranch, *cacheRoot, *cacheBase, *cacheKey)
		if err != nil {
			return emit(map[string]any{"status": "error", "workspace": absPath(resolvedWorkspace), "error": err.Error()}, *asJSON, 1)
		}
		return emit(mergeMap(map[string]any{"status": "ok"}, structToMap(status)), *asJSON, 0)
	case "cleanup":
		status, err := bootstrap.CleanupWorkspace(resolvedWorkspace, *issue, *repoURL, *defaultBranch, *cacheRoot, *cacheBase, *cacheKey)
		if err != nil {
			return emit(map[string]any{"status": "error", "workspace": absPath(resolvedWorkspace), "error": err.Error()}, *asJSON, 1)
		}
		return emit(mergeMap(map[string]any{"status": "ok"}, structToMap(status)), *asJSON, 0)
	case "validate":
		issues := splitCSV(*issuesCSV)
		report, err := bootstrap.BuildValidationReport(*repoURL, resolvedWorkspaceRoot, issues, *defaultBranch, *cacheRoot, *cacheBase, *cacheKey, *cleanup)
		if err != nil {
			return err
		}
		if resolvedReportPath != "" {
			if _, err := bootstrap.WriteValidationReport(report, resolvedReportPath); err != nil {
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

type refillFlags struct {
	repoRoot         *string
	queuePath        *string
	markdownPath     *string
	localIssuesPath  *string
	targetInProgress *int
	watch            *bool
	interval         *int
	apply            *bool
	syncQueueStatus  *bool
	refreshURL       *string
}

type refillSeedFlags struct {
	repoRoot         *string
	queuePath        *string
	markdownPath     *string
	localIssuesPath  *string
	identifier       *string
	title            *string
	track            *string
	description      *string
	stateName        *string
	priority         *int
	labelsCSV        *string
	assigned         *bool
	createdAt        *string
	recentBatch      *string
	setStateIfExists *bool
	asJSON           *bool
}

func newRefillFlagSet() (*flag.FlagSet, refillFlags) {
	flags := flag.NewFlagSet("refill", flag.ContinueOnError)
	return flags, refillFlags{
		repoRoot:         flags.String("repo", "..", "repo root"),
		queuePath:        flags.String("queue", "docs/parallel-refill-queue.json", "queue path"),
		markdownPath:     flags.String("markdown", "docs/parallel-refill-queue.md", "human-readable queue markdown path"),
		localIssuesPath:  flags.String("local-issues", "", "local issue store path"),
		targetInProgress: flags.Int("target-in-progress", -1, "override target"),
		watch:            flags.Bool("watch", false, "watch"),
		interval:         flags.Int("interval", 20, "interval"),
		apply:            flags.Bool("apply", false, "apply"),
		syncQueueStatus:  flags.Bool("sync-queue-status", false, "sync queue issue statuses and recent batches from local tracker metadata (local backend only; requires --apply to write)"),
		refreshURL:       flags.String("refresh-url", "", "refresh url"),
	}
}

func newRefillSeedFlagSet() (*flag.FlagSet, refillSeedFlags) {
	flags := flag.NewFlagSet("refill seed", flag.ContinueOnError)
	return flags, refillSeedFlags{
		repoRoot:         flags.String("repo", "..", "repo root"),
		queuePath:        flags.String("queue", "docs/parallel-refill-queue.json", "queue path"),
		markdownPath:     flags.String("markdown", "docs/parallel-refill-queue.md", "human-readable queue markdown path"),
		localIssuesPath:  flags.String("local-issues", "", "local issue store path"),
		identifier:       flags.String("identifier", "", "issue identifier (e.g. BIG-PAR-238)"),
		title:            flags.String("title", "", "issue title"),
		track:            flags.String("track", "Go Mainline Follow-ups", "queue track"),
		description:      flags.String("description", "", "local issue description"),
		stateName:        flags.String("state", "Todo", "state name"),
		priority:         flags.Int("priority", 3, "priority (1=urgent, 4=low)"),
		labelsCSV:        flags.String("labels", "", "comma-separated labels"),
		assigned:         flags.Bool("assigned-to-worker", true, "assigned to worker"),
		createdAt:        flags.String("created-at", "", "RFC3339 timestamp"),
		recentBatch:      flags.String("recent-batch", "", "optional recent batch bucket (active|standby|completed)"),
		setStateIfExists: flags.Bool("set-state-if-exists", false, "update the local issue state when the issue already exists"),
		asJSON:           flags.Bool("json", false, "json"),
	}
}

func runRefill(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printRefillUsage(os.Stdout)
		return nil
	}
	if len(args[0]) > 0 && args[0][0] != '-' {
		switch args[0] {
		case "seed":
			return runRefillSeed(args[1:])
		default:
			return fmt.Errorf("unknown refill subcommand: %s", args[0])
		}
	}

	flags, options := newRefillFlagSet()
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl refill [flags]\n       bigclawctl refill seed [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	resolvedRepoRoot := absPath(*options.repoRoot)
	resolvedQueuePath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *options.queuePath)
	queue, err := refill.LoadQueue(resolvedQueuePath)
	if err != nil {
		return err
	}
	resolvedLocalIssuesPath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *options.localIssuesPath)
	client, err := refillClientFromFlags(resolvedRepoRoot, resolvedLocalIssuesPath)
	if err != nil {
		return err
	}
	runOnce := func() error {
		var override *int
		if *options.targetInProgress >= 0 {
			override = options.targetInProgress
		}
		resolvedMarkdownPath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *options.markdownPath)
		return runRefillOnce(queue, client, *options.apply, *options.refreshURL, override, *options.syncQueueStatus, resolvedQueuePath, resolvedMarkdownPath, resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedLocalIssuesPath))
	}
	if !*options.watch {
		return runOnce()
	}
	for {
		if err := runOnce(); err != nil {
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
		time.Sleep(time.Duration(*options.interval) * time.Second)
	}
}

func runRefillSeed(args []string) error {
	flags, options := newRefillSeedFlagSet()
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl refill seed [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	visitedFlags := map[string]bool{}
	flags.Visit(func(flag *flag.Flag) {
		visitedFlags[flag.Name] = true
	})

	identifier := trim(*options.identifier)
	if identifier == "" {
		return errors.New("identifier is required")
	}
	title := trim(*options.title)
	if title == "" {
		return errors.New("title is required")
	}
	track := trim(*options.track)
	if track == "" {
		return errors.New("track is required")
	}
	stateName := trim(*options.stateName)
	if stateName == "" {
		stateName = "Todo"
	}
	when, err := parseOptionalTime(*options.createdAt)
	if err != nil {
		return err
	}

	resolvedRepoRoot := absPath(*options.repoRoot)
	resolvedQueuePath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *options.queuePath)
	resolvedMarkdownPath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *options.markdownPath)
	queue, err := refill.LoadQueue(resolvedQueuePath)
	if err != nil {
		return err
	}
	queueAction, orderAdded, err := queue.UpsertIssue(refill.IssueRecord{
		Identifier: identifier,
		Title:      title,
		Track:      track,
		Status:     stateName,
	})
	if err != nil {
		return err
	}
	recentBatchUpdated := false
	if trim(*options.recentBatch) != "" {
		recentBatchUpdated, err = queue.SetRecentBatch(*options.recentBatch, identifier)
		if err != nil {
			return fmt.Errorf("set recent batch: %w", err)
		}
	}
	if err := queue.Save(); err != nil {
		return err
	}
	markdownWritten, err := queue.SaveMarkdown(resolvedMarkdownPath, when)
	if err != nil {
		return err
	}

	resolvedLocalIssuesPath := resolvedLocalIssueStorePath(resolvedRepoRoot, resolvePathAgainstRepoRoot(resolvedRepoRoot, *options.localIssuesPath))
	localAction := "skipped"
	localState := stateName
	if trim(resolvedLocalIssuesPath) != "" {
		store, err := refill.LoadLocalIssueStore(resolvedLocalIssuesPath)
		if err != nil {
			return err
		}
		if existing, found := store.FindIssue(identifier); found {
			localAction = "exists"
			localState = existing.State
			labels := []string{}
			for _, label := range splitCSV(*options.labelsCSV) {
				if trimmed := trim(label); trimmed != "" {
					labels = append(labels, trimmed)
				}
			}
			updatedIssue, metadataChanged, err := store.UpdateIssue(identifier, refill.LocalIssueUpdateParams{
				Title:            stringPointerIfVisited(title, visitedFlags["title"]),
				Description:      stringPointerIfVisited(*options.description, visitedFlags["description"]),
				Priority:         intPointerIfVisited(*options.priority, visitedFlags["priority"]),
				Labels:           stringSlicePointerIfVisited(labels, visitedFlags["labels"]),
				AssignedToWorker: boolPointerIfVisited(*options.assigned, visitedFlags["assigned-to-worker"]),
			}, when)
			if err != nil {
				return err
			}
			if metadataChanged {
				existing = updatedIssue
				localAction = "updated"
			}
			if *options.setStateIfExists && refill.NormalizeStateName(existing.State) != refill.NormalizeStateName(stateName) {
				localState, err = store.UpdateIssueState(identifier, stateName, when)
				if err != nil {
					return err
				}
				localAction = "updated"
			}
		} else {
			labels := []string{}
			for _, label := range splitCSV(*options.labelsCSV) {
				if trimmed := trim(label); trimmed != "" {
					labels = append(labels, trimmed)
				}
			}
			created, err := store.CreateIssue(refill.LocalIssueCreateParams{
				Identifier:       identifier,
				Title:            title,
				Description:      *options.description,
				State:            stateName,
				Priority:         *options.priority,
				Labels:           labels,
				AssignedToWorker: *options.assigned,
				CreatedAt:        when,
			})
			if err != nil {
				return err
			}
			localAction = "created"
			localState = created.State
		}
	}

	payload := map[string]any{
		"status":             "ok",
		"backend":            "local",
		"issue":              identifier,
		"state":              localState,
		"queue_action":       queueAction,
		"queue_order_added":  orderAdded,
		"queue_path":         absPath(resolvedQueuePath),
		"markdown_path":      absPath(resolvedMarkdownPath),
		"markdown_written":   markdownWritten,
		"local_issue_action": localAction,
	}
	if trim(*options.recentBatch) != "" {
		payload["recent_batch"] = trim(*options.recentBatch)
		payload["recent_batch_updated"] = recentBatchUpdated
	}
	if trim(resolvedLocalIssuesPath) != "" {
		payload["local_issues"] = absPath(resolvedLocalIssuesPath)
	}
	return emit(payload, *options.asJSON, 0)
}

func runLocalIssues(args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = os.Stdout.WriteString("usage: bigclawctl local-issues <list|create|ensure|set-state|comment> [flags]\n")
		return nil
	}
	command := args[0]
	switch command {
	case "list":
		flags := flag.NewFlagSet("local-issues "+command, flag.ContinueOnError)
		repoRoot := flags.String("repo", "..", "repo root")
		storePath := flags.String("local-issues", "", "local issue store path")
		statesCSV := flags.String("states", "", "comma-separated state filter")
		asJSON := flags.Bool("json", false, "json")
		if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues list [flags]", args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				_, _ = os.Stdout.WriteString(helpText)
				return nil
			}
			return err
		}
		resolvedRepoRoot := absPath(*repoRoot)
		resolvedStorePath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *storePath)
		store, err := refill.LoadLocalIssueStore(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath))
		if err != nil {
			return err
		}
		issues := store.Issues()
		stateFilter := map[string]struct{}{}
		for _, state := range splitCSV(*statesCSV) {
			if normalized := refill.NormalizeStateName(state); normalized != "" {
				stateFilter[normalized] = struct{}{}
			}
		}
		filtered := make([]refill.LocalIssue, 0, len(issues))
		for _, issue := range issues {
			if len(stateFilter) != 0 {
				if _, ok := stateFilter[refill.NormalizeStateName(issue.State)]; !ok {
					continue
				}
			}
			filtered = append(filtered, issue)
		}
		if *asJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(map[string]any{
				"status":       "ok",
				"backend":      "local",
				"local_issues": absPath(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath)),
				"issues":       filtered,
			})
		}
		for _, issue := range filtered {
			fmt.Printf("%s\t%s\t%s\n", issue.Identifier, issue.State, issue.Title)
		}
		return nil
	case "create":
		flags := flag.NewFlagSet("local-issues "+command, flag.ContinueOnError)
		repoRoot := flags.String("repo", "..", "repo root")
		storePath := flags.String("local-issues", "", "local issue store path")
		id := flags.String("id", "", "issue id (defaults to lowercased identifier)")
		identifier := flags.String("identifier", "", "issue identifier (e.g. BIG-PAR-104)")
		title := flags.String("title", "", "issue title")
		description := flags.String("description", "", "issue description")
		stateName := flags.String("state", "Todo", "state name")
		priority := flags.Int("priority", 3, "priority (1=urgent, 4=low)")
		labelsCSV := flags.String("labels", "", "comma-separated labels")
		assigned := flags.Bool("assigned-to-worker", true, "assigned to worker")
		createdAt := flags.String("created-at", "", "RFC3339 timestamp")
		asJSON := flags.Bool("json", false, "json")
		if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues create [flags]", args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				_, _ = os.Stdout.WriteString(helpText)
				return nil
			}
			return err
		}
		resolvedRepoRoot := absPath(*repoRoot)
		resolvedStorePath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *storePath)
		store, err := refill.LoadLocalIssueStore(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath))
		if err != nil {
			return err
		}
		when, err := parseOptionalTime(*createdAt)
		if err != nil {
			return err
		}
		labels := []string{}
		for _, label := range splitCSV(*labelsCSV) {
			if trimmed := trim(label); trimmed != "" {
				labels = append(labels, trimmed)
			}
		}
		created, err := store.CreateIssue(refill.LocalIssueCreateParams{
			ID:               *id,
			Identifier:       *identifier,
			Title:            *title,
			Description:      *description,
			State:            *stateName,
			Priority:         *priority,
			Labels:           labels,
			AssignedToWorker: *assigned,
			CreatedAt:        when,
		})
		if err != nil {
			return err
		}
		return emit(map[string]any{
			"status":       "ok",
			"backend":      "local",
			"issue":        created.Identifier,
			"state":        created.State,
			"local_issues": absPath(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath)),
		}, *asJSON, 0)
	case "ensure":
		flags := flag.NewFlagSet("local-issues "+command, flag.ContinueOnError)
		repoRoot := flags.String("repo", "..", "repo root")
		storePath := flags.String("local-issues", "", "local issue store path")
		id := flags.String("id", "", "issue id (defaults to lowercased identifier)")
		identifier := flags.String("identifier", "", "issue identifier (e.g. BIG-PAR-104)")
		title := flags.String("title", "", "issue title (defaults to identifier)")
		description := flags.String("description", "", "issue description")
		stateName := flags.String("state", "Todo", "state name")
		priority := flags.Int("priority", 3, "priority (1=urgent, 4=low)")
		labelsCSV := flags.String("labels", "", "comma-separated labels")
		assigned := flags.Bool("assigned-to-worker", true, "assigned to worker")
		createdAt := flags.String("created-at", "", "RFC3339 timestamp")
		setStateIfExists := flags.Bool("set-state-if-exists", false, "update the issue state when the issue already exists")
		asJSON := flags.Bool("json", false, "json")
		if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues ensure [flags]", args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				_, _ = os.Stdout.WriteString(helpText)
				return nil
			}
			return err
		}
		visitedFlags := map[string]bool{}
		flags.Visit(func(flag *flag.Flag) {
			visitedFlags[flag.Name] = true
		})
		trimmedIdentifier := trim(*identifier)
		if trimmedIdentifier == "" {
			return errors.New("identifier is required")
		}
		resolvedRepoRoot := absPath(*repoRoot)
		resolvedStorePath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *storePath)
		store, err := refill.LoadLocalIssueStore(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath))
		if err != nil {
			return err
		}
		when, err := parseOptionalTime(*createdAt)
		if err != nil {
			return err
		}
		labels := []string{}
		for _, label := range splitCSV(*labelsCSV) {
			if trimmed := trim(label); trimmed != "" {
				labels = append(labels, trimmed)
			}
		}
		existing, found := store.FindIssue(trimmedIdentifier)
		action := "created"
		state := trim(*stateName)
		if state == "" {
			state = "Todo"
		}
		if found {
			action = "exists"
			updatedIssue, metadataChanged, err := store.UpdateIssue(trimmedIdentifier, refill.LocalIssueUpdateParams{
				Title:            stringPointerIfVisited(*title, visitedFlags["title"]),
				Description:      stringPointerIfVisited(*description, visitedFlags["description"]),
				Priority:         intPointerIfVisited(*priority, visitedFlags["priority"]),
				Labels:           stringSlicePointerIfVisited(labels, visitedFlags["labels"]),
				AssignedToWorker: boolPointerIfVisited(*assigned, visitedFlags["assigned-to-worker"]),
			}, when)
			if err != nil {
				return err
			}
			if metadataChanged {
				existing = updatedIssue
				action = "updated"
			}
			if *setStateIfExists && refill.NormalizeStateName(existing.State) != refill.NormalizeStateName(state) {
				if when.IsZero() {
					when = time.Now()
				}
				updatedState, err := store.UpdateIssueState(trimmedIdentifier, state, when)
				if err != nil {
					return err
				}
				existing.State = updatedState
				action = "updated"
			}
			return emit(map[string]any{
				"status":       "ok",
				"backend":      "local",
				"action":       action,
				"issue":        existing.Identifier,
				"state":        existing.State,
				"local_issues": absPath(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath)),
			}, *asJSON, 0)
		}
		if trim(*title) == "" {
			*title = trimmedIdentifier
		}
		created, err := store.CreateIssue(refill.LocalIssueCreateParams{
			ID:               *id,
			Identifier:       trimmedIdentifier,
			Title:            *title,
			Description:      *description,
			State:            state,
			Priority:         *priority,
			Labels:           labels,
			AssignedToWorker: *assigned,
			CreatedAt:        when,
		})
		if err != nil {
			return err
		}
		return emit(map[string]any{
			"status":       "ok",
			"backend":      "local",
			"action":       action,
			"issue":        created.Identifier,
			"state":        created.State,
			"local_issues": absPath(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath)),
		}, *asJSON, 0)
	case "set-state":
		flags := flag.NewFlagSet("local-issues "+command, flag.ContinueOnError)
		repoRoot := flags.String("repo", "..", "repo root")
		storePath := flags.String("local-issues", "", "local issue store path")
		issueRef := flags.String("issue", "", "issue id or identifier")
		stateName := flags.String("state", "", "state name")
		createdAt := flags.String("created-at", "", "RFC3339 timestamp")
		asJSON := flags.Bool("json", false, "json")
		if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues set-state [flags]", args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				_, _ = os.Stdout.WriteString(helpText)
				return nil
			}
			return err
		}
		if trim(*issueRef) == "" {
			return errors.New("issue is required")
		}
		if trim(*stateName) == "" {
			return errors.New("state is required")
		}
		when, err := parseOptionalTime(*createdAt)
		if err != nil {
			return err
		}
		resolvedRepoRoot := absPath(*repoRoot)
		resolvedStorePath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *storePath)
		store, err := refill.LoadLocalIssueStore(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath))
		if err != nil {
			return err
		}
		existing, found := store.FindIssue(*issueRef)
		if !found {
			return refill.ErrLocalIssueNotFound
		}
		if refill.NormalizeStateName(existing.State) == refill.NormalizeStateName(*stateName) {
			return emit(map[string]any{
				"status":       "ok",
				"backend":      "local",
				"action":       "exists",
				"issue":        existing.Identifier,
				"state":        existing.State,
				"local_issues": absPath(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath)),
			}, *asJSON, 0)
		}
		updatedState, err := store.UpdateIssueState(*issueRef, *stateName, when)
		if err != nil {
			return err
		}
		return emit(map[string]any{
			"status":       "ok",
			"backend":      "local",
			"issue":        *issueRef,
			"state":        updatedState,
			"local_issues": absPath(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath)),
		}, *asJSON, 0)
	case "comment":
		flags := flag.NewFlagSet("local-issues "+command, flag.ContinueOnError)
		repoRoot := flags.String("repo", "..", "repo root")
		storePath := flags.String("local-issues", "", "local issue store path")
		issueRef := flags.String("issue", "", "issue id or identifier")
		author := flags.String("author", "codex", "comment author")
		body := flags.String("body", "", "comment body")
		createdAt := flags.String("created-at", "", "RFC3339 timestamp")
		asJSON := flags.Bool("json", false, "json")
		if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues comment [flags]", args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				_, _ = os.Stdout.WriteString(helpText)
				return nil
			}
			return err
		}
		if trim(*issueRef) == "" {
			return errors.New("issue is required")
		}
		resolvedRepoRoot := absPath(*repoRoot)
		resolvedStorePath := resolvePathAgainstRepoRoot(resolvedRepoRoot, *storePath)
		store, err := refill.LoadLocalIssueStore(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath))
		if err != nil {
			return err
		}
		when, err := parseOptionalTime(*createdAt)
		if err != nil {
			return err
		}
		if err := store.AddComment(*issueRef, refill.LocalIssueComment{Author: *author, Body: *body, CreatedAt: when}); err != nil {
			return err
		}
		return emit(map[string]any{
			"status":       "ok",
			"backend":      "local",
			"issue":        *issueRef,
			"author":       *author,
			"body":         *body,
			"local_issues": absPath(resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath)),
		}, *asJSON, 0)
	default:
		return fmt.Errorf("unknown local-issues subcommand: %s", command)
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

func (c *linearClient) fetchIssueStates(projectSlug string, stateNames []string) ([]refill.TrackedIssue, error) {
	issues := []refill.TrackedIssue{}
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
			issues = append(issues, refill.TrackedIssue{ID: node.ID, Identifier: node.Identifier, StateName: node.State.Name})
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

func (c *localIssueClient) fetchIssueStates(_ string, stateNames []string) ([]refill.TrackedIssue, error) {
	if err := c.store.Reload(); err != nil {
		return nil, err
	}
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

func refillClientFromFlags(repoRoot string, localIssuesPath string) (refillClient, error) {
	if path := resolvedLocalIssueStorePath(repoRoot, localIssuesPath); path != "" {
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

func resolvedLocalIssueStorePath(repoRoot string, path string) string {
	if trim(path) != "" {
		path = resolvePathAgainstRepoRoot(repoRoot, path)
		return resolveRepoRelativePath(path)
	}
	candidates := []string{"local-issues.json", "../local-issues.json"}
	if trim(repoRoot) != "" {
		candidates = append([]string{filepath.Join(repoRoot, "local-issues.json")}, candidates...)
	}
	for _, candidate := range candidates {
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

func resolvePathAgainstRepoRoot(repoRoot string, path string) string {
	path = trim(path)
	if path == "" || filepath.IsAbs(path) || (len(path) > 0 && path[0] == '~') {
		return path
	}
	repoRoot = trim(repoRoot)
	if repoRoot == "" {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func parseOptionalTime(value string) (time.Time, error) {
	if trim(value) == "" {
		return time.Now(), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse created-at: %w", err)
	}
	return parsed, nil
}

func runRefillOnce(queue *refill.ParallelIssueQueue, client refillClient, apply bool, refreshURL string, targetOverride *int, syncQueueStatus bool, queuePath string, markdownPath string, localIssuesPath string) error {
	queueStatusUpdates := 0
	queueRecentBatchUpdates := 0
	queueStatusWritten := false
	queueStatusSynced := false
	recentBatchesSynced := false
	recentBatchesUpdated := false
	recentBatchesWritten := false
	markdownWritten := false
	var allIssues []refill.TrackedIssue
	var err error
	if client.backend() == "local" {
		allIssues, err = client.fetchIssueStates(queue.ProjectSlug(), nil)
		if err != nil {
			return err
		}
	}
	if client.backend() == "local" {
		issueStates := refill.IssueStateMap(allIssues)
		queueStatusUpdates = queue.StatusSyncUpdatesForStates(issueStates)
		queueRecentBatchUpdates = queue.RecentBatchSyncUpdatesForStates(issueStates)
		queueStatusSynced = queueStatusUpdates == 0
		recentBatchesSynced = queueRecentBatchUpdates == 0
		if syncQueueStatus {
			queueStatusUpdates = queue.SyncStatusFromStates(issueStates)
			queueRecentBatchUpdates = queue.SyncRecentBatchesFromStates(issueStates)
			if apply {
				queueStatusSynced = true
				recentBatchesSynced = true
			}
			if apply && (queueStatusUpdates > 0 || queueRecentBatchUpdates > 0) {
				if err := queue.Save(); err != nil {
					return err
				}
				queueStatusWritten = queueStatusUpdates > 0
				recentBatchesWritten = queueRecentBatchUpdates > 0
			}
		}
	}

	statesToFetch := queue.FetchStateNames()
	issues := []refill.TrackedIssue{}
	if client.backend() == "local" {
		issues = allIssues
	} else {
		issues, err = client.fetchIssueStates(queue.ProjectSlug(), statesToFetch)
		if err != nil {
			return err
		}
	}
	stateMap := refill.IssueStateMap(issues)
	liveStateMap := stateMap
	markdownGeneratedAt := time.Now().UTC()
	if client.backend() == "local" {
		liveStateMap = refill.IssueStateMap(allIssues)
		if apply {
			recentBatchesUpdated = queue.RefreshRecentBatchesFromStates(liveStateMap)
			if recentBatchesUpdated {
				recentBatchesSynced = true
			}
		}
	}
	if apply && (queueStatusUpdates > 0 || recentBatchesUpdated) {
		if err := queue.Save(); err != nil {
			return err
		}
		queueStatusWritten = queueStatusWritten || queueStatusUpdates > 0
		recentBatchesWritten = recentBatchesWritten || recentBatchesUpdated
	}
	active := map[string]struct{}{}
	issueIDs := map[string]string{}
	for _, issue := range issues {
		if issue.Identifier != "" && issue.ID != "" {
			issueIDs[issue.Identifier] = issue.ID
		}
		if refill.NormalizeStateName(issue.StateName) == refill.NormalizeStateName(queue.ActivateStateName()) {
			active[issue.Identifier] = struct{}{}
		}
	}
	candidates := queue.SelectCandidates(active, stateMap, targetOverride)
	if client.backend() == "local" && apply && trim(markdownPath) != "" {
		previewQueue, err := queue.Clone()
		if err != nil {
			return err
		}
		projectedStates := map[string]string{}
		for identifier, state := range liveStateMap {
			projectedStates[identifier] = state
		}
		for _, identifier := range candidates {
			projectedStates[identifier] = queue.ActivateStateName()
		}
		previewQueue.SyncStatusFromStates(projectedStates)
		previewQueue.RefreshRecentBatchesFromStates(projectedStates)
		markdownWritten, err = previewQueue.MarkdownNeedsWrite(markdownPath, markdownGeneratedAt)
		if err != nil {
			return err
		}
	}
	target := queue.TargetInProgress()
	if targetOverride != nil {
		target = *targetOverride
	}
	payload := map[string]any{
		"active_in_progress":         refill.SortedActive(issues, queue.ActivateStateName()),
		"backend":                    client.backend(),
		"target_in_progress":         target,
		"candidates":                 candidates,
		"mode":                       map[bool]string{true: "apply", false: "dry-run"}[apply],
		"recent_batches_synced":      recentBatchesSynced,
		"recent_batches_updated":     recentBatchesUpdated,
		"recent_batches_written":     recentBatchesWritten,
		"queue_status_synced":        queueStatusSynced,
		"queue_status_updates":       queueStatusUpdates,
		"queue_recent_batch_updates": queueRecentBatchUpdates,
		"queue_status_written":       queueStatusWritten,
		"queue_path":                 absPath(queuePath),
		"markdown_path":              absPath(markdownPath),
		"markdown_written":           markdownWritten,
	}
	if trim(localIssuesPath) != "" {
		payload["local_issues_path"] = absPath(localIssuesPath)
	}
	queueRunnable := queue.RunnableCount()
	if client.backend() == "local" {
		queueRunnable = queue.RunnableCountForStates(liveStateMap)
	}
	payload["queue_runnable"] = queueRunnable
	payload["queue_drained"] = queueRunnable == 0
	if queueRunnable == 0 {
		payload["warning"] = "refill queue drained: no runnable identifiers in docs/parallel-refill-queue.json"
		payload["next_steps"] = []string{
			"Seed the next BIG-PAR identifiers with `bash scripts/ops/bigclawctl refill seed --local-issues local-issues.json --identifier BIG-PAR-XXX --title \"...\" --state Todo --recent-batch standby --json`, or add them manually to docs/parallel-refill-queue.json.",
			"Optionally align queue metadata: `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json --sync-queue-status`.",
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return err
	}
	if !apply {
		return nil
	}
	promoted := false
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
			promoted = true
			fmt.Printf("promoted %s -> %s\n", identifier, stateName)
			if refreshURL != "" {
				request, err := http.NewRequest(http.MethodPost, refreshURL, bytes.NewReader([]byte("{}")))
				if err != nil {
					return err
				}
				request.Header.Set("Content-Type", "application/json")
				response, err := http.DefaultClient.Do(request)
				if err != nil {
					return err
				}
				response.Body.Close()
			}
		}
	}
	if client.backend() == "local" && trim(markdownPath) != "" {
		latestIssues := allIssues
		if promoted || len(latestIssues) == 0 {
			latestIssues, err = client.fetchIssueStates(queue.ProjectSlug(), nil)
			if err != nil {
				return err
			}
		}
		latestStates := refill.IssueStateMap(latestIssues)
		postStatusUpdates := queue.SyncStatusFromStates(latestStates)
		postRecentUpdates := queue.RefreshRecentBatchesFromStates(latestStates)
		if postStatusUpdates > 0 || postRecentUpdates {
			if err := queue.Save(); err != nil {
				return err
			}
		}
		markdownWritten, err = queue.SaveMarkdown(markdownPath, markdownGeneratedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func emit(payload map[string]any, asJSON bool, exitCode int) error {
	if asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)
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

type exitError int

func (e exitError) Error() string {
	return "command failed"
}

func statusToMap(status string, repo githubsync.RepoSyncStatus) map[string]any {
	return map[string]any{
		"status":         status,
		"branch":         repo.Branch,
		"detached":       repo.Detached,
		"local_sha":      repo.LocalSHA,
		"remote_sha":     repo.RemoteSHA,
		"dirty":          repo.Dirty,
		"remote_exists":  repo.RemoteExists,
		"synced":         repo.Synced,
		"pushed":         repo.Pushed,
		"relation_known": repo.RelationKnown,
		"ahead":          repo.Ahead,
		"behind":         repo.Behind,
		"diverged":       repo.Diverged,
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

func isHelpToken(token string) bool {
	switch token {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func parseFlagsWithHelp(flags *flag.FlagSet, usageLine string, args []string) (string, error) {
	var buf bytes.Buffer
	flags.SetOutput(&buf)
	flags.Usage = func() {
		fmt.Fprintln(&buf, usageLine)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return buf.String(), err
	}
	return "", nil
}

func printRefillUsage(w io.Writer) {
	flags, _ := newRefillFlagSet()
	helpText, _ := parseFlagsWithHelp(flags, "usage: bigclawctl refill [flags]\n       bigclawctl refill seed [flags]", []string{"--help"})
	_, _ = io.WriteString(w, helpText)
	_, _ = io.WriteString(w, "\nsubcommands:\n")
	_, _ = io.WriteString(w, "  seed    add or update a queue entry and matching local tracker issue\n")
}

func printRootUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: bigclawctl <github-sync|workspace|refill|local-issues|legacy-python> ...")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  github-sync     install/sync/status hooks and branch sync state")
	fmt.Fprintln(w, "  workspace       bootstrap/cleanup/validate workspaces using the shared mirror")
	fmt.Fprintln(w, "  refill          promote issues to maintain target in-progress count")
	fmt.Fprintln(w, "  local-issues    manage the repo-native issue store in local-issues.json")
	fmt.Fprintln(w, "  legacy-python   validate frozen Python compatibility shims")
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

func stringPointerIfVisited(value string, visited bool) *string {
	if !visited {
		return nil
	}
	return &value
}

func intPointerIfVisited(value int, visited bool) *int {
	if !visited {
		return nil
	}
	return &value
}

func boolPointerIfVisited(value bool, visited bool) *bool {
	if !visited {
		return nil
	}
	return &value
}

func stringSlicePointerIfVisited(value []string, visited bool) *[]string {
	if !visited {
		return nil
	}
	copyValue := append([]string{}, value...)
	return &copyValue
}
