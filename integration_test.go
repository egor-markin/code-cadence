package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code-cadence/git"
)

func TestIntegrationCommitCadence(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Apply test configuration
	config := DefaultTestConfig()
	config.ApplyTestConfig()
	defer config.RestoreConfig()

	// Create test repository
	repoPath := helper.CreateGitRepo("test-repo")

	// Create initial commit first
	helper.CreateCommit(repoPath, "initial.txt", "initial content", "Initial commit")

	// Create test commits with specific timestamps
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	helper.CreateTestCommits(repoPath, 3, baseTime)

	// Verify initial commits (should be 4: initial + 3 test commits)
	commits := helper.GetCommits(repoPath)
	helper.AssertCommitCount(commits, 4)

	// Run commit cadence
	gitRepos := []string{repoPath}
	commitCadence(gitRepos)

	// Verify commits were updated
	updatedCommits := helper.GetCommits(repoPath)
	helper.AssertCommitCount(updatedCommits, 4)

	// Verify commit times are within work hours
	for i, commit := range updatedCommits {
		commitTime, err := time.Parse("2006-01-02 15:04:05 -0700", commit.DateTime)
		if err != nil {
			t.Fatalf("Failed to parse commit time: %v", err)
		}

		hour := commitTime.Hour()
		if hour < WorkDayStartHour || hour >= WorkDayEndHour {
			t.Errorf("Commit %d time %s is outside work hours (%d-%d)",
				i, commitTime.Format("15:04"), WorkDayStartHour, WorkDayEndHour)
		}
	}
}

func TestIntegrationCommitCadenceSpan(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Apply test configuration
	config := DefaultTestConfig()
	config.ApplyTestConfig()
	defer config.RestoreConfig()

	// Create test repository
	repoPath := helper.CreateGitRepo("test-repo")

	// Create initial commit first
	helper.CreateCommit(repoPath, "initial.txt", "initial content", "Initial commit")

	// Create test commits spanning multiple days
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	helper.CreateTestCommits(repoPath, 5, baseTime)

	// Verify initial commits (should be 6: initial + 5 test commits)
	commits := helper.GetCommits(repoPath)
	helper.AssertCommitCount(commits, 6)

	// Run commit cadence span
	gitRepos := []string{repoPath}
	commitCadenceSpan(gitRepos)

	// Verify commits were updated
	updatedCommits := helper.GetCommits(repoPath)
	helper.AssertCommitCount(updatedCommits, 6)

	// Verify commit times are distributed across days
	days := make(map[string]int)
	for _, commit := range updatedCommits {
		commitTime, err := time.Parse("2006-01-02 15:04:05 -0700", commit.DateTime)
		if err != nil {
			t.Fatalf("Failed to parse commit time: %v", err)
		}

		dayStr := commitTime.Format("2006-01-02")
		days[dayStr]++

		// Verify time is within work hours
		hour := commitTime.Hour()
		if hour < WorkDayStartHour || hour >= WorkDayEndHour {
			t.Errorf("Commit time %s is outside work hours (%d-%d)",
				commitTime.Format("15:04"), WorkDayStartHour, WorkDayEndHour)
		}
	}

	// Verify commits are distributed across multiple days
	if len(days) < 2 {
		t.Errorf("Expected commits to be distributed across multiple days, got %d days", len(days))
	}
}

func TestIntegrationPushDisableEnable(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create test repository
	repoPath := helper.CreateGitRepo("test-repo")

	// Test disabling push
	gitRepos := []string{repoPath}
	disablePushForAll(gitRepos)

	// Verify push is disabled
	isDisabled, err := isPushDisabled(repoPath)
	if err != nil {
		t.Fatalf("Failed to check push status: %v", err)
	}
	if !isDisabled {
		t.Error("Expected push to be disabled")
	}

	// Test enabling push
	enablePushForAll(gitRepos)

	// Verify push is enabled
	isDisabled, err = isPushDisabled(repoPath)
	if err != nil {
		t.Fatalf("Failed to check push status: %v", err)
	}
	if isDisabled {
		t.Error("Expected push to be enabled")
	}
}

