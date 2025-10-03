package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// emptyTreeHash is the SHA-1 hash of the empty tree object in Git
const emptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// GitError represents a git command error with captured output
type GitError struct {
	Command string
	Err     error
	Stdout  string
	Stderr  string
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git command '%s' failed: %v\nstdout: %s\nstderr: %s", e.Command, e.Err, e.Stdout, e.Stderr)
}

// Commit represents a git commit with detailed information
type Commit struct {
	Hash      string
	Subject   string
	Author    string
	Email     string
	DateTime  string
	IsMerge   bool
	MergeFrom string // For merge commits, this contains the hash of the merged commit
}

// CheckGitAvailability verifies that git command is available and working
func CheckGitAvailability() error {
	// Check if git command exists
	cmd := exec.Command("git", "--version")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command not found or not working: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Verify git version output looks reasonable
	versionOutput := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(versionOutput, "git version") {
		return fmt.Errorf("unexpected git version output: %s", versionOutput)
	}

	return nil
}

// runGitCommand executes a git command in a specific directory
func runGitCommand(dir string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no git command arguments provided")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		return "", &GitError{
			Command: fmt.Sprintf("git %s (in %s)", strings.Join(args, " "), dir),
			Err:     err,
			Stdout:  stdout.String(),
			Stderr:  stderr.String(),
		}
	}

	return stdout.String(), nil
}

// parseCommitsWithMergeInfo parses git log output with merge information and returns a slice of Commit structs
func parseCommitsWithMergeInfo(output string) []Commit {
	if len(output) == 0 {
		return []Commit{}
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []Commit{}
	}

	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		// Parse commit format: hash|subject|author|email|datetime|parents
		parts := strings.Split(line, "|")
		if len(parts) >= 6 {
			parents := parts[5]
			parentHashes := strings.Fields(parents)

			commit := Commit{
				Hash:      parts[0],
				Subject:   parts[1],
				Author:    parts[2],
				Email:     parts[3],
				DateTime:  parts[4],
				IsMerge:   len(parentHashes) > 1,
				MergeFrom: "",
			}

			// For merge commits, the second parent is typically the merged branch
			if commit.IsMerge && len(parentHashes) >= 2 {
				commit.MergeFrom = parentHashes[1]
			}

			commits = append(commits, commit)
		}
	}

	return commits
}

// getCommitsFirstParentWithMerges executes git log constrained to the branch's first-parent history,
// including merge commits. This returns commits made on the current branch including merge commits.
func getCommitsFirstParentWithMerges(repoPath string, commitRange string) ([]Commit, error) {
	var args []string
	if commitRange == "" {
		args = []string{"log", "--first-parent", "--pretty=format:%h|%s|%an|%ae|%ad|%P", "--date=iso"}
	} else {
		args = []string{"log", "--first-parent", "--pretty=format:%h|%s|%an|%ae|%ad|%P", "--date=iso", commitRange}
	}

	output, err := runGitCommand(repoPath, args...)
	if err != nil {
		return nil, err
	}

	// Use the new parsing function that includes merge information
	return parseCommitsWithMergeInfo(output), nil
}

