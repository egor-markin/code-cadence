# Code Cadence

A tool for redistributing Git commit timestamps to make your work appear to happen during designated business hours.

## Motivation

Sometimes you work outside your employer's expected hours - late nights, early mornings, or weekends - but your work day is officially 9 AM to 6 PM. Modern companies use Git commit analysis tools to track employee productivity and efficiency. Working outside designated hours can create tensions with your employer.

Code Cadence solves this problem by allowing you to continue working during your most productive hours while making it appear to your employer that you worked during expected business hours.

## How It Works

Because of how Git works, you can only rewrite the history of commits that haven't been pushed yet. Not following this rule can result in serious issues.

Code Cadence looks for all unpushed commits in the current Git branch and spreads them evenly across the time period from the last pushed commit to the current moment. It also distributes commits within work days to make it look like you worked during designated hours.

## Commands

### Main Commands

- **`commit_cadence`** - Keeps all commits within their original day and only spreads them more evenly across the day
- **`commit_cadence_span`** - May move commits across days while keeping their chronological order and spreading them evenly across the provided time period

In most real-world cases, `commit_cadence_span` will be the preferred command.

### Push Management Commands

Since it's crucial to keep commits unpushed to update their timestamps later, Code Cadence provides push management commands:

- **`push_disable`** - Blocks the push command for a Git repository using a pre-push Git hook
- **`push_enable`** - Unblocks the push command by removing the pre-push Git hook
- **`push_status`** - Returns the push block status for a Git repository

### Workflow

1. Disable pushes for your Git repo before starting work to prevent accidental pushes
2. Work normally and make commits
3. Run `commit_cadence` or `commit_cadence_span` to redistribute commit timestamps
4. If satisfied with the results, enable pushes and push your changes
5. Disable pushes again for future work

### Safety Features

- It's safe to call `commit_cadence` and `commit_cadence_span` multiple times - each call creates a different random distribution
- All commands are recursive and work on single repos or entire workspace folders
- Built-in backup system (enabled by default) creates copies before modifying repositories

## Usage

All commands are recursive and can be called on a single Git repository or a folder containing multiple repositories.

### Command Examples

```bash
# Disable pushes for all repos in workspace
code-cadence push_disable /home/john/workspace/

# Check push status for all repos
code-cadence push_status /home/john/workspace/

# View unpushed commits
code-cadence commit_status /home/john/workspace/

# Redistribute commits within their original days
code-cadence commit_cadence /home/john/workspace/

# Redistribute commits across the entire time span
code-cadence commit_cadence_span /home/john/workspace/

# Re-enable pushes
code-cadence push_enable /home/john/workspace/
```

## Configuration

Code Cadence can be configured using a `.env` file. Copy `env.example` to `.env` and modify the values as needed.

### Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `WORK_DAY_START_HOUR` | Earliest hour for commits (24-hour format) | 10 |
| `WORK_DAY_END_HOUR` | Latest hour for commits (24-hour format) | 19 |
| `JITTER_MINUTES` | Random minutes to add/subtract from commit times | 30 |
| `JITTER_DAYS` | Enable random day jitter for more natural distribution | true |
| `PARENT_GIT_BRANCH_NAME` | Main branch name (e.g., "origin/main") | origin/main |
| `NEW_COMMIT_AUTHOR_NAME` | Override author name (optional) | (preserve original) |
| `NEW_COMMIT_AUTHOR_EMAIL` | Override author email (optional) | (preserve original) |
| `SKIP_WEEK_DAYS` | Days to skip (comma-separated: Sat,Sun) | Sat,Sun |
| `CREATE_BACKUP` | Create backups before modifying repos | true |

### Configuration File Locations

Code Cadence looks for `.env` files in this order:
1. Current directory
2. `~/.config/code-cadence/.env`
3. `/opt/code-cadence/.env`
4. `/usr/local/etc/code-cadence/.env`

## Installation

### Prerequisites

- Git command-line tool installed and available in your system PATH. Refer to the GIT's official [installation documentation](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).
- Go compiler version 1.25 or above (for building from source)

### Installation Options

1. **Download pre-compiled binary** from the releases section
2. **Build from source** using Go compiler

### Recommended Installation Locations

Place the executable in one of these directories (in order of preference):
- `/usr/local/bin/` (included in PATH by default)
- `~/bin/` (add to PATH manually)
- `/opt/code-cadence/` (add to PATH manually)

### Adding to PATH

If you install to `~/bin/` or `/opt/code-cadence/`, add to your shell profile:

```bash
# For ~/bin/
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc  # (for Linux) or ~/.zshrc (for macOS)

# For /opt/code-cadence/
echo 'export PATH="/opt/code-cadence:$PATH"' >> ~/.bashrc  # (or Linux) or ~/.zshrc (for macOS)
```

## Important Notes

⚠️ **Backup Recommendation**: Always create backups of Git repositories before using this tool. While backups are enabled by default, it's still recommended to create manual backups for critical repositories.

⚠️ **Beta Software**: This tool is in beta. There's no guarantee it won't cause issues with your data.

⚠️ **Unpushed Commits Only**: This tool only works with unpushed commits. Once commits are pushed, their timestamps cannot be safely modified.

## License

Licensed under [CC BY-NC 4.0](https://creativecommons.org/licenses/by-nc/4.0/) - free for non-commercial use with attribution.

## Contributing

Merge requests are welcome! If you feel the app can be improved in any way, please don't hesitate to submit a pull request. Whether it's bug fixes, new features, documentation improvements, or performance optimizations, all contributions are appreciated.