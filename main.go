package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"code-cadence/git"

	"github.com/joho/godotenv"
)

// Configuration variables loaded from environment
var (
	WorkDayStartHour     int
	WorkDayEndHour       int
	JitterMinutes        int
	JitterDays           bool
	ParentGitBranchName  string
	NewCommitAuthorName  string
	NewCommitAuthorEmail string
	CreateBackup         bool
)

// Additional configuration
var (
	SkipWeekDays    string
	skipWeekdaysSet map[time.Weekday]bool
)

// .env file locations to try in order
var envFileLocations = []string{
	".env",                             // Current directory
	"~/.config/code-cadence/.env",      // User config
	"/opt/code-cadence/.env",           // Application directory
	"/usr/local/etc/code-cadence/.env", // System-wide config
}

// loadConfig loads configuration from .env file with defaults
func loadConfig() {
	// Try to load .env file from multiple locations (ignore errors if files don't exist)
	for _, envFile := range envFileLocations {
		_ = godotenv.Load(envFile)
	}

	// Load with defaults
	WorkDayStartHour = getEnvInt("WORK_DAY_START_HOUR", 10)
	WorkDayEndHour = getEnvInt("WORK_DAY_END_HOUR", 19)
	JitterMinutes = getEnvInt("JITTER_MINUTES", 30)
	JitterDays = getEnvBool("JITTER_DAYS", true)
	ParentGitBranchName = getEnvString("PARENT_GIT_BRANCH_NAME", "origin/main")
	NewCommitAuthorName = getEnvString("NEW_COMMIT_AUTHOR_NAME", "")
	NewCommitAuthorEmail = getEnvString("NEW_COMMIT_AUTHOR_EMAIL", "")
	CreateBackup = getEnvBool("CREATE_BACKUP", false)

	// Weekday skipping configuration for commit_cadence_span
	SkipWeekDays = getEnvString("SKIP_WEEK_DAYS", "Sat,Sun")
	skipWeekdaysSet = parseWeekdays(SkipWeekDays)

	if JitterMinutes < 0 {
		JitterMinutes = 0
	}
}

// getEnvString gets environment variable with default
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets environment variable as int with default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool gets environment variable as bool with default
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		// Handle common boolean representations
		lowerValue := strings.ToLower(strings.TrimSpace(value))
		switch lowerValue {
		case "true", "1", "yes", "on", "enabled":
			return true
		case "false", "0", "no", "off", "disabled":
			return false
		}
		// Fall back to strconv.ParseBool for other formats
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// Command constants
const (
	CmdPushDisable       = "push_disable"
	CmdPushEnable        = "push_enable"
	CmdPushStatus        = "push_status"
	CmdCommitStatus      = "commit_status"
	CmdCommitCadence     = "commit_cadence"
	CmdCommitCadenceSpan = "commit_cadence_span"
)

// Valid commands slice
var validCommands = []string{
	CmdPushDisable,
	CmdPushEnable,
	CmdPushStatus,
	CmdCommitStatus,
	CmdCommitCadence,
	CmdCommitCadenceSpan,
}

// RewriteBranchName The temporary Git branch name that is used for rewriting commit times
const RewriteBranchName = "rewrite-history"

// BackupFolderPattern is the pattern used to identify backup folders created by this tool
const BackupFolderPattern = ".backup-"

// Directories to skip when scanning for git repositories
var skipDirs = []string{
	"node_modules",
	"vendor",
	"target",
	"build",
}

const prePushHookContent = `#!/bin/sh
echo "Error: git push is disabled for this repository"
echo "This repository has been configured to prevent pushing changes"
exit 1
`

