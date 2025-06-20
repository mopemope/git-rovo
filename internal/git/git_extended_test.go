package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mopemope/git-rovo/internal/logger"
)

func TestUnstageFiles(t *testing.T) {
	repo, tempDir := setupTestRepoExtended(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create initial commit first to establish HEAD
	initialFile := filepath.Join(tempDir, "initial.txt")
	if err := os.WriteFile(initialFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	if err := repo.StageFiles("initial.txt"); err != nil {
		t.Fatalf("Failed to stage initial file: %v", err)
	}

	if err := repo.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Now create and stage a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Stage the file
	if err := repo.StageFiles("test.txt"); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Verify file is staged
	status, err := repo.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	staged := false
	for _, file := range status {
		if file.Path == "test.txt" && (file.Status == "A" || file.Staged) {
			staged = true
			break
		}
	}
	if !staged {
		t.Fatal("File should be staged")
	}

	// Unstage the file
	if err := repo.UnstageFiles("test.txt"); err != nil {
		t.Fatalf("Failed to unstage file: %v", err)
	}

	// Verify file is unstaged
	status, err = repo.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	unstaged := false
	for _, file := range status {
		if file.Path == "test.txt" && (file.Status == "??" || !file.Staged) {
			unstaged = true
			break
		}
	}
	if !unstaged {
		t.Fatal("File should be unstaged")
	}
}

func TestStageAll(t *testing.T) {
	repo, tempDir := setupTestRepoExtended(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create multiple test files
	files := []string{"test1.txt", "test2.txt", "test3.txt"}
	for _, file := range files {
		testFile := filepath.Join(tempDir, file)
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Stage all files
	if err := repo.StageAll(); err != nil {
		t.Fatalf("Failed to stage all files: %v", err)
	}

	// Verify all files are staged
	status, err := repo.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	stagedCount := 0
	for _, fileStatus := range status {
		if fileStatus.Status == "A" || fileStatus.Staged {
			stagedCount++
		}
	}

	if stagedCount != len(files) {
		t.Errorf("Expected %d staged files, got %d", len(files), stagedCount)
	}
}

func TestHasUnstagedChanges(t *testing.T) {
	repo, tempDir := setupTestRepoExtended(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create initial commit first
	initialFile := filepath.Join(tempDir, "initial.txt")
	if err := os.WriteFile(initialFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	if err := repo.StageFiles("initial.txt"); err != nil {
		t.Fatalf("Failed to stage initial file: %v", err)
	}

	if err := repo.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Initially should have no unstaged changes
	hasChanges, err := repo.HasUnstagedChanges()
	if err != nil {
		t.Fatalf("Failed to check unstaged changes: %v", err)
	}
	if hasChanges {
		t.Error("Should not have unstaged changes initially")
	}

	// Modify the existing file (this should create unstaged changes)
	if err := os.WriteFile(initialFile, []byte("modified initial content"), 0644); err != nil {
		t.Fatalf("Failed to modify initial file: %v", err)
	}

	// Should now have unstaged changes
	hasChanges, err = repo.HasUnstagedChanges()
	if err != nil {
		t.Fatalf("Failed to check unstaged changes: %v", err)
	}
	if !hasChanges {
		t.Error("Should have unstaged changes after modifying committed file")
	}

	// Stage the modified file
	if err := repo.StageFiles("initial.txt"); err != nil {
		t.Fatalf("Failed to stage modified file: %v", err)
	}

	// Should not have unstaged changes after staging
	hasChanges, err = repo.HasUnstagedChanges()
	if err != nil {
		t.Fatalf("Failed to check unstaged changes: %v", err)
	}
	if hasChanges {
		t.Error("Should not have unstaged changes after staging all modifications")
	}
}

func TestGetRemoteURL(t *testing.T) {
	repo, tempDir := setupTestRepoExtended(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Initially should have no remote
	remoteURL, err := repo.GetRemoteURL("origin")
	if err == nil {
		t.Error("Expected error for repository without remote")
	}
	if remoteURL != "" {
		t.Error("Remote URL should be empty for repository without remote")
	}

	// Add a remote
	testURL := "https://github.com/test/repo.git"
	_, err = repo.runGitCommand("remote", "add", "origin", testURL)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	// Should now return the remote URL
	remoteURL, err = repo.GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("Failed to get remote URL: %v", err)
	}
	if remoteURL != testURL {
		t.Errorf("Expected remote URL %s, got %s", testURL, remoteURL)
	}
}

func TestGetLastCommitHash(t *testing.T) {
	repo, tempDir := setupTestRepoExtended(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Initially should have no commits
	hash, err := repo.GetLastCommitHash()
	if err == nil {
		t.Error("Expected error for repository without commits")
	}
	if hash != "" {
		t.Error("Hash should be empty for repository without commits")
	}

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := repo.StageFiles("test.txt"); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	if err := repo.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Should now return a commit hash
	hash, err = repo.GetLastCommitHash()
	if err != nil {
		t.Fatalf("Failed to get last commit hash: %v", err)
	}
	if hash == "" {
		t.Error("Hash should not be empty after commit")
	}
	if len(hash) != 40 { // Git SHA-1 hash length
		t.Errorf("Expected hash length 40, got %d", len(hash))
	}
}

// Helper function to setup test repository (renamed to avoid conflict)
func setupTestRepoExtended(t *testing.T) (*Repository, string) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Initialize logger for testing
	logPath := filepath.Join(tempDir, "test.log")
	if err := logger.Init(logPath, "info"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize git repository
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	repo, err := initGitRepoExtended(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	return repo, tempDir
}

func initGitRepoExtended(dir string) (*Repository, error) {
	// Run git init
	if err := runCommandExtended(dir, "git", "init"); err != nil {
		return nil, err
	}

	// Configure git user for testing
	if err := runCommandExtended(dir, "git", "config", "user.name", "Test User"); err != nil {
		return nil, err
	}
	if err := runCommandExtended(dir, "git", "config", "user.email", "test@example.com"); err != nil {
		return nil, err
	}

	return New(dir)
}

func runCommandExtended(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}
