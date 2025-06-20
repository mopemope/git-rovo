package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mopemope/git-rovo/internal/logger"
)

// Repository represents a Git repository
type Repository struct {
	workDir string
}

// FileStatus represents the status of a file in Git
type FileStatus struct {
	Path     string
	Status   string // M, A, D, R, C, U, ?
	Staged   bool
	Modified bool
}

// CommitInfo represents information about a Git commit
type CommitInfo struct {
	Hash      string
	Author    string
	Date      time.Time
	Subject   string
	Body      string
	ShortHash string
}

// DiffInfo represents diff information for files
type DiffInfo struct {
	FilePath  string
	OldPath   string
	NewPath   string
	Status    string // A, M, D, R, C
	Additions int
	Deletions int
	Content   string
	IsBinary  bool
}

// New creates a new Git repository instance
func New(workDir string) (*Repository, error) {
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Check if it's a Git repository
	if !IsGitRepository(workDir) {
		return nil, fmt.Errorf("not a git repository: %s", workDir)
	}

	return &Repository{
		workDir: workDir,
	}, nil
}

// IsGitRepository checks if the directory is a Git repository
func IsGitRepository(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	if stat, err := os.Stat(gitDir); err == nil {
		return stat.IsDir()
	}

	// Check if it's a worktree (has .git file)
	gitFile := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitFile); err == nil {
		return true
	}

	return false
}

// GetWorkDir returns the working directory
func (r *Repository) GetWorkDir() string {
	return r.workDir
}

// runGitCommand executes a Git command and returns the output
func (r *Repository) runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.workDir

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Only trim trailing newline, preserve leading spaces for porcelain format
	if len(outputStr) > 0 && outputStr[len(outputStr)-1] == '\n' {
		outputStr = outputStr[:len(outputStr)-1]
	}

	// Log the Git operation
	logger.LogGitOperation("git", args, r.workDir, err == nil, outputStr, err)

	if err != nil {
		return outputStr, fmt.Errorf("git command failed: %w\nOutput: %s", err, outputStr)
	}

	return outputStr, nil
}

// GetStatus returns the status of files in the repository
func (r *Repository) GetStatus() ([]FileStatus, error) {
	output, err := r.runGitCommand("status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}

	var files []FileStatus
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 3 {
			continue
		}

		status := line[:2]
		path := line[3:]

		// Git porcelain format:
		// First character: staged changes (index)
		// Second character: unstaged changes (working tree)
		stagedChar := status[0]
		unstagedChar := status[1]

		file := FileStatus{
			Path:     path,
			Status:   status,
			Staged:   stagedChar != ' ' && stagedChar != '?',
			Modified: unstagedChar != ' ' && status != "??", // Don't mark untracked files as modified
		}

		files = append(files, file)
	}

	return files, scanner.Err()
}

// GetDiff returns the diff for staged or unstaged changes
func (r *Repository) GetDiff(staged bool, filePaths ...string) ([]DiffInfo, error) {
	args := []string{"diff", "--no-color"}
	if staged {
		args = append(args, "--cached")
	}

	// Add specific file paths if provided
	if len(filePaths) > 0 {
		args = append(args, "--")
		args = append(args, filePaths...)
	}

	output, err := r.runGitCommand(args...)
	if err != nil {
		return nil, err
	}

	return r.parseDiff(output), nil
}

// GetUntrackedFileDiff returns the content of untracked files as diff format
func (r *Repository) GetUntrackedFileDiff(filePaths ...string) ([]DiffInfo, error) {
	var diffs []DiffInfo

	// Get status to identify untracked files
	status, err := r.GetStatus()
	if err != nil {
		return nil, err
	}

	// Create a map of untracked files for quick lookup
	untrackedFiles := make(map[string]bool)
	for _, file := range status {
		if file.Status == "??" {
			untrackedFiles[file.Path] = true
		}
	}

	// If specific files are requested, filter them
	var targetFiles []string
	if len(filePaths) > 0 {
		for _, path := range filePaths {
			if untrackedFiles[path] {
				targetFiles = append(targetFiles, path)
			}
		}
	} else {
		// Get all untracked files
		for path := range untrackedFiles {
			targetFiles = append(targetFiles, path)
		}
	}

	// Generate diff for each untracked file
	for _, filePath := range targetFiles {
		diff, err := r.generateUntrackedFileDiff(filePath)
		if err != nil {
			// Skip files that can't be read (e.g., binary files, permission issues)
			continue
		}
		if diff != nil {
			diffs = append(diffs, *diff)
		}
	}

	return diffs, nil
}