func main() {
	// Load configuration from environment
	loadConfig()

	if len(os.Args) != 3 {
		fmt.Println("Usage: code-cadence <command> <directory_path>")
		fmt.Println("Commands:")
		fmt.Println("  push_disable        - Disable git push for all repositories")
		fmt.Println("  push_enable         - Enable git push for all repositories")
		fmt.Println("  push_status         - Show push status for all repositories")
		fmt.Println("  commit_status       - Show unpushed commits for all repositories")
		fmt.Println("  commit_cadence      - Redistribute unpushed commit times across work day")
		fmt.Println("  commit_cadence_span - Redistribute unpushed commit times across all days since last push (skips configured weekdays)")
		fmt.Println("")
		fmt.Println("Example: code-cadence commit_status /home/user/workspace/")
		os.Exit(1)
	}

	command := os.Args[1]
	rootDir := os.Args[2]

	// Validate command
	if !slices.Contains(validCommands, command) {
		fmt.Printf("Error: Invalid command '%s'. Valid commands are: %s\n", command, strings.Join(validCommands, ", "))
		os.Exit(1)
	}

	// Check if directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		fmt.Printf("Error: Directory '%s' does not exist\n", rootDir)
		os.Exit(1)
	}

	// Check git availability
	if err := git.CheckGitAvailability(); err != nil {
		fmt.Printf("Error: Git is not available or not working properly: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scanning directory: %s\n", rootDir)

	gitRepos, err := findGitRepositories(rootDir)
	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if len(gitRepos) == 0 {
		fmt.Println("No Git repositories found in the specified directory")
		os.Exit(0)
	}

	fmt.Printf("Found %d Git repositories:\n", len(gitRepos))
	for _, repo := range gitRepos {
		fmt.Printf("  - %s\n", repo)
	}

	fmt.Println()

	switch command {
	case CmdPushDisable:
		disablePushForAll(gitRepos)
	case CmdPushEnable:
		enablePushForAll(gitRepos)
	case CmdPushStatus:
		showPushStatus(gitRepos)
	case CmdCommitStatus:
		showCommitStatus(gitRepos)
	case CmdCommitCadence:
		commitCadence(gitRepos)
	case CmdCommitCadenceSpan:
		commitCadenceSpan(gitRepos)
	}
}

func findGitRepositories(rootDir string) ([]string, error) {
	var gitRepos []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and common non-repo directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") && name != ".git" {
				return filepath.SkipDir
			}

			// Skip common directories that are unlikely to be repos
			if slices.Contains(skipDirs, name) {
				return filepath.SkipDir
			}
		}

		// Check if this is a .git directory
		if info.IsDir() && info.Name() == ".git" {
			// Get the parent directory (the actual repository root)
			repoPath := filepath.Dir(path)
			gitRepos = append(gitRepos, repoPath)
			return filepath.SkipDir // Don't traverse into .git directory
		}

		return nil
	})

	return gitRepos, err
}

func disablePushForAll(gitRepos []string) {
	fmt.Println("Disabling git push for all repositories...")

	disabledCount := 0
	for _, repo := range gitRepos {
		if err := disableGitPush(repo); err != nil {
			fmt.Printf("Warning: Failed to disable git push for %s: %v\n", repo, err)
		} else {
			disabledCount++
			fmt.Printf("✓ Disabled git push for: %s\n", repo)
		}
	}

	fmt.Printf("\nSummary: Successfully disabled git push for %d/%d repositories\n", disabledCount, len(gitRepos))
}

func enablePushForAll(gitRepos []string) {
	fmt.Println("Enabling git push for all repositories...")

	enabledCount := 0
	for _, repo := range gitRepos {
		if err := enableGitPush(repo); err != nil {
			fmt.Printf("Warning: Failed to enable git push for %s: %v\n", repo, err)
		} else {
			enabledCount++
			fmt.Printf("✓ Enabled git push for: %s\n", repo)
		}
	}

	fmt.Printf("\nSummary: Successfully enabled git push for %d/%d repositories\n", enabledCount, len(gitRepos))
}

func showPushStatus(gitRepos []string) {
	fmt.Println("Checking push status for all repositories...")

	disabledCount := 0
	enabledCount := 0

	for _, repo := range gitRepos {
		isDisabled, err := isPushDisabled(repo)
		if err != nil {
			fmt.Printf("Warning: Could not check status for %s: %v\n", repo, err)
			continue
		}

		if isDisabled {
			disabledCount++
			fmt.Printf("❌ Push DISABLED: %s\n", repo)
		} else {
			enabledCount++
			fmt.Printf("✅ Push ENABLED:  %s\n", repo)
		}
	}

	fmt.Printf("\nSummary: %d repositories have push enabled, %d have push disabled\n", enabledCount, disabledCount)
}