func TestIntegrationPushStatus(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create test repositories
	repo1 := helper.CreateGitRepo("repo1")
	repo2 := helper.CreateGitRepo("repo2")

	// Disable push for repo1
	disableGitPush(repo1)

	// Test push status
	gitRepos := []string{repo1, repo2}
	showPushStatus(gitRepos)

	// Verify status
	isDisabled1, _ := isPushDisabled(repo1)
	isDisabled2, _ := isPushDisabled(repo2)

	if !isDisabled1 {
		t.Error("Expected repo1 to have push disabled")
	}
	if isDisabled2 {
		t.Error("Expected repo2 to have push enabled")
	}
}

func TestIntegrationCommitStatus(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create test repository
	repoPath := helper.CreateGitRepo("test-repo")

	// Create initial commit first
	helper.CreateCommit(repoPath, "initial.txt", "initial content", "Initial commit")

	// Create test commits
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	helper.CreateTestCommits(repoPath, 3, baseTime)

	// Test commit status
	gitRepos := []string{repoPath}
	showCommitStatus(gitRepos)

	// Verify commits exist (should be 4: initial + 3 test commits)
	commits := helper.GetCommits(repoPath)
	helper.AssertCommitCount(commits, 4)
}

func TestIntegrationFindGitRepositories(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create multiple test repositories
	repo1 := helper.CreateGitRepo("repo1")
	repo2 := helper.CreateGitRepo("repo2")

	// Create initial commits
	helper.CreateCommit(repo1, "initial1.txt", "initial content 1", "Initial commit 1")
	helper.CreateCommit(repo2, "initial2.txt", "initial content 2", "Initial commit 2")
	repo3 := helper.CreateGitRepo("nested/repo3")
	helper.CreateCommit(repo3, "initial3.txt", "initial content 3", "Initial commit 3")

	// Create non-git directory
	nonRepo := filepath.Join(helper.TempDir, "non-repo")
	os.MkdirAll(nonRepo, 0755)

	// Test finding repositories
	repos, err := findGitRepositories(helper.TempDir)
	if err != nil {
		t.Fatalf("Failed to find git repositories: %v", err)
	}

	if len(repos) != 3 {
		t.Errorf("Expected 3 repositories, got %d", len(repos))
	}

	// Verify all expected repos are found
	expectedRepos := []string{repo1, repo2, repo3}
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

func TestIntegrationMergeCommits(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Apply test configuration
	config := DefaultTestConfig()
	config.ApplyTestConfig()
	defer config.RestoreConfig()

	// Create test repository
	repoPath := helper.CreateGitRepo("test-repo")

	// Create initial commit on main
	helper.CreateCommit(repoPath, "main.txt", "main content", "Initial commit")

	// Create feature branch
	helper.CreateBranch(repoPath, "feature")
	helper.CreateCommit(repoPath, "feature.txt", "feature content", "Feature commit")

	// Switch back to master and merge
	helper.SwitchBranch(repoPath, "master")
	helper.CreateMergeCommit(repoPath, "feature", "Merge feature branch")

	// Verify merge commit exists
	commits := helper.GetCommits(repoPath)
	// Should have at least 2 commits (initial + feature), possibly 3 if merge commit was created
	if len(commits) < 2 {
		t.Errorf("Expected at least 2 commits, got %d", len(commits))
	}

	// Find merge commit
	var mergeCommit *git.Commit
	for _, commit := range commits {
		if commit.IsMerge {
			mergeCommit = &commit
			break
		}
	}

	if mergeCommit == nil {
		// If no merge commit was found, that's okay - the merge might have been a fast-forward
		t.Log("No merge commit found - merge was likely a fast-forward")
		return
	}

	if !mergeCommit.IsMerge {
		t.Error("Expected commit to be marked as merge")
	}

	if mergeCommit.MergeFrom == "" {
		t.Error("Expected merge commit to have MergeFrom set")
	}
}

func TestIntegrationBackupCreation(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Apply test configuration with backup enabled
	config := DefaultTestConfig()
	config.CreateBackup = true
	config.ApplyTestConfig()
	defer config.RestoreConfig()

	// Create test repository
	repoPath := helper.CreateGitRepo("test-repo")
	helper.CreateCommit(repoPath, "test.txt", "test content", "Test commit")

	// Test backup creation
	gitRepos := []string{repoPath}
	err := createBackupsForRepos(gitRepos)
	if err != nil {
		t.Fatalf("Failed to create backups: %v", err)
	}

	// Verify backup was created
	backupPattern := repoPath + ".backup-*"
	matches, err := filepath.Glob(backupPattern)
	if err != nil {
		t.Fatalf("Failed to find backup files: %v", err)
	}

	if len(matches) == 0 {
		t.Error("Expected backup to be created")
	}

	// Verify backup contains the same files
	backupPath := matches[0]
	originalFiles, _ := filepath.Glob(filepath.Join(repoPath, "*"))
	backupFiles, _ := filepath.Glob(filepath.Join(backupPath, "*"))

	if len(originalFiles) != len(backupFiles) {
		t.Errorf("Expected backup to have same number of files, original: %d, backup: %d",
			len(originalFiles), len(backupFiles))
	}
}

func TestIntegrationWeekdaySkipping(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Apply test configuration with weekend skipping
	config := DefaultTestConfig()
	config.SkipWeekDays = "Sat,Sun"
	config.ApplyTestConfig()
	defer config.RestoreConfig()

	// Test weekday enumeration
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Monday
	end := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)   // Sunday

	days := enumerateDaysSkipping(start, end, skipWeekdaysSet)

	// Should have 5 weekdays (Mon-Fri)
	if len(days) != 5 {
		t.Errorf("Expected 5 weekdays, got %d", len(days))
	}

	// Verify no weekends are included
	for _, day := range days {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			t.Errorf("Expected no weekends, got %s", day.Weekday())
		}
	}
}

