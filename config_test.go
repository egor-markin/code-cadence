package main

import (
	"os"
	"testing"
	"time"
)

func TestConfigurationLoading(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	// Test default configuration
	loadConfig()

	// Verify default values
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
	if NewCommitAuthorName != "" {
		t.Errorf("Expected NewCommitAuthorName to be empty, got '%s'", NewCommitAuthorName)
	}
	if NewCommitAuthorEmail != "" {
		t.Errorf("Expected NewCommitAuthorEmail to be empty, got '%s'", NewCommitAuthorEmail)
	}
	if CreateBackup != false {
		t.Errorf("Expected CreateBackup to be false, got %t", CreateBackup)
	}
	if SkipWeekDays != "Sat,Sun" {
		t.Errorf("Expected SkipWeekDays to be 'Sat,Sun', got '%s'", SkipWeekDays)
	}

	// Verify skipWeekdaysSet is populated
	if skipWeekdaysSet == nil {
		t.Error("Expected skipWeekdaysSet to be populated")
	}
	if !skipWeekdaysSet[time.Saturday] {
		t.Error("Expected Saturday to be in skipWeekdaysSet")
	}
	if !skipWeekdaysSet[time.Sunday] {
		t.Error("Expected Sunday to be in skipWeekdaysSet")
	}
}

func TestConfigurationWithCustomValues(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	// Set custom environment variables
	os.Setenv("WORK_DAY_START_HOUR", "8")
	os.Setenv("WORK_DAY_END_HOUR", "18")
	os.Setenv("JITTER_MINUTES", "15")
	os.Setenv("PARENT_GIT_BRANCH_NAME", "origin/develop")
	os.Setenv("NEW_COMMIT_AUTHOR_NAME", "Custom User")
	os.Setenv("NEW_COMMIT_AUTHOR_EMAIL", "custom@example.com")
	os.Setenv("CREATE_BACKUP", "true")
	os.Setenv("SKIP_WEEK_DAYS", "Fri,Sat,Sun")

	// Load configuration
	loadConfig()

	// Verify custom values
	if WorkDayStartHour != 8 {
		t.Errorf("Expected WorkDayStartHour to be 8, got %d", WorkDayStartHour)
	}
	if WorkDayEndHour != 18 {
		t.Errorf("Expected WorkDayEndHour to be 18, got %d", WorkDayEndHour)
	}
	if JitterMinutes != 15 {
		t.Errorf("Expected JitterMinutes to be 15, got %d", JitterMinutes)
	}
	if ParentGitBranchName != "origin/develop" {
		t.Errorf("Expected ParentGitBranchName to be 'origin/develop', got '%s'", ParentGitBranchName)
	}
	if NewCommitAuthorName != "Custom User" {
		t.Errorf("Expected NewCommitAuthorName to be 'Custom User', got '%s'", NewCommitAuthorName)
	}
	if NewCommitAuthorEmail != "custom@example.com" {
		t.Errorf("Expected NewCommitAuthorEmail to be 'custom@example.com', got '%s'", NewCommitAuthorEmail)
	}
	if CreateBackup != true {
		t.Errorf("Expected CreateBackup to be true, got %t", CreateBackup)
	}
	if SkipWeekDays != "Fri,Sat,Sun" {
		t.Errorf("Expected SkipWeekDays to be 'Fri,Sat,Sun', got '%s'", SkipWeekDays)
	}

	// Verify skipWeekdaysSet is updated
	if !skipWeekdaysSet[time.Friday] {
		t.Error("Expected Friday to be in skipWeekdaysSet")
	}
	if !skipWeekdaysSet[time.Saturday] {
		t.Error("Expected Saturday to be in skipWeekdaysSet")
	}
	if !skipWeekdaysSet[time.Sunday] {
		t.Error("Expected Sunday to be in skipWeekdaysSet")
	}
	if skipWeekdaysSet[time.Monday] {
		t.Error("Expected Monday to not be in skipWeekdaysSet")
	}
}