func disableGitPush(repoPath string) error {
	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	prePushHookPath := filepath.Join(hooksDir, "pre-push")

	// Create hooks directory if it doesn't exist
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Write the pre-push hook
	if err := os.WriteFile(prePushHookPath, []byte(prePushHookContent), 0755); err != nil {
		return fmt.Errorf("failed to write pre-push hook: %w", err)
	}

	return nil
}

func enableGitPush(repoPath string) error {
	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	prePushHookPath := filepath.Join(hooksDir, "pre-push")

	// Remove the pre-push hook if it exists
	if err := os.Remove(prePushHookPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove pre-push hook: %w", err)
	}

	return nil
}

func isPushDisabled(repoPath string) (bool, error) {
	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	prePushHookPath := filepath.Join(hooksDir, "pre-push")

	// Check if pre-push hook exists
	if _, err := os.Stat(prePushHookPath); os.IsNotExist(err) {
		return false, nil // Push is enabled (no hook exists)
	} else if err != nil {
		return false, fmt.Errorf("failed to check pre-push hook: %w", err)
	}

	// Read the hook content to verify it's our disable hook
	content, err := os.ReadFile(prePushHookPath)
	if err != nil {
		return false, fmt.Errorf("failed to read pre-push hook: %w", err)
	}

	// Check if it contains our disable message
	return strings.Contains(string(content), "git push is disabled for this repository"), nil
}

func showCommitStatus(gitRepos []string) {
	fmt.Println("Checking for unpushed commits in all repositories...")

	reposWithUnpushedCommits := 0
	totalUnpushedCommits := 0

	for _, repo := range gitRepos {
		unpushedCommits, err := git.GetUnpushedCommits(repo, ParentGitBranchName)
		if err != nil {
			fmt.Printf("Warning: Could not check commits for %s: %v\n", repo, err)
			continue
		}

		if len(unpushedCommits) > 0 {
			reposWithUnpushedCommits++
			totalUnpushedCommits += len(unpushedCommits)
			fmt.Printf("\n📦 %s (%d unpushed commits):\n", repo, len(unpushedCommits))
			for _, commit := range unpushedCommits {
				fmt.Printf("   • %s %s (%s <%s> - %s)\n", commit.Hash, commit.Subject, commit.Author, commit.Email, commit.DateTime)
			}
		} else {
			fmt.Printf("✅ %s: All commits pushed\n", repo)
		}
	}

	fmt.Printf("\nSummary: %d repositories have unpushed commits (%d total unpushed commits)\n",
		reposWithUnpushedCommits, totalUnpushedCommits)
}

// isBackupFolder checks if a git repository path matches the backup folder pattern
func isBackupFolder(repoPath string) bool {
	baseName := filepath.Base(repoPath)
	return strings.Contains(baseName, BackupFolderPattern)
}

