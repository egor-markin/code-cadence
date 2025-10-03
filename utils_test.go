package main

import (
	"testing"
	"time"

	"code-cadence/git"
)

func TestParseWeekdays(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[time.Weekday]bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[time.Weekday]bool{},
		},
		{
			name:     "single day abbreviation",
			input:    "Mon",
			expected: map[time.Weekday]bool{time.Monday: true},
		},
		{
			name:     "single day full name",
			input:    "Monday",
			expected: map[time.Weekday]bool{time.Monday: true},
		},
		{
			name:     "single day numeric",
			input:    "1",
			expected: map[time.Weekday]bool{time.Monday: true},
		},
		{
			name:  "multiple days with commas",
			input: "Sat,Sun",
			expected: map[time.Weekday]bool{
				time.Saturday: true,
				time.Sunday:   true,
			},
		},
		{
			name:  "multiple days with spaces",
			input: "Saturday, Sunday",
			expected: map[time.Weekday]bool{
				time.Saturday: true,
				time.Sunday:   true,
			},
		},
		{
			name:  "numeric format",
			input: "0,6",
			expected: map[time.Weekday]bool{
				time.Sunday:   true,
				time.Saturday: true,
			},
		},
		{
			name:  "mixed format",
			input: "Mon,2,Wednesday",
			expected: map[time.Weekday]bool{
				time.Monday:    true,
				time.Tuesday:   true,
				time.Wednesday: true,
			},
		},
		{
			name:  "case insensitive",
			input: "MONDAY,tue,FRIDAY",
			expected: map[time.Weekday]bool{
				time.Monday:  true,
				time.Tuesday: true,
				time.Friday:  true,
			},
		},
		{
			name:     "invalid days mixed with valid",
			input:    "InvalidDay,Mon,AnotherInvalid",
			expected: map[time.Weekday]bool{time.Monday: true},
		},
		{
			name:     "all invalid days",
			input:    "InvalidDay,AnotherInvalid",
			expected: map[time.Weekday]bool{},
		},
		{
			name:  "whitespace handling",
			input: " Mon , Tue , Wed ",
			expected: map[time.Weekday]bool{
				time.Monday:    true,
				time.Tuesday:   true,
				time.Wednesday: true,
			},
		},
		{
			name:  "empty elements",
			input: "Mon,,Tue,",
			expected: map[time.Weekday]bool{
				time.Monday:  true,
				time.Tuesday: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseWeekdays(test.input)

			if len(result) != len(test.expected) {
				t.Errorf("Expected %d weekdays, got %d", len(test.expected), len(result))
			}

			for weekday, expected := range test.expected {
				if result[weekday] != expected {
					t.Errorf("Expected %v to be %t, got %t", weekday, expected, result[weekday])
				}
			}

			// Verify no unexpected weekdays are set
			for weekday, value := range result {
				if expected, exists := test.expected[weekday]; !exists || !expected {
					t.Errorf("Unexpected weekday %v set to %t", weekday, value)
				}
			}
		})
	}
}

