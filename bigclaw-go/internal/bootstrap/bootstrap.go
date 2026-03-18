package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	cacheRemote           = "cache"
	bootstrapBranchPrefix = "symphony"
)

type CacheBootstrapState struct {
	CacheRoot     string `json:"cache_root"`
	CacheKey      string `json:"cache_key"`
	MirrorPath    string `json:"mirror_path"`
	SeedPath      string `json:"seed_path"`
	MirrorCreated bool   `json:"mirror_created"`
	SeedCreated   bool   `json:"seed_created"`
}

type WorkspaceBootstrapStatus struct {
	Workspace       string `json:"workspace"`
	Branch          string `json:"branch"`
	CacheRoot       string `json:"cache_root"`
	CacheKey        string `json:"cache_key"`
	MirrorPath      string `json:"mirror_path"`
	SeedPath        string `json:"seed_path"`
	Reused          bool   `json:"reused"`
	CacheReused     bool   `json:"cache_reused"`
	CloneSuppressed bool   `json:"clone_suppressed"`
	MirrorCreated   bool   `json:"mirror_created"`
	SeedCreated     bool   `json:"seed_created"`
	WorkspaceMode   string `json:"workspace_mode"`
	Removed         bool   `json:"removed"`
}

type ValidationSummary struct {
	WorkspaceCount               int      `json:"workspace_count"`
	UniqueCacheRoots             []string `json:"unique_cache_roots"`
	UniqueMirrorPaths            []string `json:"unique_mirror_paths"`
	UniqueSeedPaths              []string `json:"unique_seed_paths"`
	SingleCacheRootReused        bool     `json:"single_cache_root_reused"`
	SingleMirrorReused           bool     `json:"single_mirror_reused"`
	SingleSeedReused             bool     `json:"single_seed_reused"`
	MirrorCreations              int      `json:"mirror_creations"`
	SeedCreations                int      `json:"seed_creations"`
	CloneSuppressedAfterFirst    bool     `json:"clone_suppressed_after_first"`
	CacheReusedAfterFirst        bool     `json:"cache_reused_after_first"`
	AllWorkspacesCreatedWorktree bool     `json:"all_workspaces_created_via_worktree"`
	CleanupPreservedCache        bool     `json:"cleanup_preserved_cache"`
}

type ValidationReport struct {
	RepoURL          string                     `json:"repo_url"`
	DefaultBranch    string                     `json:"default_branch"`
	WorkspaceRoot    string                     `json:"workspace_root"`
	IssueIdentifiers []string                   `json:"issue_identifiers"`
	BootstrapResults []WorkspaceBootstrapStatus `json:"bootstrap_results"`
	CleanupResults   []WorkspaceBootstrapStatus `json:"cleanup_results"`
	Summary          ValidationSummary          `json:"summary"`
}

type commandResult struct {
	stdout     string
	stderr     string
	returnCode int
}

func run(command []string, cwd string) commandResult {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	result := commandResult{stdout: string(output), stderr: string(output)}
	if err == nil {
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.returnCode = exitErr.ExitCode()
		return result
	}
	result.returnCode = 1
	return result
}

func git(repo string, args ...string) commandResult {
	return run(append([]string{"git"}, args...), repo)
}

func requireGit(repo string, args ...string) (string, error) {
	result := git(repo, args...)
	if result.returnCode != 0 {
		detail := trim(result.stderr)
		if detail == "" {
			detail = trim(result.stdout)
		}
		if detail == "" {
			detail = fmt.Sprintf("git %s failed", join(args))
		}
		return "", errors.New(detail)
	}
	return trim(result.stdout), nil
}

func SanitizeIssueIdentifier(identifier string) string {
	raw := trim(identifier)
	if raw == "" {
		raw = "issue"
	}
	result := make([]rune, 0, len(raw))
	for _, character := range raw {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '.' || character == '-' || character == '_' {
			result = append(result, character)
			continue
		}
		result = append(result, '_')
	}
	return string(result)
}