// commitCadence redistributes unpushed commit times across work day
func commitCadence(gitRepos []string) {
	fmt.Println("Redistributing unpushed commit times across work day...")

	fmt.Println()

	// Create backups if enabled
	if err := createBackupsForRepos(gitRepos); err != nil {
		fmt.Printf("Warning: Failed to create backups: %v\n", err)
	}

	fmt.Println()

	processedRepos := 0
	totalCommitsUpdated := 0

	for _, repo := range gitRepos {
		// Skip backup folders
		if isBackupFolder(repo) {
			fmt.Printf("⏭️  Skipping backup folder: %s\n", repo)
			continue
		}

		unpushedCommits, err := git.GetUnpushedCommits(repo, ParentGitBranchName)
		if err != nil {
			fmt.Printf("Warning: Could not check commits for %s: %v\n", repo, err)
			continue
		}

		if len(unpushedCommits) == 0 {
			fmt.Printf("✅ %s: No unpushed commits to redistribute\n", repo)
			continue
		}

		fmt.Printf("\n📦 %s (%d unpushed commits):\n", repo, len(unpushedCommits))

		// Get current branch name
		currentBranch, err := git.GetCurrentBranch(repo)
		if err != nil {
			fmt.Printf("   ❌ Error: Could not get current branch for %s: %v\n", repo, err)
			os.Exit(1)
		}
		fmt.Printf("   🌿 Current branch: %s\n", currentBranch)

		// Find parent commit of the first unpushed commit (last in the slice since they're in reverse chronological order)
		firstUnpushedCommit := unpushedCommits[len(unpushedCommits)-1]
		parentCommitHash, err := git.GetParentCommit(repo, firstUnpushedCommit.Hash)
		if err != nil {
			// If this is the first commit in the repository, use empty tree as parent
			fmt.Printf("   ⚠️  First commit in repository, using empty tree as parent\n")
			parentCommitHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904" // Empty tree hash
		} else {
			fmt.Printf("   📍 Parent commit: %s\n", parentCommitHash)
		}

		// Group commits by day
		commitsByDay := groupCommitsByDay(unpushedCommits)

		// Collect all commits and their new times across all days
		var allCommits []git.Commit
		var allNewTimes []time.Time

		// Sort days to process them in chronological order (earliest to latest)
		var sortedDays []string
		for dayStr := range commitsByDay {
			sortedDays = append(sortedDays, dayStr)
		}
		sort.Strings(sortedDays) // YYYY-MM-DD format sorts chronologically

		for _, dayStr := range sortedDays {
			dayCommits := commitsByDay[dayStr]
			fmt.Printf("   📅 %s (%d commits):\n", dayStr, len(dayCommits))

			// Get timezone from the first commit of the day
			firstCommit := dayCommits[0]
			firstCommitTime, err := time.Parse("2006-01-02 15:04:05 -0700", firstCommit.DateTime)
			if err != nil {
				fmt.Printf("      ❌ Failed to parse commit time %s: %v\n", firstCommit.DateTime, err)
				continue
			}

			// Parse the day to get the actual date in the commit's timezone
			day := time.Date(firstCommitTime.Year(), firstCommitTime.Month(), firstCommitTime.Day(), 0, 0, 0, 0, firstCommitTime.Location())

			// Reverse commits so older commits get earlier times
			reversedCommits := make([]git.Commit, len(dayCommits))
			for i, commit := range dayCommits {
				reversedCommits[len(dayCommits)-1-i] = commit
			}

			// Generate new commit times for this specific day
			newTimes := generateCommitTimesForDay(day, len(reversedCommits))

			// Add to the collection for batch processing
			allCommits = append(allCommits, reversedCommits...)
			allNewTimes = append(allNewTimes, newTimes...)

			// Show what will be updated for this day
			for i, commit := range reversedCommits {
				newTime := newTimes[i]
				if commit.IsMerge {
					fmt.Printf("      • Will update merge %s: %s -> %s\n", commit.Hash, commit.DateTime, newTime.Format("2006-01-02 15:04:05"))
				} else {
					fmt.Printf("      • Will update %s: %s -> %s\n", commit.Hash, commit.DateTime, newTime.Format("2006-01-02 15:04:05"))
				}
			}
		}

		// Update all commits in a single operation
		repoUpdatedCount := 0
		if len(allCommits) > 0 {
			updatedCount, err := git.UpdateCommitTimes(repo, allCommits, allNewTimes, parentCommitHash, currentBranch, RewriteBranchName, NewCommitAuthorName, NewCommitAuthorEmail)
			if err != nil {
				fmt.Printf("   ❌ Failed to update commits: %v\n", err)
			} else {
				repoUpdatedCount = updatedCount
			}
		}

		if repoUpdatedCount > 0 {
			processedRepos++
			totalCommitsUpdated += repoUpdatedCount
			fmt.Printf("   ✅ Successfully updated %d commits total\n", repoUpdatedCount)
		}
	}

	fmt.Printf("\nSummary: Updated %d commits across %d repositories\n", totalCommitsUpdated, processedRepos)
}