func TestConfigurationWithInvalidValues(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	// Set invalid environment variables
	os.Setenv("WORK_DAY_START_HOUR", "invalid")
	os.Setenv("WORK_DAY_END_HOUR", "not_a_number")
	os.Setenv("JITTER_MINUTES", "abc")
	os.Setenv("CREATE_BACKUP", "maybe")

	// Load configuration
	loadConfig()

	// Verify default values are used for invalid inputs
	if WorkDayStartHour != 10 {
		t.Errorf("Expected WorkDayStartHour to be 10 (default), got %d", WorkDayStartHour)
	}
	if WorkDayEndHour != 19 {
		t.Errorf("Expected WorkDayEndHour to be 19 (default), got %d", WorkDayEndHour)
	}
	if JitterMinutes != 30 {
		t.Errorf("Expected JitterMinutes to be 30 (default), got %d", JitterMinutes)
	}
	if CreateBackup != false {
		t.Errorf("Expected CreateBackup to be false (default), got %t", CreateBackup)
	}
}

func TestConfigurationJitterMinutesValidation(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	// Test negative jitter minutes
	os.Setenv("JITTER_MINUTES", "-5")
	loadConfig()

	if JitterMinutes != 0 {
		t.Errorf("Expected JitterMinutes to be 0 (clamped), got %d", JitterMinutes)
	}

	// Test zero jitter minutes
	os.Setenv("JITTER_MINUTES", "0")
	loadConfig()

	if JitterMinutes != 0 {
		t.Errorf("Expected JitterMinutes to be 0, got %d", JitterMinutes)
	}

	// Test positive jitter minutes
	os.Setenv("JITTER_MINUTES", "45")
	loadConfig()

	if JitterMinutes != 45 {
		t.Errorf("Expected JitterMinutes to be 45, got %d", JitterMinutes)
	}
}

func TestConfigurationJitterDaysValidation(t *testing.T) {
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

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	// Test false jitter days
	os.Setenv("JITTER_DAYS", "false")
	loadConfig()

	if JitterDays != false {
		t.Errorf("Expected JitterDays to be false, got %t", JitterDays)
	}

	// Test true jitter days
	os.Setenv("JITTER_DAYS", "true")
	loadConfig()

	if JitterDays != true {
		t.Errorf("Expected JitterDays to be true, got %t", JitterDays)
	}

	// Test default value (no env var set)
	os.Unsetenv("JITTER_DAYS")
	loadConfig()

	if !JitterDays {
		t.Errorf("Expected JitterDays to be false (default), got %t", JitterDays)
	}
}

func TestConfigurationSkipWeekDaysVariations(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	tests := []struct {
		name     string
		skipDays string
		expected map[time.Weekday]bool
	}{
		{
			name:     "empty string",
			skipDays: "",
			expected: map[time.Weekday]bool{time.Saturday: true, time.Sunday: true}, // Default value is "Sat,Sun"
		},
		{
			name:     "single day",
			skipDays: "Monday",
			expected: map[time.Weekday]bool{time.Monday: true},
		},
		{
			name:     "multiple days with spaces",
			skipDays: "Friday, Saturday, Sunday",
			expected: map[time.Weekday]bool{
				time.Friday:   true,
				time.Saturday: true,
				time.Sunday:   true,
			},
		},
		{
			name:     "numeric format",
			skipDays: "0,6",
			expected: map[time.Weekday]bool{
				time.Sunday:   true,
				time.Saturday: true,
			},
		},
		{
			name:     "mixed format",
			skipDays: "Mon,2,Wednesday",
			expected: map[time.Weekday]bool{
				time.Monday:    true,
				time.Tuesday:   true,
				time.Wednesday: true,
			},
		},
		{
			name:     "invalid days",
			skipDays: "InvalidDay,Mon,AnotherInvalid",
			expected: map[time.Weekday]bool{time.Monday: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Temporarily rename .env file to prevent it from being loaded
			envBackup := ".env.backup"
			if _, err := os.Stat(".env"); err == nil {
				os.Rename(".env", envBackup)
			}

			// Restore .env file after subtest
			defer func() {
				if _, err := os.Stat(envBackup); err == nil {
					os.Rename(envBackup, ".env")
				}
			}()

			// Clear the environment variable first
			os.Unsetenv("SKIP_WEEK_DAYS")
			if test.skipDays != "" {
				os.Setenv("SKIP_WEEK_DAYS", test.skipDays)
			}
			loadConfig()

			if len(skipWeekdaysSet) != len(test.expected) {
				t.Errorf("Expected %d skip days, got %d", len(test.expected), len(skipWeekdaysSet))
			}

			for weekday, expected := range test.expected {
				if skipWeekdaysSet[weekday] != expected {
					t.Errorf("Expected %v to be %t, got %t", weekday, expected, skipWeekdaysSet[weekday])
				}
			}
		})
	}
}

