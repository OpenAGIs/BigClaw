package githubsync

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	configLockRetryCount = 6
	configLockRetryDelay = 50 * time.Millisecond
)

type RepoSyncStatus struct {
	Branch       string `json:"branch"`
	LocalSHA     string `json:"local_sha"`
	RemoteSHA    string `json:"remote_sha"`
	Dirty        bool   `json:"dirty"`
	RemoteExists bool   `json:"remote_exists"`
	Synced       bool   `json:"synced"`
	Pushed       bool   `json:"pushed"`
}

type commandResult struct {
	stdout     string
	stderr     string
	returnCode int
}

func run(command []string, repo string) commandResult {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = repo
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
		detail := trimmed(result.stderr)
		if detail == "" {
			detail = trimmed(result.stdout)
		}
		if detail == "" {
			detail = fmt.Sprintf("git %s failed", joinArgs(args))
		}
		return "", errors.New(detail)
	}
	return trimmed(result.stdout), nil
}

func requireGitWithConfigLockRetry(repo string, args ...string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < configLockRetryCount; attempt++ {
		output, err := requireGit(repo, args...)
		if err == nil {
			return output, nil
		}
		lastErr = err
		if !isConfigLockError(err) || attempt == configLockRetryCount-1 {
			break
		}
		time.Sleep(time.Duration(attempt+1) * configLockRetryDelay)
	}
	return "", lastErr
}

func trimmed(value string) string {
	return stringTrimSpace(value)
}

func stringTrimSpace(value string) string {
	start := 0
	end := len(value)
	for start < end && (value[start] == ' ' || value[start] == '\n' || value[start] == '\t' || value[start] == '\r') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\n' || value[end-1] == '\t' || value[end-1] == '\r') {
		end--
	}
	return value[start:end]
}

func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	result := args[0]
	for _, item := range args[1:] {
		result += " " + item
	}
	return result
}

func dirty(repo string) (bool, error) {
	status, err := requireGit(repo, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return status != "", nil
}

func remoteDefaultBranch(repo string, remote string) (string, error) {
	symbolicRef := git(repo, "symbolic-ref", "--quiet", fmt.Sprintf("refs/remotes/%s/HEAD", remote))
	if symbolicRef.returnCode == 0 {
		value := trimmed(symbolicRef.stdout)
		prefix := fmt.Sprintf("refs/remotes/%s/", remote)
		if len(value) > len(prefix) && value[:len(prefix)] == prefix {
			return value[len(prefix):], nil
		}
	}

	symref := git(repo, "ls-remote", "--symref", remote, "HEAD")
	if symref.returnCode != 0 {
		detail := trimmed(symref.stderr)
		if detail == "" {
			detail = trimmed(symref.stdout)
		}
		if detail == "" {
			detail = fmt.Sprintf("git ls-remote --symref failed for %s/HEAD", remote)
		}
		return "", errors.New(detail)
	}

	for _, line := range splitLines(symref.stdout) {
		if hasPrefix(line, "ref: ") && hasSuffix(line, "\tHEAD") {
			parts := splitWhitespace(line)
			if len(parts) < 2 {
				continue
			}
			ref := parts[1]
			const prefix = "refs/heads/"
			if hasPrefix(ref, prefix) {
				return ref[len(prefix):], nil
			}
		}
	}
	return "", fmt.Errorf("could not determine default branch for remote %s", remote)
}

func remoteBranchSHA(repo string, remote string, branch string) (string, error) {
	localRef := git(repo, "rev-parse", fmt.Sprintf("refs/remotes/%s/%s", remote, branch))
	if localRef.returnCode == 0 && trimmed(localRef.stdout) != "" {
		return trimmed(localRef.stdout), nil
	}

	remoteResult := git(repo, "ls-remote", "--heads", remote, branch)
	if remoteResult.returnCode != 0 {
		detail := trimmed(remoteResult.stderr)
		if detail == "" {
			detail = trimmed(remoteResult.stdout)
		}
		if detail == "" {
			detail = fmt.Sprintf("git ls-remote failed for %s/%s", remote, branch)
		}
		return "", errors.New(detail)
	}
	fields := splitWhitespace(remoteResult.stdout)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], nil
}