func BootstrapBranchName(identifier string) string {
	return fmt.Sprintf("%s/%s", bootstrapBranchPrefix, SanitizeIssueIdentifier(identifier))
}

func DefaultCacheBase(path string) string {
	if trim(path) != "" {
		return absExpand(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Clean("~/.cache/symphony/repos")
	}
	return filepath.Join(home, ".cache", "symphony", "repos")
}

func NormalizeRepoLocator(repoURL string) string {
	raw := trim(repoURL)
	if raw == "" {
		return ""
	}
	if hasSubstring(raw, "://") {
		parsed, err := url.Parse(raw)
		if err == nil {
			return trimSuffix(parsed.Host+parsed.Path, ".git")
		}
	}
	if colon := indexByte(raw, ':'); colon > 0 {
		prefix := raw[:colon]
		if indexByte(prefix, '@') >= 0 {
			host := prefix
			if at := indexByte(prefix, '@'); at >= 0 {
				host = prefix[at+1:]
			}
			return trimSuffix(host+"/"+raw[colon+1:], ".git")
		}
	}
	return trimSuffix(trimSuffix(raw, "/"), ".git")
}

func RepoCacheKey(repoURL string, cacheKey string) string {
	raw := trim(cacheKey)
	if raw == "" {
		raw = NormalizeRepoLocator(repoURL)
	}
	raw = lower(raw)
	sanitized := make([]rune, 0, len(raw))
	lastHyphen := false
	for _, character := range raw {
		valid := (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '.' || character == '_' || character == '-'
		if !valid {
			character = '-'
		}
		if character == '-' {
			if lastHyphen {
				continue
			}
			lastHyphen = true
		} else {
			lastHyphen = false
		}
		sanitized = append(sanitized, character)
	}
	result := trim(string(sanitized))
	result = trimPrefix(result, "-")
	result = trimSuffix(result, "-")
	if result == "" {
		return "repo"
	}
	return result
}

func CacheRootForRepo(repoURL string, cacheBase string, cacheKey string) string {
	return filepath.Join(DefaultCacheBase(cacheBase), RepoCacheKey(repoURL, cacheKey))
}

func resolveCacheRoot(repoURL string, cacheRoot string, cacheBase string, cacheKey string) string {
	if trim(cacheRoot) != "" {
		return absExpand(cacheRoot)
	}
	return CacheRootForRepo(repoURL, cacheBase, cacheKey)
}

func cacheState(repoURL string, repoCacheRoot string, cacheKey string, mirrorCreated bool, seedCreated bool) CacheBootstrapState {
	return CacheBootstrapState{
		CacheRoot:     repoCacheRoot,
		CacheKey:      RepoCacheKey(repoURL, cacheKey),
		MirrorPath:    filepath.Join(repoCacheRoot, "mirror.git"),
		SeedPath:      filepath.Join(repoCacheRoot, "seed"),
		MirrorCreated: mirrorCreated,
		SeedCreated:   seedCreated,
	}
}

func removePath(path string) error {
	return os.RemoveAll(path)
}

func EnsureMirror(repoURL string, cacheRoot string, cacheBase string, cacheKey string) (CacheBootstrapState, error) {
	repoCacheRoot := resolveCacheRoot(repoURL, cacheRoot, cacheBase, cacheKey)
	mirrorPath := filepath.Join(repoCacheRoot, "mirror.git")
	if err := os.MkdirAll(filepath.Dir(mirrorPath), 0o755); err != nil {
		return CacheBootstrapState{}, err
	}
	mirrorCreated := false
	if _, err := os.Stat(filepath.Join(mirrorPath, "HEAD")); err != nil {
		if _, err := os.Stat(mirrorPath); err == nil {
			if err := removePath(mirrorPath); err != nil {
				return CacheBootstrapState{}, err
			}
		}
		result := run([]string{"git", "clone", "--mirror", repoURL, mirrorPath}, ".")
		if result.returnCode != 0 {
			detail := trim(result.stderr)
			if detail == "" {
				detail = trim(result.stdout)
			}
			if detail == "" {
				detail = "git clone --mirror failed"
			}
			return CacheBootstrapState{}, errors.New(detail)
		}
		mirrorCreated = true
	} else {
		if _, err := requireGit(mirrorPath, "remote", "set-url", "origin", repoURL); err != nil {
			return CacheBootstrapState{}, err
		}
		if _, err := requireGit(mirrorPath, "fetch", "--prune", "origin"); err != nil {
			return CacheBootstrapState{}, err
		}
	}
	return cacheState(repoURL, repoCacheRoot, cacheKey, mirrorCreated, false), nil
}

func ConfigureSeedRemotes(seedPath string, repoURL string, mirrorPath string) error {
	remotesOutput, err := requireGit(seedPath, "remote")
	if err != nil {
		return err
	}
	remotes := toSet(splitLines(remotesOutput))
	if _, ok := remotes[cacheRemote]; !ok {
		if _, ok := remotes["origin"]; ok {
			currentOrigin, err := requireGit(seedPath, "remote", "get-url", "origin")
			if err != nil {
				return err
			}
			resolvedOrigin := absExpand(currentOrigin)
			resolvedMirror := absExpand(mirrorPath)
			if resolvedOrigin == resolvedMirror {
				if _, err := requireGit(seedPath, "remote", "rename", "origin", cacheRemote); err != nil {
					return err
				}
				remotesOutput, err = requireGit(seedPath, "remote")
				if err != nil {
					return err
				}
				remotes = toSet(splitLines(remotesOutput))
			}
		}
	}
	if _, ok := remotes[cacheRemote]; !ok {
		if _, err := requireGit(seedPath, "remote", "add", cacheRemote, mirrorPath); err != nil {
			return err
		}
	} else {
		if _, err := requireGit(seedPath, "remote", "set-url", cacheRemote, mirrorPath); err != nil {
			return err
		}
	}
	remotesOutput, err = requireGit(seedPath, "remote")
	if err != nil {
		return err
	}
	remotes = toSet(splitLines(remotesOutput))
	if _, ok := remotes["origin"]; !ok {
		if _, err := requireGit(seedPath, "remote", "add", "origin", repoURL); err != nil {
			return err
		}
	} else {
		if _, err := requireGit(seedPath, "remote", "set-url", "origin", repoURL); err != nil {
			return err
		}
	}
	_, err = requireGit(seedPath, "config", "remote.pushDefault", "origin")
	return err
}

func EnsureSeed(repoURL string, defaultBranch string, cacheRoot string, cacheBase string, cacheKey string) (CacheBootstrapState, error) {
	state, err := EnsureMirror(repoURL, cacheRoot, cacheBase, cacheKey)
	if err != nil {
		return CacheBootstrapState{}, err
	}
	seedPath := state.SeedPath
	seedCreated := false
	if _, err := os.Stat(filepath.Join(seedPath, ".git")); err != nil {
		if _, err := os.Stat(seedPath); err == nil {
			if err := removePath(seedPath); err != nil {
				return CacheBootstrapState{}, err
			}
		}
		result := run([]string{"git", "clone", state.MirrorPath, seedPath}, ".")
		if result.returnCode != 0 {
			detail := trim(result.stderr)
			if detail == "" {
				detail = trim(result.stdout)
			}
			if detail == "" {
				detail = "git clone seed failed"
			}
			return CacheBootstrapState{}, errors.New(detail)
		}
		seedCreated = true
	}
	if err := ConfigureSeedRemotes(seedPath, repoURL, state.MirrorPath); err != nil {
		return CacheBootstrapState{}, err
	}
	for _, args := range [][]string{
		{"fetch", "--prune", cacheRemote},
		{"worktree", "prune"},
		{"checkout", "-B", defaultBranch, fmt.Sprintf("%s/%s", cacheRemote, defaultBranch)},
	} {
		if _, err := requireGit(seedPath, args...); err != nil {
			return CacheBootstrapState{}, err
		}
	}
	return cacheState(repoURL, state.CacheRoot, cacheKey, state.MirrorCreated, seedCreated), nil
}

func BootstrapWorkspace(workspace string, issueIdentifier string, repoURL string, defaultBranch string, cacheRoot string, cacheBase string, cacheKey string) (WorkspaceBootstrapStatus, error) {
	workspacePath := absExpand(workspace)
	state, err := EnsureSeed(repoURL, defaultBranch, cacheRoot, cacheBase, cacheKey)
	if err != nil {
		return WorkspaceBootstrapStatus{}, err
	}
	seedPath := state.SeedPath
	branch := BootstrapBranchName(firstNonEmpty(issueIdentifier, filepath.Base(workspacePath)))
	cacheReused := !state.MirrorCreated && !state.SeedCreated
	cloneSuppressed := !state.MirrorCreated
	if _, err := os.Stat(filepath.Join(workspacePath, ".git")); err == nil {
		currentBranch, err := requireGit(workspacePath, "branch", "--show-current")
		if err != nil {
			return WorkspaceBootstrapStatus{}, err
		}
		return WorkspaceBootstrapStatus{
			Workspace:       workspacePath,
			Branch:          firstNonEmpty(currentBranch, branch),
			CacheRoot:       state.CacheRoot,
			CacheKey:        state.CacheKey,
			MirrorPath:      state.MirrorPath,
			SeedPath:        state.SeedPath,
			Reused:          true,
			CacheReused:     cacheReused,
			CloneSuppressed: cloneSuppressed,
			MirrorCreated:   state.MirrorCreated,
			SeedCreated:     state.SeedCreated,
			WorkspaceMode:   "workspace_reused",
		}, nil
	}
	if err := os.MkdirAll(filepath.Dir(workspacePath), 0o755); err != nil {
		return WorkspaceBootstrapStatus{}, err
	}
	if info, err := os.Stat(workspacePath); err == nil && info.IsDir() {
		entries, err := os.ReadDir(workspacePath)
		if err != nil {
			return WorkspaceBootstrapStatus{}, err
		}
		if len(entries) > 0 {
			return WorkspaceBootstrapStatus{}, fmt.Errorf("workspace is not empty: %s", workspacePath)
		}
	}
	if _, err := requireGit(seedPath, "worktree", "add", "--force", "-B", branch, workspacePath, fmt.Sprintf("%s/%s", cacheRemote, defaultBranch)); err != nil {
		return WorkspaceBootstrapStatus{}, err
	}
	return WorkspaceBootstrapStatus{
		Workspace:       workspacePath,
		Branch:          branch,
		CacheRoot:       state.CacheRoot,
		CacheKey:        state.CacheKey,
		MirrorPath:      state.MirrorPath,
		SeedPath:        state.SeedPath,
		Reused:          false,
		CacheReused:     cacheReused,
		CloneSuppressed: cloneSuppressed,
		MirrorCreated:   state.MirrorCreated,
		SeedCreated:     state.SeedCreated,
		WorkspaceMode:   "worktree_created",
	}, nil
}

func CleanupWorkspace(workspace string, issueIdentifier string, repoURL string, defaultBranch string, cacheRoot string, cacheBase string, cacheKey string) (WorkspaceBootstrapStatus, error) {
	workspacePath := absExpand(workspace)
	repoCacheRoot := resolveCacheRoot(repoURL, cacheRoot, cacheBase, cacheKey)
	state := cacheState(repoURL, repoCacheRoot, cacheKey, false, false)
	seedPath := state.SeedPath
	mirrorPath := state.MirrorPath
	branch := BootstrapBranchName(firstNonEmpty(issueIdentifier, filepath.Base(workspacePath)))
	if _, err := os.Stat(filepath.Join(seedPath, ".git")); err != nil {
		return WorkspaceBootstrapStatus{
			Workspace:       workspacePath,
			Branch:          branch,
			CacheRoot:       state.CacheRoot,
			CacheKey:        state.CacheKey,
			MirrorPath:      state.MirrorPath,
			SeedPath:        state.SeedPath,
			CacheReused:     pathExists(seedPath) || pathExists(mirrorPath),
			CloneSuppressed: true,
			WorkspaceMode:   "cleanup",
			Removed:         false,
		}, nil
	}
	if _, err := os.Stat(workspacePath); err != nil {
		return WorkspaceBootstrapStatus{
			Workspace:       workspacePath,
			Branch:          branch,
			CacheRoot:       state.CacheRoot,
			CacheKey:        state.CacheKey,
			MirrorPath:      state.MirrorPath,
			SeedPath:        state.SeedPath,
			CacheReused:     true,
			CloneSuppressed: true,
			WorkspaceMode:   "cleanup",
			Removed:         false,
		}, nil
	}
	if err := ConfigureSeedRemotes(seedPath, repoURL, mirrorPath); err != nil {
		return WorkspaceBootstrapStatus{}, err
	}
	if git(workspacePath, "rev-parse", "--git-dir").returnCode == 0 {
		currentBranch, err := requireGit(workspacePath, "branch", "--show-current")
		if err == nil && currentBranch != "" {
			branch = currentBranch
		}
	}
	worktreeList, err := requireGit(seedPath, "worktree", "list", "--porcelain")
	if err != nil {
		return WorkspaceBootstrapStatus{}, err
	}
	registered := hasSubstring(worktreeList, "worktree "+workspacePath)
	if !registered {
		if resolved, err := filepath.EvalSymlinks(workspacePath); err == nil {
			registered = hasSubstring(worktreeList, "worktree "+resolved)
		}
	}
	if registered {
		// Detach HEAD in the issue worktree before removal so the bootstrap branch
		// is not still considered checked out when we delete it from the seed repo.
		_, _ = requireGit(workspacePath, "switch", "--detach")
		if _, err := requireGit(seedPath, "worktree", "remove", "--force", workspacePath); err != nil {
			return WorkspaceBootstrapStatus{}, err
		}
		if _, err := requireGit(seedPath, "worktree", "prune"); err != nil {
			return WorkspaceBootstrapStatus{}, err
		}
	}
	localBranchesOutput, err := requireGit(seedPath, "branch", "--format", "%(refname:short)")
	if err != nil {
		return WorkspaceBootstrapStatus{}, err
	}
	localBranches := toSet(splitLines(localBranchesOutput))
	if hasPrefix(branch, bootstrapBranchPrefix+"/") {
		if _, ok := localBranches[branch]; ok {
			if _, err := requireGit(seedPath, "branch", "-D", branch); err != nil {
				return WorkspaceBootstrapStatus{}, err
			}
		}
	}
	if _, err := requireGit(seedPath, "checkout", "-B", defaultBranch, fmt.Sprintf("%s/%s", cacheRemote, defaultBranch)); err != nil {
		return WorkspaceBootstrapStatus{}, err
	}
	return WorkspaceBootstrapStatus{
		Workspace:       workspacePath,
		Branch:          branch,
		CacheRoot:       state.CacheRoot,
		CacheKey:        state.CacheKey,
		MirrorPath:      state.MirrorPath,
		SeedPath:        state.SeedPath,
		CacheReused:     true,
		CloneSuppressed: true,
		WorkspaceMode:   "cleanup",
		Removed:         registered,
	}, nil
}

func BuildValidationReport(repoURL string, workspaceRoot string, issueIdentifiers []string, defaultBranch string, cacheRoot string, cacheBase string, cacheKey string, cleanup bool) (ValidationReport, error) {
	workspaceRootPath := absExpand(workspaceRoot)
	if err := os.MkdirAll(workspaceRootPath, 0o755); err != nil {
		return ValidationReport{}, err
	}
	bootstrapResults := make([]WorkspaceBootstrapStatus, 0, len(issueIdentifiers))
	for _, issueIdentifier := range issueIdentifiers {
		workspacePath := filepath.Join(workspaceRootPath, issueIdentifier)
		status, err := BootstrapWorkspace(workspacePath, issueIdentifier, repoURL, defaultBranch, cacheRoot, cacheBase, cacheKey)
		if err != nil {
			return ValidationReport{}, err
		}
		bootstrapResults = append(bootstrapResults, status)
	}
	cacheRoots := uniqueStrings(func() []string {
		values := make([]string, 0, len(bootstrapResults))
		for _, result := range bootstrapResults {
			values = append(values, result.CacheRoot)
		}
		return values
	}())
	mirrorPaths := uniqueStrings(func() []string {
		values := make([]string, 0, len(bootstrapResults))
		for _, result := range bootstrapResults {
			values = append(values, result.MirrorPath)
		}
		return values
	}())
	seedPaths := uniqueStrings(func() []string {
		values := make([]string, 0, len(bootstrapResults))
		for _, result := range bootstrapResults {
			values = append(values, result.SeedPath)
		}
		return values
	}())
	cleanupResults := make([]WorkspaceBootstrapStatus, 0, len(issueIdentifiers))
	if cleanup {
		for _, issueIdentifier := range issueIdentifiers {
			workspacePath := filepath.Join(workspaceRootPath, issueIdentifier)
			status, err := CleanupWorkspace(workspacePath, issueIdentifier, repoURL, defaultBranch, cacheRoot, cacheBase, cacheKey)
			if err != nil {
				return ValidationReport{}, err
			}
			cleanupResults = append(cleanupResults, status)
		}
	}
	summary := ValidationSummary{
		WorkspaceCount:        len(bootstrapResults),
		UniqueCacheRoots:      cacheRoots,
		UniqueMirrorPaths:     mirrorPaths,
		UniqueSeedPaths:       seedPaths,
		SingleCacheRootReused: len(cacheRoots) == 1,
		SingleMirrorReused:    len(mirrorPaths) == 1,
		SingleSeedReused:      len(seedPaths) == 1,
	}
	for _, result := range bootstrapResults {
		if result.MirrorCreated {
			summary.MirrorCreations++
		}
		if result.SeedCreated {
			summary.SeedCreations++
		}
	}
	summary.CloneSuppressedAfterFirst = true
	summary.CacheReusedAfterFirst = true
	summary.AllWorkspacesCreatedWorktree = true
	for idx, result := range bootstrapResults {
		if idx > 0 && !result.CloneSuppressed {
			summary.CloneSuppressedAfterFirst = false
		}
		if idx > 0 && !result.CacheReused {
			summary.CacheReusedAfterFirst = false
		}
		if result.WorkspaceMode != "worktree_created" && result.WorkspaceMode != "workspace_reused" {
			summary.AllWorkspacesCreatedWorktree = false
		}
	}
	if len(bootstrapResults) > 0 {
		summary.CleanupPreservedCache = pathExists(bootstrapResults[0].MirrorPath) && pathExists(filepath.Join(bootstrapResults[0].SeedPath, ".git"))
	}
	return ValidationReport{
		RepoURL:          repoURL,
		DefaultBranch:    defaultBranch,
		WorkspaceRoot:    workspaceRootPath,
		IssueIdentifiers: append([]string{}, issueIdentifiers...),
		BootstrapResults: bootstrapResults,
		CleanupResults:   cleanupResults,
		Summary:          summary,
	}, nil
}

func RenderValidationMarkdown(report ValidationReport) string {
	lines := []string{
		"# Symphony bootstrap cache validation",
		"",
		fmt.Sprintf("- Repo: `%s`", report.RepoURL),
		fmt.Sprintf("- Workspace root: `%s`", report.WorkspaceRoot),
		fmt.Sprintf("- Workspaces: `%d`", report.Summary.WorkspaceCount),
		fmt.Sprintf("- Single cache root reused: `%t`", report.Summary.SingleCacheRootReused),
		fmt.Sprintf("- Mirror creations: `%d`", report.Summary.MirrorCreations),
		fmt.Sprintf("- Seed creations: `%d`", report.Summary.SeedCreations),
		fmt.Sprintf("- Clone suppressed after first workspace: `%t`", report.Summary.CloneSuppressedAfterFirst),
		fmt.Sprintf("- Cleanup preserved cache: `%t`", report.Summary.CleanupPreservedCache),
		"",
		"## Bootstrap Results",
		"",
	}
	for _, result := range report.BootstrapResults {
		lines = append(lines,
			fmt.Sprintf("- `%s`", result.Workspace),
			fmt.Sprintf("  - `cache_root=%s`", result.CacheRoot),
			fmt.Sprintf("  - `cache_key=%s`", result.CacheKey),
			fmt.Sprintf("  - `workspace_mode=%s`", result.WorkspaceMode),
			fmt.Sprintf("  - `cache_reused=%t`", result.CacheReused),
			fmt.Sprintf("  - `clone_suppressed=%t`", result.CloneSuppressed),
			fmt.Sprintf("  - `mirror_created=%t`", result.MirrorCreated),
			fmt.Sprintf("  - `seed_created=%t`", result.SeedCreated),
		)
	}
	return joinLines(lines) + "\n"
}

func WriteValidationReport(report ValidationReport, path string) (string, error) {
	target := absExpand(path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if hasSuffix(lower(target), ".md") {
		if err := os.WriteFile(target, []byte(RenderValidationMarkdown(report)), 0o644); err != nil {
			return "", err
		}
		return target, nil
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, body, 0o644); err != nil {
		return "", err
	}
	return target, nil
}

func absExpand(path string) string {
	value := trim(path)
	if value == "" {
		value = "."
	}
	if hasPrefix(value, "~/") || value == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if value == "~" {
				value = home
			} else {
				value = filepath.Join(home, value[2:])
			}
		}
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return filepath.Clean(value)
	}
	return absolute
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func toSet(values []string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, value := range values {
		value = trim(value)
		if value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func splitLines(value string) []string {
	lines := []string{}
	current := ""
	for _, r := range value {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
			continue
		}
		if r != '\r' {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func join(items []string) string {
	if len(items) == 0 {
		return ""
	}
	value := items[0]
	for _, item := range items[1:] {
		value += " " + item
	}
	return value
}

func joinLines(items []string) string {
	if len(items) == 0 {
		return ""
	}
	value := items[0]
	for _, item := range items[1:] {
		value += "\n" + item
	}
	return value
}

func trim(value string) string {
	start := 0
	end := len(value)
	for start < end && (value[start] == ' ' || value[start] == '\n' || value[start] == '\r' || value[start] == '\t') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\n' || value[end-1] == '\r' || value[end-1] == '\t') {
		end--
	}
	return value[start:end]
}

func trimSuffix(value string, suffix string) string {
	if hasSuffix(value, suffix) {
		return value[:len(value)-len(suffix)]
	}
	return value
}

func trimPrefix(value string, prefix string) string {
	if hasPrefix(value, prefix) {
		return value[len(prefix):]
	}
	return value
}

func hasPrefix(value string, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

func hasSuffix(value string, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}

func hasSubstring(value string, part string) bool {
	if part == "" {
		return true
	}
	for idx := 0; idx+len(part) <= len(value); idx++ {
		if value[idx:idx+len(part)] == part {
			return true
		}
	}
	return false
}

func indexByte(value string, target byte) int {
	for idx := 0; idx < len(value); idx++ {
		if value[idx] == target {
			return idx
		}
	}
	return -1
}

func lower(value string) string {
	result := make([]byte, 0, len(value))
	for idx := 0; idx < len(value); idx++ {
		character := value[idx]
		if character >= 'A' && character <= 'Z' {
			character = character + ('a' - 'A')
		}
		result = append(result, character)
	}
	return string(result)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trim(value) != "" {
			return value
		}
	}
	return ""
}