// GetUnpushedCommits finds unpushed commits in a repository
func GetUnpushedCommits(repoPath string, parentGitBranchName string) ([]Commit, error) {
	// Get the current branch
	branchOutput, err := runGitCommand(repoPath, "branch", "--show-current")
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	currentBranch := strings.TrimSpace(branchOutput)

	if currentBranch == "" {
		// Probably in detached HEAD state or no commits yet
		return []Commit{}, nil
	}

	// First check if there are any commits at all
	if _, err := runGitCommand(repoPath, "rev-parse", "HEAD"); err != nil {
		// No commits in the repository
		return []Commit{}, nil
	}

	// Check if the current branch has an upstream tracking branch
	upstreamOutput, err := runGitCommand(repoPath, "rev-parse", "--abbrev-ref", fmt.Sprintf("%s@{upstream}", currentBranch))

	if err != nil {
		// No upstream branch configured, check if there are any remotes
		remotesOutput, remotesErr := runGitCommand(repoPath, "remote")

		if remotesErr != nil || strings.TrimSpace(remotesOutput) == "" {
			// No remotes configured; return commits made on this branch's first-parent history including merges
			commits, err := getCommitsFirstParentWithMerges(repoPath, "")
			if err != nil {
				return []Commit{}, nil
			}
			return commits, nil
		}

		// There are remotes but no upstream branch, try different strategies to find unpushed commits

		// Strategy 1: Check against origin/<branch> if it exists
		if _, originErr := runGitCommand(repoPath, "rev-parse", "--verify", fmt.Sprintf("origin/%s", currentBranch)); originErr == nil {
			// origin/<branch> exists, compare against it, including merge commits
			commits, err := getCommitsFirstParentWithMerges(repoPath, fmt.Sprintf("origin/%s..%s", currentBranch, currentBranch))
			if err != nil {
				return nil, fmt.Errorf("failed to get unpushed commits: %w", err)
			}
			return commits, nil
		}

		// Strategy 2: Check against any remote branches that match current branch name
		remotesList := strings.Fields(strings.TrimSpace(remotesOutput))
		for _, remote := range remotesList {
			if _, remoteBranchErr := runGitCommand(repoPath, "rev-parse", "--verify", fmt.Sprintf("%s/%s", remote, currentBranch)); remoteBranchErr == nil {
				// Found matching remote branch, compare against it including merge commits
				commits, err := getCommitsFirstParentWithMerges(repoPath, fmt.Sprintf("%s/%s..%s", remote, currentBranch, currentBranch))
				if err != nil {
					return nil, fmt.Errorf("failed to get unpushed commits: %w", err)
				}
				return commits, nil
			}
		}

		// Strategy 3: Find the actual parent/base branch dynamically
		commits, err := getCommitsFirstParentWithMerges(repoPath, fmt.Sprintf("%s..%s", parentGitBranchName, currentBranch))
		if err == nil {
			return commits, nil
		}

		// Strategy 4: If all else fails, assume all commits are unpushed (fallback)
		commits, err = getCommitsFirstParentWithMerges(repoPath, "")
		if err != nil {
			return []Commit{}, nil
		}
		return commits, nil
	}

	// Upstream branch exists, compare against it
	upstream := strings.TrimSpace(upstreamOutput)
	commits, err := getCommitsFirstParentWithMerges(repoPath, fmt.Sprintf("%s..%s", upstream, currentBranch))
	if err != nil {
		return nil, fmt.Errorf("failed to get unpushed commits: %w", err)
	}

	return commits, nil
}

// GetParentCommit finds the parent commit of the first unpushed commit
func GetParentCommit(repoPath string, firstUnpushedCommitHash string) (string, error) {
	// Get parent commit hash using git rev-parse
	parentOutput, err := runGitCommand(repoPath, "rev-parse", fmt.Sprintf("%s^", firstUnpushedCommitHash))
	if err != nil {
		return "", fmt.Errorf("failed to get parent commit: %w", err)
	}

	parentHash := strings.TrimSpace(parentOutput)
	return parentHash, nil
}

// GetLastPushedCommit gets the last pushed commit for a repository
func GetLastPushedCommit(repoPath string, parentGitBranchName string) (*Commit, error) {
	// Get the current branch
	branchOutput, err := runGitCommand(repoPath, "branch", "--show-current")
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	currentBranch := strings.TrimSpace(branchOutput)

	if currentBranch == "" {
		// Probably in detached HEAD state or no commits yet
		return nil, nil
	}

	// First check if there are any commits at all
	if _, err := runGitCommand(repoPath, "rev-parse", "HEAD"); err != nil {
		// No commits in the repository
		return nil, nil
	}

	// Check if the current branch has an upstream tracking branch
	upstreamOutput, err := runGitCommand(repoPath, "rev-parse", "--abbrev-ref", fmt.Sprintf("%s@{upstream}", currentBranch))

	if err != nil {
		// No upstream branch configured, check if there are any remotes
		remotesOutput, remotesErr := runGitCommand(repoPath, "remote")

		if remotesErr != nil || strings.TrimSpace(remotesOutput) == "" {
			// No remotes configured; no pushed commits
			return nil, nil
		}

		// There are remotes but no upstream branch, try different strategies to find last pushed commit

		// Strategy 1: Check against origin/<branch> if it exists
		if _, originErr := runGitCommand(repoPath, "rev-parse", "--verify", fmt.Sprintf("origin/%s", currentBranch)); originErr == nil {
			// origin/<branch> exists, get the last commit on it
			output, err := runGitCommand(repoPath, "log", "-1", "--pretty=format:%H|%s|%an|%ae|%ad|%P", "--date=format:%Y-%m-%d %H:%M:%S %z", fmt.Sprintf("origin/%s", currentBranch))
			if err != nil {
				return nil, nil
			}
			commits := parseCommitsWithMergeInfo(output)
			if len(commits) > 0 {
				return &commits[0], nil
			}
			return nil, nil
		}

		// Strategy 2: Check against any remote branches that match current branch name
		remotesList := strings.Fields(strings.TrimSpace(remotesOutput))
		for _, remote := range remotesList {
			if _, remoteBranchErr := runGitCommand(repoPath, "rev-parse", "--verify", fmt.Sprintf("%s/%s", remote, currentBranch)); remoteBranchErr == nil {
				// Found matching remote branch, get the last commit on it
				output, err := runGitCommand(repoPath, "log", "-1", "--pretty=format:%H|%s|%an|%ae|%ad|%P", "--date=format:%Y-%m-%d %H:%M:%S %z", fmt.Sprintf("%s/%s", remote, currentBranch))
				if err != nil {
					continue
				}
				commits := parseCommitsWithMergeInfo(output)
				if len(commits) > 0 {
					return &commits[0], nil
				}
			}
		}

		// Strategy 3: Try against parent branch
		output, err := runGitCommand(repoPath, "log", "-1", "--pretty=format:%H|%s|%an|%ae|%ad|%P", "--date=format:%Y-%m-%d %H:%M:%S %z", parentGitBranchName)
		if err == nil {
			commits := parseCommitsWithMergeInfo(output)
			if len(commits) > 0 {
				return &commits[0], nil
			}
		}

		// No pushed commits found
		return nil, nil
	}

	// Upstream branch exists, get the last commit on it
	upstream := strings.TrimSpace(upstreamOutput)
	output, err := runGitCommand(repoPath, "log", "-1", "--pretty=format:%H|%s|%an|%ae|%ad|%P", "--date=format:%Y-%m-%d %H:%M:%S %z", upstream)
	if err != nil {
		return nil, fmt.Errorf("failed to get last pushed commit: %w", err)
	}

	commits := parseCommitsWithMergeInfo(output)
	if len(commits) > 0 {
		return &commits[0], nil
	}

	return nil, nil
}