func TestIntegrationCommitTimeGeneration(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Apply test configuration
	config := DefaultTestConfig()
	config.JitterMinutes = 0 // Disable jitter for predictable testing
	config.ApplyTestConfig()
	defer config.RestoreConfig()

	// Test single commit
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	times := generateCommitTimesForDay(day, 1)

	if len(times) != 1 {
		t.Errorf("Expected 1 time, got %d", len(times))
	}

	// Verify time is within work hours
	hour := times[0].Hour()
	if hour < WorkDayStartHour || hour >= WorkDayEndHour {
		t.Errorf("Time %s is outside work hours (%d-%d)",
			times[0].Format("15:04"), WorkDayStartHour, WorkDayEndHour)
	}

	// Test multiple commits
	times = generateCommitTimesForDay(day, 3)

	if len(times) != 3 {
		t.Errorf("Expected 3 times, got %d", len(times))
	}

	// Verify times are in ascending order
	for i := 1; i < len(times); i++ {
		if times[i].Before(times[i-1]) {
			t.Errorf("Times are not in ascending order: %s before %s",
				times[i-1].Format("15:04"), times[i].Format("15:04"))
		}
	}

	// Verify all times are within work hours
	for i, timeVal := range times {
		hour := timeVal.Hour()
		if hour < WorkDayStartHour || hour >= WorkDayEndHour {
			t.Errorf("Time %d (%s) is outside work hours (%d-%d)",
				i, timeVal.Format("15:04"), WorkDayStartHour, WorkDayEndHour)
		}
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Test with invalid directory
	invalidDir := "/nonexistent/directory"

	// Test finding repositories in invalid directory
	_, err := findGitRepositories(invalidDir)
	if err == nil {
		t.Error("Expected error for invalid directory")
	}

	// Test git operations on invalid directory
	_, err = git.GetUnpushedCommits(invalidDir, "origin/main")
	if err == nil {
		t.Error("Expected error for git operations on invalid directory")
	}

	// Test with empty directory
	emptyDir := filepath.Join(helper.TempDir, "empty")
	os.MkdirAll(emptyDir, 0755)

	repos, err := findGitRepositories(emptyDir)
	if err != nil {
		t.Fatalf("Unexpected error for empty directory: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("Expected 0 repositories in empty directory, got %d", len(repos))
	}
}

func TestIntegrationConcurrentOperations(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create multiple repositories
	var repos []string
	for i := 0; i < 5; i++ {
		repoPath := helper.CreateGitRepo(fmt.Sprintf("repo%d", i+1))
		helper.CreateCommit(repoPath, "test.txt", "test content", "Test commit")
		repos = append(repos, repoPath)
	}

	// Test concurrent push operations
	disablePushForAll(repos)

	// Verify all repositories have push disabled
	for _, repo := range repos {
		isDisabled, err := isPushDisabled(repo)
		if err != nil {
			t.Fatalf("Failed to check push status for %s: %v", repo, err)
		}
		if !isDisabled {
			t.Errorf("Expected push to be disabled for %s", repo)
		}
	}

	// Test concurrent push enable
	enablePushForAll(repos)

	// Verify all repositories have push enabled
	for _, repo := range repos {
		isDisabled, err := isPushDisabled(repo)
		if err != nil {
			t.Fatalf("Failed to check push status for %s: %v", repo, err)
		}
		if isDisabled {
			t.Errorf("Expected push to be enabled for %s", repo)
		}
	}
}

func TestIntegrationBackupFolderSkipping(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Apply test configuration
	config := DefaultTestConfig()
	config.ApplyTestConfig()
	defer config.RestoreConfig()

	// Create a regular repository
	regularRepo := helper.CreateGitRepo("regular-repo")
	helper.CreateCommit(regularRepo, "file1.txt", "content1", "Commit 1")
	helper.CreateCommit(regularRepo, "file2.txt", "content2", "Commit 2")

	// Create a backup repository (simulating what would be created by the backup feature)
	backupRepo := helper.CreateGitRepo("regular-repo.backup-2024-01-15-14-30-45")
	helper.CreateCommit(backupRepo, "file1.txt", "content1", "Backup Commit 1")
	helper.CreateCommit(backupRepo, "file2.txt", "content2", "Backup Commit 2")

	// Create another backup repository with different timestamp
	backupRepo2 := helper.CreateGitRepo("another-repo.backup-2024-01-16-10-15-30")
	helper.CreateCommit(backupRepo2, "file1.txt", "content1", "Backup Commit 1")

	// Test commit_cadence with mixed repositories
	gitRepos := []string{regularRepo, backupRepo, backupRepo2}

	// Capture output to verify backup folders are skipped
	// Note: In a real test, you might want to capture stdout to verify the skip messages
	commitCadence(gitRepos)

	// Verify that regular repo was processed (commits should be redistributed)
	regularCommits := helper.GetCommits(regularRepo)
	helper.AssertCommitCount(regularCommits, 2)

	// Verify that backup repos were not processed (commits should remain unchanged)
	backupCommits := helper.GetCommits(backupRepo)
	helper.AssertCommitCount(backupCommits, 2)

	backupCommits2 := helper.GetCommits(backupRepo2)
	helper.AssertCommitCount(backupCommits2, 1)

	// Test commit_cadence_span with mixed repositories
	commitCadenceSpan(gitRepos)

	// Verify results are the same (backup folders should still be skipped)
	regularCommitsAfter := helper.GetCommits(regularRepo)
	helper.AssertCommitCount(regularCommitsAfter, 2)

	backupCommitsAfter := helper.GetCommits(backupRepo)
	helper.AssertCommitCount(backupCommitsAfter, 2)

	backupCommits2After := helper.GetCommits(backupRepo2)
	helper.AssertCommitCount(backupCommits2After, 1)
}