func matchesRemoteDefaultBranch(repo string, remote string, localSHA string) bool {
	defaultBranch, err := remoteDefaultBranch(repo, remote)
	if err != nil {
		return false
	}
	defaultSHA, err := remoteBranchSHA(repo, remote, defaultBranch)
	if err != nil {
		return false
	}
	return defaultSHA != "" && defaultSHA == localSHA
}

func InspectRepoSync(repo string, remote string) (RepoSyncStatus, error) {
	repoPath, err := filepath.Abs(repo)
	if err != nil {
		return RepoSyncStatus{}, err
	}
	branch, err := requireGit(repoPath, "branch", "--show-current")
	if err != nil {
		return RepoSyncStatus{}, err
	}
	if branch == "" {
		return RepoSyncStatus{}, errors.New("detached HEAD does not support issue branch sync automation")
	}
	localSHA, err := requireGit(repoPath, "rev-parse", "HEAD")
	if err != nil {
		return RepoSyncStatus{}, err
	}
	remoteResult := git(repoPath, "ls-remote", "--heads", remote, branch)
	if remoteResult.returnCode != 0 {
		detail := trimmed(remoteResult.stderr)
		if detail == "" {
			detail = trimmed(remoteResult.stdout)
		}
		if detail == "" {
			detail = fmt.Sprintf("git ls-remote failed for %s/%s", remote, branch)
		}
		return RepoSyncStatus{}, errors.New(detail)
	}
	fields := splitWhitespace(remoteResult.stdout)
	remoteSHA := ""
	if len(fields) > 0 {
		remoteSHA = fields[0]
	}
	isDirty, err := dirty(repoPath)
	if err != nil {
		return RepoSyncStatus{}, err
	}
	remoteExists := remoteSHA != ""
	synced := remoteExists && localSHA == remoteSHA
	if !remoteExists && matchesRemoteDefaultBranch(repoPath, remote, localSHA) {
		synced = true
	}
	// "pushed" should be stable: it means the remote branch exists and contains HEAD.
	// A synced default-branch fallback (remote branch absent but SHA matches default) is not considered pushed.
	pushed := remoteExists && localSHA == remoteSHA
	return RepoSyncStatus{
		Branch:       branch,
		LocalSHA:     localSHA,
		RemoteSHA:    remoteSHA,
		Dirty:        isDirty,
		RemoteExists: remoteExists,
		Synced:       synced,
		Pushed:       pushed,
	}, nil
}

func InstallGitHooks(repo string, hooksPath string) (string, error) {
	repoPath, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	hooksDir := filepath.Join(repoPath, hooksPath)
	info, err := os.Stat(hooksDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("missing hooks directory: %s", hooksDir)
	}

	currentHooksPath, configured, err := gitConfigValue(repoPath, "core.hooksPath")
	if err != nil {
		return "", err
	}
	if !configured || currentHooksPath != hooksPath {
		if _, err := requireGitWithConfigLockRetry(repoPath, "config", "core.hooksPath", hooksPath); err != nil {
			return "", err
		}
	}

	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(hooksDir, entry.Name())
		stat, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		if err := os.Chmod(path, stat.Mode()|0o111); err != nil {
			return "", err
		}
	}
	return hooksDir, nil
}

func gitConfigValue(repo string, key string) (string, bool, error) {
	result := git(repo, "config", "--get", key)
	if result.returnCode == 0 {
		return trimmed(result.stdout), true, nil
	}
	if result.returnCode == 1 {
		return "", false, nil
	}
	detail := trimmed(result.stderr)
	if detail == "" {
		detail = trimmed(result.stdout)
	}
	if detail == "" {
		detail = fmt.Sprintf("git config --get %s failed", key)
	}
	return "", false, errors.New(detail)
}

func isConfigLockError(err error) bool {
	if err == nil {
		return false
	}
	return hasSubstring(err.Error(), "could not lock config file")
}

func hasSubstring(value string, needle string) bool {
	if needle == "" {
		return true
	}
	if len(needle) > len(value) {
		return false
	}
	lastStart := len(value) - len(needle)
	for start := 0; start <= lastStart; start++ {
		if value[start:start+len(needle)] == needle {
			return true
		}
	}
	return false
}