func TestConfigurationWorkDayHoursValidation(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	// Test valid work day hours
	os.Setenv("WORK_DAY_START_HOUR", "9")
	os.Setenv("WORK_DAY_END_HOUR", "17")
	loadConfig()

	if WorkDayStartHour != 9 {
		t.Errorf("Expected WorkDayStartHour to be 9, got %d", WorkDayStartHour)
	}
	if WorkDayEndHour != 17 {
		t.Errorf("Expected WorkDayEndHour to be 17, got %d", WorkDayEndHour)
	}

	// Test edge cases
	os.Setenv("WORK_DAY_START_HOUR", "0")
	os.Setenv("WORK_DAY_END_HOUR", "23")
	loadConfig()

	if WorkDayStartHour != 0 {
		t.Errorf("Expected WorkDayStartHour to be 0, got %d", WorkDayStartHour)
	}
	if WorkDayEndHour != 23 {
		t.Errorf("Expected WorkDayEndHour to be 23, got %d", WorkDayEndHour)
	}

	// Test invalid values (should use defaults)
	os.Setenv("WORK_DAY_START_HOUR", "invalid")
	os.Setenv("WORK_DAY_END_HOUR", "not_a_number")
	loadConfig()

	if WorkDayStartHour != 10 {
		t.Errorf("Expected WorkDayStartHour to be 10 (default), got %d", WorkDayStartHour)
	}
	if WorkDayEndHour != 19 {
		t.Errorf("Expected WorkDayEndHour to be 19 (default), got %d", WorkDayEndHour)
	}
}

func TestConfigurationBooleanValues(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	// Test various boolean representations
	booleanTests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"TRUE", true},
		{"FALSE", false},
		{"True", true},
		{"False", false},
		{"1", true},
		{"0", false},
		{"yes", true},
		{"no", false},
		{"YES", true},
		{"NO", false},
		{"invalid", false}, // Should default to false
		{"", false},        // Should default to false
	}

	for _, test := range booleanTests {
		t.Run(test.value, func(t *testing.T) {
			os.Setenv("CREATE_BACKUP", test.value)
			loadConfig()

			if CreateBackup != test.expected {
				t.Errorf("Expected CreateBackup to be %t for value '%s', got %t",
					test.expected, test.value, CreateBackup)
			}
		})
	}
}

func TestConfigurationStringValues(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"WORK_DAY_START_HOUR", "WORK_DAY_END_HOUR", "JITTER_MINUTES",
		"PARENT_GIT_BRANCH_NAME", "NEW_COMMIT_AUTHOR_NAME", "NEW_COMMIT_AUTHOR_EMAIL",
		"CREATE_BACKUP", "SKIP_WEEK_DAYS",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Temporarily rename .env file to prevent it from being loaded
	envBackup := ".env.backup"
	if _, err := os.Stat(".env"); err == nil {
		os.Rename(".env", envBackup)
	}

	// Restore environment and .env file after test
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
		// Restore .env file
		if _, err := os.Stat(envBackup); err == nil {
			os.Rename(envBackup, ".env")
		}
	}()

	// Test string values
	os.Setenv("PARENT_GIT_BRANCH_NAME", "origin/feature-branch")
	os.Setenv("NEW_COMMIT_AUTHOR_NAME", "John Doe")
	os.Setenv("NEW_COMMIT_AUTHOR_EMAIL", "john.doe@company.com")

	loadConfig()

	if ParentGitBranchName != "origin/feature-branch" {
		t.Errorf("Expected ParentGitBranchName to be 'origin/feature-branch', got '%s'", ParentGitBranchName)
	}
	if NewCommitAuthorName != "John Doe" {
		t.Errorf("Expected NewCommitAuthorName to be 'John Doe', got '%s'", NewCommitAuthorName)
	}
	if NewCommitAuthorEmail != "john.doe@company.com" {
		t.Errorf("Expected NewCommitAuthorEmail to be 'john.doe@company.com', got '%s'", NewCommitAuthorEmail)
	}

	// Test empty string values
	os.Setenv("NEW_COMMIT_AUTHOR_NAME", "")
	os.Setenv("NEW_COMMIT_AUTHOR_EMAIL", "")

	loadConfig()

	if NewCommitAuthorName != "" {
		t.Errorf("Expected NewCommitAuthorName to be empty, got '%s'", NewCommitAuthorName)
	}
	if NewCommitAuthorEmail != "" {
		t.Errorf("Expected NewCommitAuthorEmail to be empty, got '%s'", NewCommitAuthorEmail)
	}
}
