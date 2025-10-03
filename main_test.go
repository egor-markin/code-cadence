package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES", "JITTER_DAYS",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env files to prevent them from being loaded
	envBackup := ".env.backup"
	systemEnvBackup := "/usr/local/etc/code-cadence/.env.backup"
	envExists := false
	systemEnvExists := false

	if _, err := os.Stat(".env"); err == nil {
		envExists = true
		if err := os.Rename(".env", envBackup); err != nil {
			t.Fatalf("Failed to rename .env file: %v", err)
		}
	}

	if _, err := os.Stat("/usr/local/etc/code-cadence/.env"); err == nil {
		systemEnvExists = true
		if err := os.Rename("/usr/local/etc/code-cadence/.env", systemEnvBackup); err != nil {
			t.Fatalf("Failed to rename system .env file: %v", err)
		}
	}

	// Restore environment and .env files after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env files
		if envExists {
			if err := os.Rename(envBackup, ".env"); err != nil {
				t.Logf("Failed to restore .env file: %v", err)
			}
		}
		if systemEnvExists {
			if err := os.Rename(systemEnvBackup, "/usr/local/etc/code-cadence/.env"); err != nil {
				t.Logf("Failed to restore system .env file: %v", err)
			}
		}
	}()

	// Test default values
	loadConfig()

	if WorkDayStartHour != 10 {
		t.Errorf("Expected WorkDayStartHour to be 10, got %d", WorkDayStartHour)
	}
	if WorkDayEndHour != 19 {
		t.Errorf("Expected WorkDayEndHour to be 19, got %d", WorkDayEndHour)
	}
	if JitterMinutes != 30 {
		t.Errorf("Expected JitterMinutes to be 30, got %d", JitterMinutes)
	}
	if ParentGitBranchName != "origin/main" {
		t.Errorf("Expected ParentGitBranchName to be 'origin/main', got '%s'", ParentGitBranchName)
	}
	if CreateBackup != false {
		t.Errorf("Expected CreateBackup to be false, got %t", CreateBackup)
	}
	if SkipWeekDays != "Sat,Sun" {
		t.Errorf("Expected SkipWeekDays to be 'Sat,Sun', got '%s'", SkipWeekDays)
	}

	// Test custom values
	os.Setenv("WORK_DAY_START_HOUR", "9")
	os.Setenv("WORK_DAY_END_HOUR", "17")
	os.Setenv("JITTER_MINUTES", "15")
	os.Setenv("PARENT_GIT_BRANCH_NAME", "origin/develop")
	os.Setenv("NEW_COMMIT_AUTHOR_NAME", "Test User")
	os.Setenv("NEW_COMMIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("CREATE_BACKUP", "true")
	os.Setenv("SKIP_WEEK_DAYS", "Fri,Sat,Sun")

	loadConfig()

	if WorkDayStartHour != 9 {
		t.Errorf("Expected WorkDayStartHour to be 9, got %d", WorkDayStartHour)
	}
	if WorkDayEndHour != 17 {
		t.Errorf("Expected WorkDayEndHour to be 17, got %d", WorkDayEndHour)
	}
	if JitterMinutes != 15 {
		t.Errorf("Expected JitterMinutes to be 15, got %d", JitterMinutes)
	}
	if ParentGitBranchName != "origin/develop" {
		t.Errorf("Expected ParentGitBranchName to be 'origin/develop', got '%s'", ParentGitBranchName)
	}
	if NewCommitAuthorName != "Test User" {
		t.Errorf("Expected NewCommitAuthorName to be 'Test User', got '%s'", NewCommitAuthorName)
	}
	if NewCommitAuthorEmail != "test@example.com" {
		t.Errorf("Expected NewCommitAuthorEmail to be 'test@example.com', got '%s'", NewCommitAuthorEmail)
	}
	if CreateBackup != true {
		t.Errorf("Expected CreateBackup to be true, got %t", CreateBackup)
	}
	if SkipWeekDays != "Fri,Sat,Sun" {
		t.Errorf("Expected SkipWeekDays to be 'Fri,Sat,Sun', got '%s'", SkipWeekDays)
	}
}

