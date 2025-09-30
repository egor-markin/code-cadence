package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"code-cadence/git"
)

// TestHelper provides utilities for testing
type TestHelper struct {
	TempDir string
	t       *testing.T
}

// NewTestHelper creates a new test helper instance
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{
		TempDir: t.TempDir(),
		t:       t,
	}
}

// CreateGitRepo creates a git repository in the temp directory
func (th *TestHelper) CreateGitRepo(name string) string {
	repoPath := filepath.Join(th.TempDir, name)

	// Create the directory first
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		th.t.Fatalf("Failed to create directory %s: %v", repoPath, err)
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		th.t.Fatalf("Failed to initialize git repository %s: %v", name, err)
	}

	// Set git config to avoid prompts
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		th.t.Fatalf("Failed to set git user.name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		th.t.Fatalf("Failed to set git user.email: %v", err)
	}

	return repoPath
}

// CreateCommit creates a commit in the given repository
func (th *TestHelper) CreateCommit(repoPath, filename, content, message string) string {
	// Create file
	filePath := filepath.Join(repoPath, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		th.t.Fatalf("Failed to create file %s: %v", filename, err)
	}

	// Add file
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		th.t.Fatalf("Failed to add file %s: %v", filename, err)
	}

	// Commit with work hours timestamp
	commitTime := time.Now().Truncate(24 * time.Hour).Add(12 * time.Hour) // 12:00 today
	timeStr := commitTime.Format("2006-01-02T15:04:05")
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE="+timeStr,
		"GIT_COMMITTER_DATE="+timeStr,
	)
	if err := cmd.Run(); err != nil {
		th.t.Fatalf("Failed to commit %s: %v", filename, err)
	}

	// Get commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		th.t.Fatalf("Failed to get commit hash: %v", err)
	}

	return string(output[:8]) // Return short hash
}

// CreateBranch creates a new branch in the repository
func (th *TestHelper) CreateBranch(repoPath, branchName string) {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		th.t.Fatalf("Failed to create branch %s: %v", branchName, err)
	}
}

// SwitchBranch switches to the specified branch
func (th *TestHelper) SwitchBranch(repoPath, branchName string) {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		th.t.Fatalf("Failed to switch to branch %s: %v", branchName, err)
	}
}

// CreateMergeCommit creates a merge commit
func (th *TestHelper) CreateMergeCommit(repoPath, branchName, message string) {
	cmd := exec.Command("git", "merge", "--no-ff", "-m", message, branchName)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if err := cmd.Run(); err != nil {
		th.t.Fatalf("Failed to merge branch %s: %v", branchName, err)
	}
}

// GetCommits returns all commits in the repository
func (th *TestHelper) GetCommits(repoPath string) []git.Commit {
	commits, err := git.GetUnpushedCommits(repoPath, "origin/main")
	if err != nil {
		th.t.Fatalf("Failed to get commits: %v", err)
	}
	return commits
}

// AssertCommitCount checks if the number of commits matches expected
func (th *TestHelper) AssertCommitCount(commits []git.Commit, expected int) {
	if len(commits) != expected {
		th.t.Errorf("Expected %d commits, got %d", expected, len(commits))
	}
}

// CreateTestCommits creates multiple test commits with specific timestamps
func (th *TestHelper) CreateTestCommits(repoPath string, count int, baseTime time.Time) []string {
	var hashes []string

	for i := 0; i < count; i++ {
		filename := filepath.Join(repoPath, fmt.Sprintf("file%d.txt", i))

		content := fmt.Sprintf("Test content %d", i)
		message := fmt.Sprintf("Test commit %d", i)

		// Create file
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			th.t.Fatalf("Failed to create file: %v", err)
		}

		// Add file
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			th.t.Fatalf("Failed to add files: %v", err)
		}

		// Commit with specific timestamp
		commitTime := baseTime.Add(time.Duration(i) * time.Hour)
		timeStr := commitTime.Format("2006-01-02T15:04:05")

		cmd = exec.Command("git", "commit", "-m", message)
		cmd.Dir = repoPath
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test User",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test User",
			"GIT_COMMITTER_EMAIL=test@example.com",
			"GIT_AUTHOR_DATE="+timeStr,
			"GIT_COMMITTER_DATE="+timeStr,
		)
		if err := cmd.Run(); err != nil {
			th.t.Fatalf("Failed to commit: %v", err)
		}

		// Get commit hash
		cmd = exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = repoPath
		output, err := cmd.Output()
		if err != nil {
			th.t.Fatalf("Failed to get commit hash: %v", err)
		}

		hashes = append(hashes, string(output[:8]))
	}

	return hashes
}

// Cleanup removes the temporary directory
func (th *TestHelper) Cleanup() {
	os.RemoveAll(th.TempDir)
}

// TestConfig holds test configuration
type TestConfig struct {
	WorkDayStartHour     int
	WorkDayEndHour       int
	JitterMinutes        int
	ParentGitBranchName  string
	NewCommitAuthorName  string
	NewCommitAuthorEmail string
	CreateBackup         bool
	SkipWeekDays         string
}

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		WorkDayStartHour:     9,
		WorkDayEndHour:       17,
		JitterMinutes:        0, // Disable jitter for predictable tests
		ParentGitBranchName:  "origin/main",
		NewCommitAuthorName:  "Test User",
		NewCommitAuthorEmail: "test@example.com",
		CreateBackup:         false,
		SkipWeekDays:         "Sat,Sun",
	}
}

// ApplyTestConfig applies the test configuration to global variables
func (tc *TestConfig) ApplyTestConfig() {
	WorkDayStartHour = tc.WorkDayStartHour
	WorkDayEndHour = tc.WorkDayEndHour
	JitterMinutes = tc.JitterMinutes
	ParentGitBranchName = tc.ParentGitBranchName
	NewCommitAuthorName = tc.NewCommitAuthorName
	NewCommitAuthorEmail = tc.NewCommitAuthorEmail
	CreateBackup = tc.CreateBackup
	SkipWeekDays = tc.SkipWeekDays
	skipWeekdaysSet = parseWeekdays(tc.SkipWeekDays)
}

// RestoreConfig restores the original configuration
func (tc *TestConfig) RestoreConfig() {
	loadConfig() // Reload from environment
}