// generateUntrackedFileDiff creates a diff-like representation for an untracked file
func (r *Repository) generateUntrackedFileDiff(filePath string) (*DiffInfo, error) {
	fullPath := filepath.Join(r.workDir, filePath)

	// Check if file exists and is readable
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	// Skip directories
	if fileInfo.IsDir() {
		return nil, nil
	}

	// Skip very large files (> 1MB)
	if fileInfo.Size() > 1024*1024 {
		return &DiffInfo{
			FilePath: filePath,
			OldPath:  "",
			Status:   "A",
			IsBinary: true,
			Content:  fmt.Sprintf("Binary file %s (size: %d bytes)", filePath, fileInfo.Size()),
		}, nil
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	// Check if file is binary
	if isBinaryContent(content) {
		return &DiffInfo{
			FilePath: filePath,
			OldPath:  "",
			Status:   "A",
			IsBinary: true,
			Content:  fmt.Sprintf("Binary file %s", filePath),
		}, nil
	}

	// Generate diff-like content for text files
	lines := strings.Split(string(content), "\n")
	var diffContent strings.Builder

	diffContent.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
	diffContent.WriteString("new file mode 100644\n")
	diffContent.WriteString("index 0000000..0000000\n")
	diffContent.WriteString("--- /dev/null\n")
	diffContent.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
	diffContent.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))

	for _, line := range lines {
		diffContent.WriteString(fmt.Sprintf("+%s\n", line))
	}

	return &DiffInfo{
		FilePath: filePath,
		OldPath:  "",
		Status:   "A",
		IsBinary: false,
		Content:  diffContent.String(),
	}, nil
}

// isBinaryContent checks if content appears to be binary
func isBinaryContent(content []byte) bool {
	// Simple heuristic: if content contains null bytes, consider it binary
	for _, b := range content {
		if b == 0 {
			return true
		}
	}

	// Check for high ratio of non-printable characters
	nonPrintable := 0
	for _, b := range content {
		if b < 32 && b != 9 && b != 10 && b != 13 { // Allow tab, LF, CR
			nonPrintable++
		}
	}

	// If more than 30% non-printable, consider binary
	if len(content) > 0 && float64(nonPrintable)/float64(len(content)) > 0.3 {
		return true
	}

	return false
}

// parseDiff parses git diff output into DiffInfo structs
func (r *Repository) parseDiff(diffOutput string) []DiffInfo {
	var diffs []DiffInfo
	var currentDiff *DiffInfo

	scanner := bufio.NewScanner(strings.NewReader(diffOutput))
	var contentLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "diff --git") {
			// Save previous diff if exists
			if currentDiff != nil {
				currentDiff.Content = strings.Join(contentLines, "\n")
				diffs = append(diffs, *currentDiff)
				contentLines = nil
			}

			// Start new diff
			currentDiff = &DiffInfo{}

			// Parse file paths from "diff --git a/file b/file"
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentDiff.OldPath = strings.TrimPrefix(parts[2], "a/")
				currentDiff.NewPath = strings.TrimPrefix(parts[3], "b/")
				currentDiff.FilePath = currentDiff.NewPath
			}
		} else if strings.HasPrefix(line, "new file mode") {
			if currentDiff != nil {
				currentDiff.Status = "A" // Added
			}
		} else if strings.HasPrefix(line, "deleted file mode") {
			if currentDiff != nil {
				currentDiff.Status = "D" // Deleted
				currentDiff.FilePath = currentDiff.OldPath
			}
		} else if strings.HasPrefix(line, "rename from") {
			if currentDiff != nil {
				currentDiff.Status = "R" // Renamed
			}
		} else if strings.HasPrefix(line, "Binary files") {
			if currentDiff != nil {
				currentDiff.IsBinary = true
			}
		} else if strings.HasPrefix(line, "@@") {
			// Hunk header - extract line counts
			if currentDiff != nil && currentDiff.Status == "" {
				currentDiff.Status = "M" // Modified
			}
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			if currentDiff != nil {
				currentDiff.Additions++
			}
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			if currentDiff != nil {
				currentDiff.Deletions++
			}
		}

		contentLines = append(contentLines, line)
	}

	// Save last diff
	if currentDiff != nil {
		currentDiff.Content = strings.Join(contentLines, "\n")
		diffs = append(diffs, *currentDiff)
	}

	return diffs
}