// generateCommitTimesForDay creates evenly distributed times across work day for a specific day
func generateCommitTimesForDay(day time.Time, commitCount int) []time.Time {
	if commitCount <= 0 {
		return []time.Time{}
	}

	workDayStart := time.Date(day.Year(), day.Month(), day.Day(), WorkDayStartHour, 0, 0, 0, day.Location())
	workDayEnd := time.Date(day.Year(), day.Month(), day.Day(), WorkDayEndHour, 0, 0, 0, day.Location())
	workDayDuration := workDayEnd.Sub(workDayStart)

	times := make([]time.Time, commitCount)

	if commitCount == 1 {
		// Single commit goes closer to evening (7 PM)
		eveningTime := workDayEnd.Add(-time.Duration(rand.Intn(60)) * time.Minute) // Within 1 hour of end
		var jitter time.Duration
		if JitterMinutes > 0 {
			jitter = time.Duration(rand.Intn(JitterMinutes*2)-JitterMinutes) * time.Minute
		}
		times[0] = eveningTime.Add(jitter)
	} else {
		// Multiple commits distributed evenly
		interval := workDayDuration / time.Duration(commitCount-1)

		for i := 0; i < commitCount; i++ {
			baseTime := workDayStart.Add(time.Duration(i) * interval)
			var jitter time.Duration
			if JitterMinutes > 0 {
				jitter = time.Duration(rand.Intn(JitterMinutes*2)-JitterMinutes) * time.Minute
			}
			times[i] = baseTime.Add(jitter)
		}
	}

	// Ensure all times are within work hours
	for i, timeVal := range times {
		if timeVal.Before(workDayStart) {
			times[i] = workDayStart
		} else if timeVal.After(workDayEnd) || timeVal.Equal(workDayEnd) {
			times[i] = workDayEnd.Add(-time.Minute) // Just before end of work day
		}
	}

	// Sort times to ensure they're in chronological order
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	return times
}

// groupCommitsByDay groups commits by their date (YYYY-MM-DD format)
func groupCommitsByDay(commits []git.Commit) map[string][]git.Commit {
	commitsByDay := make(map[string][]git.Commit)

	for _, commit := range commits {
		// Parse the commit datetime in ISO format to extract the date
		commitTime, err := time.Parse("2006-01-02 15:04:05 -0700", commit.DateTime)
		if err != nil {
			// If parsing fails, use current date as fallback
			commitTime = time.Now()
		}

		dayStr := commitTime.Format("2006-01-02")
		commitsByDay[dayStr] = append(commitsByDay[dayStr], commit)
	}

	return commitsByDay
}

// parseWeekdays converts a CSV of weekday names/numbers to a set
// Accepts: "Sat,Sun", "Saturday, Sunday", "Mon", or digits 0-6 (0=Sunday)
func parseWeekdays(s string) map[time.Weekday]bool {
	m := make(map[time.Weekday]bool)
	if strings.TrimSpace(s) == "" {
		return m
	}
	items := strings.Split(s, ",")
	for _, raw := range items {
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		l := strings.ToLower(t)
		switch l {
		case "sun", "sunday", "0":
			m[time.Sunday] = true
		case "mon", "monday", "1":
			m[time.Monday] = true
		case "tue", "tues", "tuesday", "2":
			m[time.Tuesday] = true
		case "wed", "weds", "wednesday", "3":
			m[time.Wednesday] = true
		case "thu", "thur", "thurs", "thursday", "4":
			m[time.Thursday] = true
		case "fri", "friday", "5":
			m[time.Friday] = true
		case "sat", "saturday", "6":
			m[time.Saturday] = true
		}
	}
	return m
}

// enumerateDaysSkipping returns inclusive days [start..end], skipping any day whose Weekday() is in skip set.
func enumerateDaysSkipping(start, end time.Time, skip map[time.Weekday]bool) []time.Time {
	var days []time.Time
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		if skip != nil && skip[d.Weekday()] {
			continue
		}
		days = append(days, d)
	}
	return days
}

