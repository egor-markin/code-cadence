package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitError(t *testing.T) {
	err := &GitError{
		Command: "git status",
		Err:     exec.ErrNotFound,
		Stdout:  "stdout content",
		Stderr:  "stderr content",
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "git command 'git status' failed") {
		t.Errorf("Error message should contain command info, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "stdout: stdout content") {
		t.Errorf("Error message should contain stdout, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "stderr: stderr content") {
		t.Errorf("Error message should contain stderr, got: %s", errorMsg)
	}
}

func TestCheckGitAvailability(t *testing.T) {
	err := CheckGitAvailability()
	if err != nil {
		t.Errorf("Git should be available in test environment, got error: %v", err)
	}
}

func TestRunGitCommand(t *testing.T) {
	// Test with invalid directory
	_, err := runGitCommand("/nonexistent/directory", "status")
	if err == nil {
		t.Error("Expected error for invalid directory")
	}

	// Test with no arguments
	_, err = runGitCommand(".", "")
	if err == nil {
		t.Error("Expected error for no arguments")
	}

	// Test with valid command in a git repository
	tempDir := t.TempDir()
	// Initialize git repository
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tempDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	output, err := runGitCommand(tempDir, "status")
	if err != nil {
		t.Errorf("Unexpected error for valid git command: %v", err)
	}
	if output == "" {
		t.Error("Expected non-empty output from git status")
	}
}

func TestParseCommitsWithMergeInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Commit
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []Commit{},
		},
		{
			name:     "single line with empty content",
			input:    "\n",
			expected: []Commit{},
		},
		{
			name:  "regular commit",
			input: "abc123|Fix bug|John Doe|john@example.com|2024-01-01 10:00:00 +0000|def456",
			expected: []Commit{
				{
					Hash:      "abc123",
					Subject:   "Fix bug",
					Author:    "John Doe",
					Email:     "john@example.com",
					DateTime:  "2024-01-01 10:00:00 +0000",
					IsMerge:   false,
					MergeFrom: "",
				},
			},
		},
		{
			name:  "merge commit",
			input: "abc123|Merge branch 'feature'|John Doe|john@example.com|2024-01-01 10:00:00 +0000|def456 ghi789",
			expected: []Commit{
				{
					Hash:      "abc123",
					Subject:   "Merge branch 'feature'",
					Author:    "John Doe",
					Email:     "john@example.com",
					DateTime:  "2024-01-01 10:00:00 +0000",
					IsMerge:   true,
					MergeFrom: "ghi789",
				},
			},
		},
		{
			name:  "multiple commits",
			input: "abc123|First commit|John|john@example.com|2024-01-01 10:00:00 +0000|def456\ndef456|Second commit|Jane|jane@example.com|2024-01-01 11:00:00 +0000|ghi789",
			expected: []Commit{
				{
					Hash:      "abc123",
					Subject:   "First commit",
					Author:    "John",
					Email:     "john@example.com",
					DateTime:  "2024-01-01 10:00:00 +0000",
					IsMerge:   false,
					MergeFrom: "",
				},
				{
					Hash:      "def456",
					Subject:   "Second commit",
					Author:    "Jane",
					Email:     "jane@example.com",
					DateTime:  "2024-01-01 11:00:00 +0000",
					IsMerge:   false,
					MergeFrom: "",
				},
			},
		},
		{
			name:     "invalid format",
			input:    "abc123|Incomplete",
			expected: []Commit{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseCommitsWithMergeInfo(test.input)

			if len(result) != len(test.expected) {
				t.Errorf("Expected %d commits, got %d", len(test.expected), len(result))
				return
			}

			for i, expected := range test.expected {
				if result[i].Hash != expected.Hash {
					t.Errorf("Commit %d: expected Hash %s, got %s", i, expected.Hash, result[i].Hash)
				}
				if result[i].Subject != expected.Subject {
					t.Errorf("Commit %d: expected Subject %s, got %s", i, expected.Subject, result[i].Subject)
				}
				if result[i].Author != expected.Author {
					t.Errorf("Commit %d: expected Author %s, got %s", i, expected.Author, result[i].Author)
				}
				if result[i].Email != expected.Email {
					t.Errorf("Commit %d: expected Email %s, got %s", i, expected.Email, result[i].Email)
				}
				if result[i].DateTime != expected.DateTime {
					t.Errorf("Commit %d: expected DateTime %s, got %s", i, expected.DateTime, result[i].DateTime)
				}
				if result[i].IsMerge != expected.IsMerge {
					t.Errorf("Commit %d: expected IsMerge %t, got %t", i, expected.IsMerge, result[i].IsMerge)
				}
				if result[i].MergeFrom != expected.MergeFrom {
					t.Errorf("Commit %d: expected MergeFrom %s, got %s", i, expected.MergeFrom, result[i].MergeFrom)
				}
			}
		})
	}
}

func TestExtractBranchNameFromMergeMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "standard merge message with quotes",
			message:  "Merge branch 'feature-branch' into main",
			expected: "feature-branch",
		},
		{
			name:     "merge message without quotes",
			message:  "Merge branch feature-branch into main",
			expected: "feature-branch",
		},
		{
			name:     "merge commit message",
			message:  "Merge commit abc123 into main",
			expected: "",
		},
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
		{
			name:     "non-merge message",
			message:  "Regular commit message",
			expected: "",
		},
		{
			name:     "merge message with multiple lines",
			message:  "Merge branch 'feature' into main\n\nThis is a merge commit",
			expected: "feature",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := extractBranchNameFromMergeMessage(test.message)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test getting current branch
	branch, err := GetCurrentBranch(tempDir)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch to be 'main' or 'master', got '%s'", branch)
	}
}

func TestGetCurrentBranchDetachedHead(t *testing.T) {
	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get the commit hash and checkout to it (detached HEAD)
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	commitHash := strings.TrimSpace(string(output))

	cmd = exec.Command("git", "checkout", commitHash)
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to checkout commit: %v", err)
	}

	// Test getting current branch in detached HEAD state
	_, err = GetCurrentBranch(tempDir)
	if err == nil {
		t.Error("Expected error for detached HEAD state")
	}
}

func TestGetCurrentBranchNoCommits(t *testing.T) {
	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Test getting current branch with no commits
	// Note: Modern git versions may return a default branch name even for empty repos
	// So we'll just verify the function doesn't crash
	branch, err := GetCurrentBranch(tempDir)
	if err != nil {
		// This is expected for some git versions
		t.Logf("GetCurrentBranch returned error as expected: %v", err)
	} else {
		// This is also acceptable for modern git versions
		t.Logf("GetCurrentBranch returned branch: %s", branch)
	}
}

func TestGetCommitMessage(t *testing.T) {
	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitMessage := "Test commit message\n\nThis is a detailed description."
	cmd = exec.Command("git", "commit", "-m", commitMessage)
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get the commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	commitHash := strings.TrimSpace(string(output))

	// Test getting commit message
	message, err := GetCommitMessage(tempDir, commitHash)
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if !strings.Contains(message, "Test commit message") {
		t.Errorf("Expected commit message to contain 'Test commit message', got: %s", message)
	}
}

func TestGetParentCommit(t *testing.T) {
	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create first commit
	testFile1 := filepath.Join(tempDir, "test1.txt")
	if err := os.WriteFile(testFile1, []byte("test content 1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test1.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "First commit")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get first commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	firstCommitHash := strings.TrimSpace(string(output))

	// Create second commit
	testFile2 := filepath.Join(tempDir, "test2.txt")
	if err := os.WriteFile(testFile2, []byte("test content 2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test2.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Second commit")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get second commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tempDir
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	secondCommitHash := strings.TrimSpace(string(output))

	// Test getting parent commit
	parentHash, err := GetParentCommit(tempDir, secondCommitHash)
	if err != nil {
		t.Fatalf("Failed to get parent commit: %v", err)
	}

	if parentHash != firstCommitHash {
		t.Errorf("Expected parent hash %s, got %s", firstCommitHash, parentHash)
	}
}

func TestGetUnpushedCommits(t *testing.T) {
	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create first commit
	testFile1 := filepath.Join(tempDir, "test1.txt")
	if err := os.WriteFile(testFile1, []byte("test content 1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test1.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "First commit")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create second commit
	testFile2 := filepath.Join(tempDir, "test2.txt")
	if err := os.WriteFile(testFile2, []byte("test content 2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test2.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Second commit")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test getting unpushed commits (should return all commits since no remote)
	commits, err := GetUnpushedCommits(tempDir, "origin/main")
	if err != nil {
		t.Fatalf("Failed to get unpushed commits: %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("Expected 2 unpushed commits, got %d", len(commits))
	}

	// Verify commit details
	if commits[0].Subject != "Second commit" {
		t.Errorf("Expected first commit subject 'Second commit', got '%s'", commits[0].Subject)
	}
	if commits[1].Subject != "First commit" {
		t.Errorf("Expected second commit subject 'First commit', got '%s'", commits[1].Subject)
	}
}

func TestGetUnpushedCommitsNoCommits(t *testing.T) {
	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Test getting unpushed commits with no commits
	commits, err := GetUnpushedCommits(tempDir, "origin/main")
	if err != nil {
		t.Fatalf("Failed to get unpushed commits: %v", err)
	}

	if len(commits) != 0 {
		t.Errorf("Expected 0 unpushed commits, got %d", len(commits))
	}
}

func TestGetUnpushedCommitsInvalidDirectory(t *testing.T) {
	// Test with invalid directory
	_, err := GetUnpushedCommits("/nonexistent/directory", "origin/main")
	if err == nil {
		t.Error("Expected error for invalid directory")
	}
}

// Benchmark tests
func BenchmarkParseCommitsWithMergeInfo(b *testing.B) {
	input := `abc123|First commit|John|john@example.com|2024-01-01 10:00:00 +0000|def456
def456|Second commit|Jane|jane@example.com|2024-01-01 11:00:00 +0000|ghi789
ghi789|Merge branch 'feature'|John|john@example.com|2024-01-01 12:00:00 +0000|jkl012 mno345`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseCommitsWithMergeInfo(input)
	}
}

func BenchmarkExtractBranchNameFromMergeMessage(b *testing.B) {
	message := "Merge branch 'feature-branch' into main\n\nThis is a merge commit"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractBranchNameFromMergeMessage(message)
	}
}