func TestEnumerateDaysSkipping(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		skip     map[time.Weekday]bool
		expected int
	}{
		{
			name:     "no skip days",
			start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			skip:     nil,
			expected: 3,
		},
		{
			name:     "skip weekends",
			start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // Monday
			end:      time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC), // Sunday
			skip:     map[time.Weekday]bool{time.Saturday: true, time.Sunday: true},
			expected: 5, // Mon-Fri
		},
		{
			name:  "skip weekdays",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // Monday
			end:   time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC), // Sunday
			skip: map[time.Weekday]bool{
				time.Monday: true, time.Tuesday: true, time.Wednesday: true,
				time.Thursday: true, time.Friday: true,
			},
			expected: 2, // Sat-Sun
		},
		{
			name:  "skip all days",
			start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC),
			skip: map[time.Weekday]bool{
				time.Sunday: true, time.Monday: true, time.Tuesday: true,
				time.Wednesday: true, time.Thursday: true, time.Friday: true, time.Saturday: true,
			},
			expected: 0,
		},
		{
			name:     "single day range",
			start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			skip:     nil,
			expected: 1,
		},
		{
			name:     "single day range with skip",
			start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // Monday
			end:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // Monday
			skip:     map[time.Weekday]bool{time.Monday: true},
			expected: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := enumerateDaysSkipping(test.start, test.end, test.skip)

			if len(result) != test.expected {
				t.Errorf("Expected %d days, got %d", test.expected, len(result))
			}

			// Verify no skipped days are included
			for _, day := range result {
				if test.skip != nil && test.skip[day.Weekday()] {
					t.Errorf("Unexpected skipped day %v included", day.Weekday())
				}
			}

			// Verify days are in chronological order
			for i := 1; i < len(result); i++ {
				if result[i].Before(result[i-1]) {
					t.Errorf("Days are not in chronological order: %s before %s",
						result[i-1].Format("2006-01-02"), result[i].Format("2006-01-02"))
				}
			}
		})
	}
}

func TestAllocateAcrossDays(t *testing.T) {
	// Test deterministic behavior (no jitter)
	originalJitterDays := JitterDays
	JitterDays = false
	defer func() { JitterDays = originalJitterDays }()

	tests := []struct {
		name     string
		n        int
		m        int
		expected []int
	}{
		{
			name:     "zero items",
			n:        0,
			m:        3,
			expected: []int{0, 0, 0},
		},
		{
			name:     "zero days",
			n:        5,
			m:        0,
			expected: nil,
		},
		{
			name:     "single item",
			n:        1,
			m:        1,
			expected: []int{1},
		},
		{
			name:     "single item multiple days",
			n:        1,
			m:        3,
			expected: []int{0, 0, 1},
		},
		{
			name:     "exact division",
			n:        6,
			m:        3,
			expected: []int{1, 4, 1},
		},
		{
			name:     "remainder 1",
			n:        7,
			m:        3,
			expected: []int{1, 5, 1},
		},
		{
			name:     "remainder 2",
			n:        8,
			m:        3,
			expected: []int{1, 6, 1},
		},
		{
			name:     "large remainder",
			n:        10,
			m:        3,
			expected: []int{1, 8, 1},
		},
		{
			name:     "more items than days",
			n:        15,
			m:        4,
			expected: []int{1, 13, 0, 1},
		},
		{
			name:     "fewer items than days",
			n:        2,
			m:        5,
			expected: []int{1, 0, 0, 0, 1},
		},
		{
			name:     "single day",
			n:        5,
			m:        1,
			expected: []int{5},
		},
		{
			name:     "two days",
			n:        5,
			m:        2,
			expected: []int{3, 2},
		},
		{
			name:     "spacing example - 3 items across 7 days",
			n:        3,
			m:        7,
			expected: []int{1, 1, 0, 0, 0, 0, 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := allocateAcrossDays(test.n, test.m)

			if test.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}

			if len(result) != len(test.expected) {
				t.Errorf("Expected length %d, got %d", len(test.expected), len(result))
				return
			}

			// Verify individual values
			for i, expected := range test.expected {
				if result[i] != expected {
					t.Errorf("Expected result[%d] = %d, got %d", i, expected, result[i])
				}
			}

			// Verify total sum
			total := 0
			for _, val := range result {
				total += val
			}
			if total != test.n {
				t.Errorf("Expected total %d, got %d", test.n, total)
			}

			// Verify first and last positioning rules
			if test.n == 1 && test.m > 1 {
				// Single commit should be in last day
				if result[test.m-1] != 1 {
					t.Errorf("Single commit should be in last day, got %v", result)
				}
			} else if test.n > 1 && test.m > 1 {
				// Multiple commits: first in first day, last in last day
				if result[0] == 0 {
					t.Errorf("First commit should be in first day, got %v", result)
				}
				if result[test.m-1] == 0 {
					t.Errorf("Last commit should be in last day, got %v", result)
				}
			}
		})
	}
}