// allocateAcrossDays spreads n items across m buckets with specific positioning rules.
func allocateAcrossDays(n, m int) []int {
	if m <= 0 {
		return nil
	}
	out := make([]int, m)
	if n <= 0 {
		return out
	}

	// Special case: single commit goes to last day
	if n == 1 {
		out[m-1] = 1
		return out
	}

	// For multiple commits:
	// - First commit goes to first day
	// - Last commit goes to last day
	// - Middle commits are spread with jitter between first and last days

	// Special case: only one day available
	if m == 1 {
		out[0] = n
		return out
	}

	// Place first commit
	out[0] = 1

	// Place last commit
	out[m-1] = 1

	// Handle middle commits (n-2 remaining)
	if n > 2 {
		middleCommits := n - 2
		availableDays := m - 2 // Days between first and last (exclusive)

		if availableDays > 0 {
			// Add jitter by using random distribution
			for i := 0; i < middleCommits; i++ {
				var dayOffset int
				if JitterDays {
					// Use random jitter
					dayOffset = rand.Intn(availableDays)
				} else {
					// Use original deterministic distribution when no jitter
					dayOffset = (i*7 + i*i) % availableDays
				}
				dayIndex := 1 + dayOffset // Start from day 1 (after first day)
				out[dayIndex]++
			}
		} else {
			// If no middle days available, distribute between first and last
			for i := 0; i < middleCommits; i++ {
				if i%2 == 0 {
					out[0]++ // Even indices go to first day
				} else {
					out[m-1]++ // Odd indices go to last day
				}
			}
		}
	}

	return out
}

// createBackup creates a timestamped backup of a directory
func createBackup(sourcePath string) (string, error) {
	// Generate timestamp for backup folder name
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	backupPath := fmt.Sprintf("%s%s%s", sourcePath, BackupFolderPattern, timestamp)

	// Use cp command to copy the directory recursively
	cmd := exec.Command("cp", "-r", sourcePath, backupPath)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create backup of %s: %v\nstdout: %s\nstderr: %s", sourcePath, err, stdout.String(), stderr.String())
	}

	return backupPath, nil
}

// createBackupsForRepos creates backups for all repositories if backup is enabled
func createBackupsForRepos(gitRepos []string) error {
	if !CreateBackup {
		return nil // Backup is disabled
	}

	fmt.Println("Creating backups of repositories...")
	backupCount := 0

	for _, repo := range gitRepos {
		backupPath, err := createBackup(repo)
		if err != nil {
			fmt.Printf("Warning: Failed to create backup for %s: %v\n", repo, err)
			continue
		}
		backupCount++
		fmt.Printf("✓ Created backup: %s\n", backupPath)
	}

	if backupCount > 0 {
		fmt.Printf("Successfully created %d backups\n", backupCount)
	}

	return nil
}