// GetCurrentBranch gets the current branch name for the repository
func GetCurrentBranch(repoPath string) (string, error) {
	// Get the current branch
	branchOutput, err := runGitCommand(repoPath, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	currentBranch := strings.TrimSpace(branchOutput)
	if currentBranch == "" {
		return "", fmt.Errorf("repository is in detached HEAD state or has no commits")
	}

	return currentBranch, nil
}

// GetCommitMessage gets the full commit message for a given commit hash
func GetCommitMessage(repoPath string, commitHash string) (string, error) {
	output, err := runGitCommand(repoPath, "log", "--format=%B", "-n", "1", commitHash)
	if err != nil {
		return "", fmt.Errorf("failed to get commit message for %s: %w", commitHash, err)
	}
	return output, nil
}

// extractBranchNameFromMergeMessage extracts the branch name from a merge commit message
// Handles formats like "Merge branch 'feature-branch' into main" or "Merge commit abc123 into main"
func extractBranchNameFromMergeMessage(message string) string {
	lines := strings.Split(strings.TrimSpace(message), "\n")
	if len(lines) == 0 {
		return ""
	}

	firstLine := lines[0]

	// Look for patterns like "Merge branch 'branch-name' into target"
	if strings.Contains(firstLine, "Merge branch '") {
		start := strings.Index(firstLine, "Merge branch '") + len("Merge branch '")
		end := strings.Index(firstLine[start:], "'")
		if end != -1 {
			return firstLine[start : start+end]
		}
	}

	// Look for patterns like "Merge branch branch-name into target" (without quotes)
	if strings.Contains(firstLine, "Merge branch ") {
		parts := strings.Fields(firstLine)
		for i, part := range parts {
			if part == "branch" && i+1 < len(parts) {
				nextPart := parts[i+1]
				// Remove "into" if it's the next word
				if nextPart == "into" {
					continue
				}
				return nextPart
			}
		}
	}

	// If we can't extract a branch name, return empty string
	return ""
}

// UpdateCommitTimes updates the commit times by processing all commits in a single git filter-repo run
func UpdateCommitTimes(repoPath string, commits []Commit, newTimes []time.Time, parentCommitHash string, branchName string, rewriteBranchName string, newCommitAuthorName string, newCommitAuthorEmail string) (int, error) {
	// Checkout the parent commit (skip if it's the empty tree hash)
	if parentCommitHash != emptyTreeHash {
		if _, err := runGitCommand(repoPath, "checkout", parentCommitHash); err != nil {
			return 0, fmt.Errorf("failed to checkout parent commit %s: %w", parentCommitHash, err)
		}
	}

	// Create and checkout the rewrite branch
	if _, err := runGitCommand(repoPath, "checkout", "-b", rewriteBranchName); err != nil {
		return 0, fmt.Errorf("failed to create rewrite branch %s: %w", rewriteBranchName, err)
	}

	successfulUpdates := 0

	// Process each commit and update its metadata (commits are already in correct order)
	for i, commit := range commits {
		newTime := newTimes[i]

		if commit.IsMerge {
			// Handle merge commits by merging the original merged commit
			if commit.MergeFrom == "" {
				return successfulUpdates, fmt.Errorf("merge commit %s has no merge source", commit.Hash)
			}

			// Get the original merge commit message to extract branch information
			originalMessage, err := GetCommitMessage(repoPath, commit.Hash)
			if err != nil {
				return successfulUpdates, fmt.Errorf("failed to get original merge commit message for %s: %w", commit.Hash, err)
			}

			// Extract the original branch name from the merge message
			originalBranchName := extractBranchNameFromMergeMessage(originalMessage)
			if originalBranchName == "" {
				// Fallback: use the commit hash if we can't extract branch name
				originalBranchName = commit.MergeFrom[:8] // Use short hash
			}

			// Create a custom merge message with proper branch names
			customMergeMessage := fmt.Sprintf("Merge branch '%s' into %s", originalBranchName, branchName)

			// Merge the commit that was originally merged with custom message
			if _, err := runGitCommand(repoPath, "merge", "-m", customMergeMessage, commit.MergeFrom); err != nil {
				return successfulUpdates, fmt.Errorf("failed to merge commit %s: %w", commit.MergeFrom, err)
			}

			// For merge commits, use the provided newTime (which should be same or later than original)
			// This ensures merge commits maintain chronological order with the rewrite branch
		} else {
			// Handle regular commits by cherry-picking
			// Try cherry-pick first
			_, err := runGitCommand(repoPath, "cherry-pick", commit.Hash)
			if err != nil {
				// Check if we're in a cherry-pick state by looking at git status
				status, statusErr := runGitCommand(repoPath, "status")
				if statusErr == nil && strings.Contains(status, "cherry-picking") {
					// We're in a cherry-pick state, try to continue
					_, continueErr := runGitCommand(repoPath, "cherry-pick", "--continue")
					if continueErr != nil {
						// If continue fails, try to skip the commit
						_, skipErr := runGitCommand(repoPath, "cherry-pick", "--skip")
						if skipErr != nil {
							// If skip also fails, abort and try with --allow-empty
							runGitCommand(repoPath, "cherry-pick", "--abort")
							if _, allowEmptyErr := runGitCommand(repoPath, "cherry-pick", "--allow-empty", commit.Hash); allowEmptyErr != nil {
								return successfulUpdates, fmt.Errorf("failed to cherry-pick commit %s: %w", commit.Hash, err)
							}
						}
					}
				} else {
					// Not in cherry-pick state, try with --allow-empty
					if _, allowEmptyErr := runGitCommand(repoPath, "cherry-pick", "--allow-empty", commit.Hash); allowEmptyErr != nil {
						return successfulUpdates, fmt.Errorf("failed to cherry-pick commit %s: %w", commit.Hash, err)
					}
				}
			}
		}

		// Format the time for git environment variables
		newTimeStr := newTime.Format("2006-01-02T15:04:05")

		// Update commit metadata using git commit --amend with environment variables
		cmd := exec.Command("git", "commit", "--amend", "--no-edit", "--reset-author")
		cmd.Dir = repoPath

		// Build environment variables
		env := os.Environ()
		env = append(env, fmt.Sprintf("GIT_AUTHOR_DATE=%s", newTimeStr))
		env = append(env, fmt.Sprintf("GIT_COMMITTER_DATE=%s", newTimeStr))

		// Only set author name and email if they're provided
		if newCommitAuthorName != "" {
			env = append(env, fmt.Sprintf("GIT_AUTHOR_NAME=%s", newCommitAuthorName))
			env = append(env, fmt.Sprintf("GIT_COMMITTER_NAME=%s", newCommitAuthorName))
		}
		if newCommitAuthorEmail != "" {
			env = append(env, fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", newCommitAuthorEmail))
			env = append(env, fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", newCommitAuthorEmail))
		}

		cmd.Env = env

		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return successfulUpdates, &GitError{
				Command: fmt.Sprintf("git commit --amend (in %s)", repoPath),
				Err:     err,
				Stdout:  stdout.String(),
				Stderr:  stderr.String(),
			}
		}

		successfulUpdates++
	}

	// Checkout to the original branch (force create)
	if _, err := runGitCommand(repoPath, "checkout", "-B", branchName); err != nil {
		return successfulUpdates, fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	// Delete the rewrite-history branch
	if _, err := runGitCommand(repoPath, "branch", "-D", rewriteBranchName); err != nil {
		return successfulUpdates, fmt.Errorf("failed to delete rewrite branch %s: %w", rewriteBranchName, err)
	}

	return successfulUpdates, nil
}