func TestGetEnvString(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnvString("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}

	// Test with non-existing environment variable
	result = getEnvString("NON_EXISTING_VAR", "default_value")
	if result != "default_value" {
		t.Errorf("Expected 'default_value', got '%s'", result)
	}

	// Test with empty environment variable
	os.Setenv("EMPTY_VAR", "")
	result = getEnvString("EMPTY_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

func TestGetEnvInt(t *testing.T) {
	// Test with valid integer
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	result := getEnvInt("TEST_INT", 0)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	// Test with invalid integer
	os.Setenv("INVALID_INT", "not_a_number")
	result = getEnvInt("INVALID_INT", 10)
	if result != 10 {
		t.Errorf("Expected 10, got %d", result)
	}

	// Test with non-existing variable
	result = getEnvInt("NON_EXISTING", 5)
	if result != 5 {
		t.Errorf("Expected 5, got %d", result)
	}
}

func TestGetEnvBool(t *testing.T) {
	// Test with true
	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")

	result := getEnvBool("TEST_BOOL", false)
	if result != true {
		t.Errorf("Expected true, got %t", result)
	}

	// Test with false
	os.Setenv("TEST_BOOL", "false")
	result = getEnvBool("TEST_BOOL", true)
	if result != false {
		t.Errorf("Expected false, got %t", result)
	}

	// Test with invalid boolean
	os.Setenv("INVALID_BOOL", "maybe")
	result = getEnvBool("INVALID_BOOL", true)
	if result != true {
		t.Errorf("Expected true, got %t", result)
	}

	// Test with non-existing variable
	result = getEnvBool("NON_EXISTING", false)
	if result != false {
		t.Errorf("Expected false, got %t", result)
	}
}

func TestFindGitRepositories(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create some directories
	repo1 := filepath.Join(tempDir, "repo1")
	repo2 := filepath.Join(tempDir, "repo2")
	nonRepo := filepath.Join(tempDir, "non-repo")

	os.MkdirAll(repo1, 0755)
	os.MkdirAll(repo2, 0755)
	os.MkdirAll(nonRepo, 0755)

	// Create .git directories
	os.MkdirAll(filepath.Join(repo1, ".git"), 0755)
	os.MkdirAll(filepath.Join(repo2, ".git"), 0755)

	// Create a nested repo
	nestedRepo := filepath.Join(tempDir, "parent", "nested-repo")
	os.MkdirAll(filepath.Join(nestedRepo, ".git"), 0755)

	// Test finding repositories
	repos, err := findGitRepositories(tempDir)
	if err != nil {
		t.Fatalf("Error finding git repositories: %v", err)
	}

	if len(repos) != 3 {
		t.Errorf("Expected 3 repositories, got %d", len(repos))
	}

	// Verify all expected repos are found
	expectedRepos := []string{repo1, repo2, nestedRepo}
	for _, expected := range expectedRepos {
		found := false
		for _, repo := range repos {
			if repo == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find repository %s", expected)
		}
	}
}

func TestDisableEnableGitPush(t *testing.T) {
	// Create a temporary directory with .git structure
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git", "hooks")
	os.MkdirAll(gitDir, 0755)

	// Test disabling push
	err := disableGitPush(tempDir)
	if err != nil {
		t.Fatalf("Error disabling git push: %v", err)
	}

	// Verify hook was created
	hookPath := filepath.Join(gitDir, "pre-push")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Error("Pre-push hook was not created")
	}

	// Verify hook content
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("Error reading hook content: %v", err)
	}

	if !strings.Contains(string(content), "git push is disabled for this repository") {
		t.Error("Hook content does not contain expected disable message")
	}

	// Test checking if push is disabled
	isDisabled, err := isPushDisabled(tempDir)
	if err != nil {
		t.Fatalf("Error checking push status: %v", err)
	}
	if !isDisabled {
		t.Error("Expected push to be disabled")
	}

	// Test enabling push
	err = enableGitPush(tempDir)
	if err != nil {
		t.Fatalf("Error enabling git push: %v", err)
	}

	// Verify hook was removed
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("Pre-push hook was not removed")
	}

	// Test checking if push is enabled
	isDisabled, err = isPushDisabled(tempDir)
	if err != nil {
		t.Fatalf("Error checking push status: %v", err)
	}
	if isDisabled {
		t.Error("Expected push to be enabled")
	}
}

func TestValidCommands(t *testing.T) {
	expectedCommands := []string{
		CmdPushDisable,
		CmdPushEnable,
		CmdPushStatus,
		CmdCommitStatus,
		CmdCommitCadence,
		CmdCommitCadenceSpan,
	}

	if len(validCommands) != len(expectedCommands) {
		t.Errorf("Expected %d valid commands, got %d", len(expectedCommands), len(validCommands))
	}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range validCommands {
			if cmd == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command '%s' not found in validCommands", expected)
		}
	}
}

func TestIsBackupFolder(t *testing.T) {
	tests := []struct {
		name     string
		repoPath string
		expected bool
	}{
		{
			name:     "regular repository path",
			repoPath: "/path/to/my-repo",
			expected: false,
		},
		{
			name:     "backup folder with timestamp",
			repoPath: "/path/to/my-repo.backup-2024-01-15-14-30-45",
			expected: true,
		},
		{
			name:     "backup folder in nested path",
			repoPath: "/home/user/workspace/project.backup-2024-01-15-14-30-45",
			expected: true,
		},
		{
			name:     "folder with backup in middle of name",
			repoPath: "/path/to/my-backup-repo",
			expected: false,
		},
		{
			name:     "folder ending with backup pattern",
			repoPath: "/path/to/something.backup-",
			expected: true,
		},
		{
			name:     "folder with backup pattern but no timestamp",
			repoPath: "/path/to/repo.backup",
			expected: false,
		},
		{
			name:     "empty path",
			repoPath: "",
			expected: false,
		},
		{
			name:     "just backup pattern",
			repoPath: ".backup-2024-01-15",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBackupFolder(tt.repoPath)
			if result != tt.expected {
				t.Errorf("isBackupFolder(%q) = %v, expected %v", tt.repoPath, result, tt.expected)
			}
		})
	}
}