// commitCadenceSpan redistributes unpushed commit times across all days from oldest unpushed commit through today.
// It skips weekdays configured via SKIP_WEEK_DAYS and keeps commits within work hours.
func commitCadenceSpan(gitRepos []string) {
	fmt.Println("Redistributing unpushed commit times across all days since last push...")

	// Create backups if enabled
	if err := createBackupsForRepos(gitRepos); err != nil {
		fmt.Printf("Warning: Failed to create backups: %v\n", err)
	}

	fmt.Println()

	processedRepos := 0
	totalCommitsUpdated := 0

	now := time.Now()

	for _, repo := range gitRepos {
		// Skip backup folders
		if isBackupFolder(repo) {
			fmt.Printf("⏭️  Skipping backup folder: %s\n", repo)
			continue
		}

		unpushedCommits, err := git.GetUnpushedCommits(repo, ParentGitBranchName)
		if err != nil {
			fmt.Printf("Warning: Could not check commits for %s: %v\n", repo, err)
			continue
		}
		if len(unpushedCommits) == 0 {
			fmt.Printf("✅ %s: No unpushed commits to redistribute\n", repo)
			continue
		}

		fmt.Printf("\n📦 %s (%d unpushed commits):\n", repo, len(unpushedCommits))

		currentBranch, err := git.GetCurrentBranch(repo)
		if err != nil {
			fmt.Printf("   ❌ Error: Could not get current branch for %s: %v\n", repo, err)
			continue
		}
		fmt.Printf("   🌿 Current branch: %s\n", currentBranch)

		oldestUnpushed := unpushedCommits[len(unpushedCommits)-1]
		parentCommitHash, err := git.GetParentCommit(repo, oldestUnpushed.Hash)
		if err != nil {
			// If this is the first commit in the repository, use empty tree as parent
			fmt.Printf("   ⚠️  First commit in repository, using empty tree as parent\n")
			parentCommitHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904" // Empty tree hash
		} else {
			fmt.Printf("   📍 Parent commit: %s\n", parentCommitHash)
		}

		oldestTime, err := time.Parse("2006-01-02 15:04:05 -0700", oldestUnpushed.DateTime)
		if err != nil {
			fmt.Printf("   ❌ Failed to parse oldest commit time %s: %v\n", oldestUnpushed.DateTime, err)
			continue
		}
		loc := oldestTime.Location()

		startDay := time.Date(oldestTime.Year(), oldestTime.Month(), oldestTime.Day(), 0, 0, 0, 0, loc)
		today := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), 0, 0, 0, 0, loc)

		// Build list of eligible days [startDay..today], skipping configured weekdays
		days := enumerateDaysSkipping(startDay, today, skipWeekdaysSet)
		if len(days) == 0 {
			fmt.Printf("   ⚠️ No eligible days in range after applying SKIP_WEEK_DAYS=%q\n", SkipWeekDays)
			continue
		}

		// Order commits oldest -> newest for allocation
		ordered := make([]git.Commit, len(unpushedCommits))
		for i := range unpushedCommits {
			ordered[i] = unpushedCommits[len(unpushedCommits)-1-i]
		}

		alloc := allocateAcrossDays(len(ordered), len(days))

		var allCommits []git.Commit
		var allNewTimes []time.Time

		cursor := 0
		for i, day := range days {
			k := alloc[i]
			if k == 0 {
				continue
			}
			sub := ordered[cursor : cursor+k]
			cursor += k

			newTimes := generateCommitTimesForDay(day, len(sub))

			fmt.Printf("   📅 %s (%d commits):\n", day.Format("2006-01-02"), len(sub))
			for j := range sub {
				if sub[j].IsMerge {
					fmt.Printf("      • Will update merge %s: %s -> %s\n",
						sub[j].Hash,
						sub[j].DateTime,
						newTimes[j].Format("2006-01-02 15:04:05"),
					)
				} else {
					fmt.Printf("      • Will update %s: %s -> %s\n",
						sub[j].Hash,
						sub[j].DateTime,
						newTimes[j].Format("2006-01-02 15:04:05"),
					)
				}
			}

			allCommits = append(allCommits, sub...)
			allNewTimes = append(allNewTimes, newTimes...)
		}

		if len(allCommits) != len(allNewTimes) || len(allCommits) == 0 {
			fmt.Printf("   ❌ Internal error: mismatched allocation (commits=%d times=%d)\n", len(allCommits), len(allNewTimes))
			continue
		}

		updatedCount, err := git.UpdateCommitTimes(repo, allCommits, allNewTimes, parentCommitHash, currentBranch, RewriteBranchName, NewCommitAuthorName, NewCommitAuthorEmail)
		if err != nil {
			fmt.Printf("   ❌ Failed to update commits: %v\n", err)
			continue
		}

		if updatedCount > 0 {
			processedRepos++
			totalCommitsUpdated += updatedCount
			fmt.Printf("   ✅ Successfully updated %d commits total\n", updatedCount)
		}
	}

	fmt.Printf("\nSummary: Updated %d commits across %d repositories\n", totalCommitsUpdated, processedRepos)
}