func TestAllocateAcrossDaysWithJitter(t *testing.T) {
	// Test jitter behavior
	originalJitterDays := JitterDays
	JitterDays = true // Enable jitter
	defer func() { JitterDays = originalJitterDays }()

	tests := []struct {
		name string
		n    int
		m    int
	}{
		{
			name: "multiple items with jitter",
			n:    5,
			m:    7,
		},
		{
			name: "many items with jitter",
			n:    10,
			m:    5,
		},
		{
			name: "few items with jitter",
			n:    3,
			m:    10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Run multiple times to test randomness
			results := make([][]int, 10)
			for i := 0; i < 10; i++ {
				results[i] = allocateAcrossDays(test.n, test.m)
			}

			// Verify all results have correct properties
			for i, result := range results {
				// Verify total sum
				total := 0
				for _, val := range result {
					total += val
				}
				if total != test.n {
					t.Errorf("Iteration %d: Expected total %d, got %d", i, test.n, total)
				}

				// Verify length
				if len(result) != test.m {
					t.Errorf("Iteration %d: Expected length %d, got %d", i, test.m, len(result))
				}

				// Verify first and last positioning rules
				if test.n == 1 && test.m > 1 {
					// Single commit should be in last day
					if result[test.m-1] != 1 {
						t.Errorf("Iteration %d: Single commit should be in last day, got %v", i, result)
					}
				} else if test.n > 1 && test.m > 1 {
					// Multiple commits: first in first day, last in last day
					if result[0] == 0 {
						t.Errorf("Iteration %d: First commit should be in first day, got %v", i, result)
					}
					if result[test.m-1] == 0 {
						t.Errorf("Iteration %d: Last commit should be in last day, got %v", i, result)
					}
				}
			}

			// Check that results are different (jitter is working)
			// This is probabilistic, but with 10 iterations it should be very likely
			allSame := true
			for i := 1; i < len(results); i++ {
				if !slicesEqual(results[0], results[i]) {
					allSame = false
					break
				}
			}
			if allSame && test.n > 2 && test.m > 2 {
				t.Logf("Warning: All 10 iterations produced identical results for %s. This might indicate jitter is not working properly.", test.name)
			}
		})
	}
}

func slicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGroupCommitsByDay(t *testing.T) {
	tests := []struct {
		name     string
		commits  []git.Commit
		expected map[string]int
	}{
		{
			name:     "empty commits",
			commits:  []git.Commit{},
			expected: map[string]int{},
		},
		{
			name: "single commit",
			commits: []git.Commit{
				{Hash: "abc123", DateTime: "2024-01-01 10:00:00 +0000"},
			},
			expected: map[string]int{"2024-01-01": 1},
		},
		{
			name: "multiple commits same day",
			commits: []git.Commit{
				{Hash: "abc123", DateTime: "2024-01-01 10:00:00 +0000"},
				{Hash: "def456", DateTime: "2024-01-01 14:00:00 +0000"},
				{Hash: "ghi789", DateTime: "2024-01-01 16:00:00 +0000"},
			},
			expected: map[string]int{"2024-01-01": 3},
		},
		{
			name: "multiple commits different days",
			commits: []git.Commit{
				{Hash: "abc123", DateTime: "2024-01-01 10:00:00 +0000"},
				{Hash: "def456", DateTime: "2024-01-02 14:00:00 +0000"},
				{Hash: "ghi789", DateTime: "2024-01-03 16:00:00 +0000"},
			},
			expected: map[string]int{
				"2024-01-01": 1,
				"2024-01-02": 1,
				"2024-01-03": 1,
			},
		},
		{
			name: "mixed commits",
			commits: []git.Commit{
				{Hash: "abc123", DateTime: "2024-01-01 10:00:00 +0000"},
				{Hash: "def456", DateTime: "2024-01-01 14:00:00 +0000"},
				{Hash: "ghi789", DateTime: "2024-01-02 16:00:00 +0000"},
				{Hash: "jkl012", DateTime: "2024-01-02 18:00:00 +0000"},
				{Hash: "mno345", DateTime: "2024-01-02 20:00:00 +0000"},
			},
			expected: map[string]int{
				"2024-01-01": 2,
				"2024-01-02": 3,
			},
		},
		{
			name: "invalid datetime format",
			commits: []git.Commit{
				{Hash: "abc123", DateTime: "invalid datetime"},
				{Hash: "def456", DateTime: "2024-01-01 14:00:00 +0000"},
			},
			expected: map[string]int{
				"2024-01-01": 1,
			},
		},
		{
			name: "different timezones",
			commits: []git.Commit{
				{Hash: "abc123", DateTime: "2024-01-01 10:00:00 +0000"},
				{Hash: "def456", DateTime: "2024-01-01 10:00:00 +0100"},
				{Hash: "ghi789", DateTime: "2024-01-01 10:00:00 -0500"},
			},
			expected: map[string]int{
				"2024-01-01": 3,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := groupCommitsByDay(test.commits)

			// Special handling for invalid datetime format test
			if test.name == "invalid datetime format" {
				// Should have at least 1 day (2024-01-01) and possibly more for invalid datetime
				if len(result) < 1 {
					t.Errorf("Expected at least 1 day, got %d", len(result))
				}
				// Check that 2024-01-01 has 1 commit
				if len(result["2024-01-01"]) != 1 {
					t.Errorf("Expected 1 commit on 2024-01-01, got %d", len(result["2024-01-01"]))
				}
			} else {
				if len(result) != len(test.expected) {
					t.Errorf("Expected %d days, got %d", len(test.expected), len(result))
				}

				for day, expectedCount := range test.expected {
					if len(result[day]) != expectedCount {
						t.Errorf("Expected %d commits on %s, got %d", expectedCount, day, len(result[day]))
					}
				}
			}

			// Verify total commit count
			totalCommits := 0
			for _, commits := range result {
				totalCommits += len(commits)
			}
			if totalCommits != len(test.commits) {
				t.Errorf("Expected total %d commits, got %d", len(test.commits), totalCommits)
			}
		})
	}
}

func TestGenerateCommitTimesForDay(t *testing.T) {
	// Set up test configuration
	WorkDayStartHour = 9
	WorkDayEndHour = 17
	JitterMinutes = 0 // Disable jitter for predictable testing

	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		commitCount int
		expected    int
	}{
		{
			name:        "zero commits",
			commitCount: 0,
			expected:    0,
		},
		{
			name:        "single commit",
			commitCount: 1,
			expected:    1,
		},
		{
			name:        "multiple commits",
			commitCount: 3,
			expected:    3,
		},
		{
			name:        "many commits",
			commitCount: 10,
			expected:    10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := generateCommitTimesForDay(day, test.commitCount, nil)

			if len(result) != test.expected {
				t.Errorf("Expected %d times, got %d", test.expected, len(result))
			}

			// Verify times are within work hours
			for i, timeVal := range result {
				hour := timeVal.Hour()
				if hour < WorkDayStartHour || hour >= WorkDayEndHour {
					t.Errorf("Time %d (%s) is outside work hours (%d-%d)",
						i, timeVal.Format("15:04"), WorkDayStartHour, WorkDayEndHour)
				}
			}

			// Verify times are in ascending order
			for i := 1; i < len(result); i++ {
				if result[i].Before(result[i-1]) {
					t.Errorf("Times are not in ascending order: %s before %s",
						result[i-1].Format("15:04"), result[i].Format("15:04"))
				}
			}

			// Verify times are on the correct day
			for i, timeVal := range result {
				if timeVal.Year() != day.Year() || timeVal.Month() != day.Month() || timeVal.Day() != day.Day() {
					t.Errorf("Time %d (%s) is not on the correct day (%s)",
						i, timeVal.Format("2006-01-02"), day.Format("2006-01-02"))
				}
			}
		})
	}
}