func EnsureRepoSync(repo string, remote string, autoPush bool, allowDirty bool) (RepoSyncStatus, error) {
	repoPath, err := filepath.Abs(repo)
	if err != nil {
		return RepoSyncStatus{}, err
	}
	status, err := InspectRepoSync(repoPath, remote)
	if err != nil {
		return RepoSyncStatus{}, err
	}
	if status.Dirty && !allowDirty {
		return RepoSyncStatus{}, errors.New("working tree is dirty; commit or stash changes before syncing")
	}
	if status.RemoteExists && !status.Synced {
		fetchResult := git(repoPath, "fetch", remote, status.Branch)
		if fetchResult.returnCode != 0 {
			detail := trimmed(fetchResult.stderr)
			if detail == "" {
				detail = trimmed(fetchResult.stdout)
			}
			if detail == "" {
				detail = fmt.Sprintf("git fetch %s %s failed", remote, status.Branch)
			}
			return RepoSyncStatus{}, errors.New(detail)
		}
		relation, err := branchRelation(repoPath, remote, status.Branch)
		if err != nil {
			return RepoSyncStatus{}, err
		}
		if relation.behind > 0 {
			if status.Dirty {
				return RepoSyncStatus{}, fmt.Errorf("remote branch moved while working tree is dirty: branch=%s ahead=%d behind=%d", status.Branch, relation.ahead, relation.behind)
			}
			pullResult := git(repoPath, "pull", "--ff-only", remote, status.Branch)
			if pullResult.returnCode != 0 {
				detail := trimmed(pullResult.stderr)
				if detail == "" {
					detail = trimmed(pullResult.stdout)
				}
				if detail == "" {
					detail = fmt.Sprintf("git pull --ff-only %s %s failed", remote, status.Branch)
				}
				return RepoSyncStatus{}, errors.New(detail)
			}
		}
		status, err = InspectRepoSync(repoPath, remote)
		if err != nil {
			return RepoSyncStatus{}, err
		}
	}
	if !autoPush || status.Synced {
		return status, nil
	}
	pushArgs := []string{"push", remote, "HEAD"}
	if !status.RemoteExists {
		pushArgs = []string{"push", "-u", remote, "HEAD"}
	}
	pushResult := git(repoPath, pushArgs...)
	if pushResult.returnCode != 0 {
		detail := trimmed(pushResult.stderr)
		if detail == "" {
			detail = trimmed(pushResult.stdout)
		}
		if detail == "" {
			detail = fmt.Sprintf("git %s failed", joinArgs(pushArgs))
		}
		return RepoSyncStatus{}, errors.New(detail)
	}
	refreshed, err := InspectRepoSync(repoPath, remote)
	if err != nil {
		return RepoSyncStatus{}, err
	}
	refreshed.Pushed = true
	if !refreshed.Synced {
		return RepoSyncStatus{}, fmt.Errorf(
			"remote SHA mismatch after push: local=%s remote=%s",
			refreshed.LocalSHA,
			valueOr(refreshed.RemoteSHA, "missing"),
		)
	}
	return refreshed, nil
}

func valueOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

type syncRelation struct {
	ahead  int
	behind int
}

func branchRelation(repo string, remote string, branch string) (syncRelation, error) {
	output, err := requireGit(repo, "rev-list", "--left-right", "--count", "HEAD...refs/remotes/"+remote+"/"+branch)
	if err != nil {
		return syncRelation{}, err
	}
	fields := splitWhitespace(output)
	if len(fields) != 2 {
		return syncRelation{}, fmt.Errorf("unexpected rev-list output: %q", output)
	}
	ahead, err := parseInt(fields[0])
	if err != nil {
		return syncRelation{}, err
	}
	behind, err := parseInt(fields[1])
	if err != nil {
		return syncRelation{}, err
	}
	return syncRelation{ahead: ahead, behind: behind}, nil
}

func parseInt(value string) (int, error) {
	result := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer: %q", value)
		}
		result = result*10 + int(r-'0')
	}
	return result, nil
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

func splitWhitespace(value string) []string {
	result := []string{}
	current := ""
	for _, r := range value {
		switch r {
		case ' ', '\n', '\r', '\t':
			if current != "" {
				result = append(result, current)
				current = ""
			}
		default:
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func hasPrefix(value string, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

func hasSuffix(value string, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}
