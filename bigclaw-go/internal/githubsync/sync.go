package githubsync

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	return RepoSyncStatus{
		Branch:       branch,
		LocalSHA:     localSHA,
		RemoteSHA:    remoteSHA,
		Dirty:        isDirty,
		RemoteExists: remoteExists,
		Synced:       synced,
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
	if _, err := requireGit(repoPath, "config", "core.hooksPath", hooksPath); err != nil {
		return "", err
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

func EnsureRepoSync(repo string, remote string, autoPush bool, allowDirty bool) (RepoSyncStatus, error) {
	repoPath, err := filepath.Abs(repo)
	if err != nil {
		return RepoSyncStatus{}, err
	}
	status, err := InspectRepoSync(repoPath, remote)
	if err != nil {
		return RepoSyncStatus{}, err
	}
	if status.Dirty {
		if allowDirty {
			return status, nil
		}
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
		status, err = InspectRepoSync(repoPath, remote)
		if err != nil {
			return RepoSyncStatus{}, err
		}
	}
	if !autoPush {
		return status, nil
	}
	if status.Synced && status.RemoteExists {
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