func TestGenerateCommitTimesForDayWithJitter(t *testing.T) {
	// Set up test configuration with jitter
	WorkDayStartHour = 9
	WorkDayEndHour = 17
	JitterMinutes = 30

	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test with jitter enabled
	result := generateCommitTimesForDay(day, 3, nil)

	if len(result) != 3 {
		t.Errorf("Expected 3 times, got %d", len(result))
	}

	// Verify times are still within work hours (with jitter tolerance)
	for i, timeVal := range result {
		hour := timeVal.Hour()
		minute := timeVal.Minute()

		// Allow for jitter - times should be within work hours plus/minus jitter
		if hour < WorkDayStartHour-1 || hour >= WorkDayEndHour+1 {
			t.Errorf("Time %d (%s) is outside work hours with jitter tolerance (%d-%d)",
				i, timeVal.Format("15:04"), WorkDayStartHour, WorkDayEndHour)
		}

		// If it's at the boundary, check minutes
		if hour == WorkDayStartHour-1 && minute < 30 {
			t.Errorf("Time %d (%s) is too early with jitter", i, timeVal.Format("15:04"))
		}
		if hour == WorkDayEndHour && minute > 30 {
			t.Errorf("Time %d (%s) is too late with jitter", i, timeVal.Format("15:04"))
		}
	}
}

func TestGenerateCommitTimesForDayEdgeCases(t *testing.T) {
	// Set up test configuration
	WorkDayStartHour = 9
	WorkDayEndHour = 17
	JitterMinutes = 0

	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test with very short work day
	WorkDayStartHour = 12
	WorkDayEndHour = 13

	result := generateCommitTimesForDay(day, 2, nil)

	if len(result) != 2 {
		t.Errorf("Expected 2 times, got %d", len(result))
	}

	// Verify times are within the short work day
	for i, timeVal := range result {
		hour := timeVal.Hour()
		if hour < WorkDayStartHour || hour >= WorkDayEndHour {
			t.Errorf("Time %d (%s) is outside short work hours (%d-%d)",
				i, timeVal.Format("15:04"), WorkDayStartHour, WorkDayEndHour)
		}
	}

	// Test with same start and end hour
	WorkDayStartHour = 12
	WorkDayEndHour = 12

	result = generateCommitTimesForDay(day, 1, nil)

	if len(result) != 1 {
		t.Errorf("Expected 1 time, got %d", len(result))
	}

	// The time should be at the start hour
	if result[0].Hour() != WorkDayStartHour {
		t.Errorf("Expected time to be at hour %d, got %d", WorkDayStartHour, result[0].Hour())
	}
}

// Benchmark tests
func BenchmarkParseWeekdays(b *testing.B) {
	input := "Mon,Tue,Wed,Thu,Fri,Sat,Sun"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseWeekdays(input)
	}
}

func BenchmarkEnumerateDaysSkipping(b *testing.B) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	skip := map[time.Weekday]bool{time.Saturday: true, time.Sunday: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enumerateDaysSkipping(start, end, skip)
	}
}

func BenchmarkAllocateAcrossDays(b *testing.B) {
	n, m := 1000, 30

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		allocateAcrossDays(n, m)
	}
}

func BenchmarkGroupCommitsByDay(b *testing.B) {
	commits := []git.Commit{
		{Hash: "abc123", DateTime: "2024-01-01 10:00:00 +0000"},
		{Hash: "def456", DateTime: "2024-01-01 14:00:00 +0000"},
		{Hash: "ghi789", DateTime: "2024-01-02 16:00:00 +0000"},
		{Hash: "jkl012", DateTime: "2024-01-02 18:00:00 +0000"},
		{Hash: "mno345", DateTime: "2024-01-03 20:00:00 +0000"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		groupCommitsByDay(commits)
	}
}

func BenchmarkGenerateCommitTimesForDay(b *testing.B) {
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	commitCount := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateCommitTimesForDay(day, commitCount, nil)
	}
}