// StageFiles stages the specified files
func (r *Repository) StageFiles(filePaths ...string) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no files specified to stage")
	}

	args := append([]string{"add"}, filePaths...)
	_, err := r.runGitCommand(args...)
	return err
}

// UnstageFiles unstages the specified files
func (r *Repository) UnstageFiles(filePaths ...string) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no files specified to unstage")
	}

	args := append([]string{"reset", "HEAD"}, filePaths...)
	_, err := r.runGitCommand(args...)
	return err
}

// StageAll stages all modified files
func (r *Repository) StageAll() error {
	_, err := r.runGitCommand("add", ".")
	return err
}

// Commit creates a new commit with the specified message
func (r *Repository) Commit(message string) error {
	if message == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	_, err := r.runGitCommand("commit", "-m", message)
	return err
}

// GetCommitHistory returns the commit history
func (r *Repository) GetCommitHistory(limit int) ([]CommitInfo, error) {
	args := []string{"log", "--pretty=format:%H|%an|%ad|%s|%b", "--date=iso"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	output, err := r.runGitCommand(args...)
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 4 {
			continue
		}

		commit := CommitInfo{
			Hash:      parts[0],
			Author:    parts[1],
			Subject:   parts[3],
			ShortHash: parts[0][:8],
		}

		// Parse date
		if date, err := time.Parse("2006-01-02 15:04:05 -0700", parts[2]); err == nil {
			commit.Date = date
		}

		// Add body if present
		if len(parts) == 5 {
			commit.Body = parts[4]
		}

		commits = append(commits, commit)
	}

	return commits, scanner.Err()
}

// GetCurrentBranch returns the current branch name
func (r *Repository) GetCurrentBranch() (string, error) {
	output, err := r.runGitCommand("branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// HasStagedChanges checks if there are staged changes
func (r *Repository) HasStagedChanges() (bool, error) {
	output, err := r.runGitCommand("diff", "--cached", "--name-only")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

// HasUnstagedChanges checks if there are unstaged changes
func (r *Repository) HasUnstagedChanges() (bool, error) {
	output, err := r.runGitCommand("diff", "--name-only")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

// GetRemoteURL returns the remote URL
func (r *Repository) GetRemoteURL(remoteName string) (string, error) {
	if remoteName == "" {
		remoteName = "origin"
	}

	output, err := r.runGitCommand("remote", "get-url", remoteName)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// IsClean checks if the working directory is clean (no changes)
func (r *Repository) IsClean() (bool, error) {
	output, err := r.runGitCommand("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "", nil
}

// GetLastCommitHash returns the hash of the last commit
func (r *Repository) GetLastCommitHash() (string, error) {
	output, err := r.runGitCommand("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// RunGitCommand executes a Git command and returns the output (public method)
func (r *Repository) RunGitCommand(args ...string) (string, error) {
	return r.runGitCommand(args...)
}

// ParseDiff parses git diff output into DiffInfo structs (public method)
func (r *Repository) ParseDiff(diffOutput string) []DiffInfo {
	return r.parseDiff(diffOutput)
}
