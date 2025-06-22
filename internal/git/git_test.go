package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopemope/git-rovo/internal/logger"
)

func setupTestRepo(t *testing.T) (*Repository, string) {
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
	repo, err := initGitRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	return repo, tempDir
}

func initGitRepo(dir string) (*Repository, error) {
	// Run git init
	if err := runCommand(dir, "git", "init"); err != nil {
		return nil, err
	}

	// Configure git user for testing
	if err := runCommand(dir, "git", "config", "user.name", "Test User"); err != nil {
		return nil, err
	}
	if err := runCommand(dir, "git", "config", "user.email", "test@example.com"); err != nil {
		return nil, err
	}

	return New(dir)
}

func runCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

func TestNew(t *testing.T) {
	repo, _ := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	if repo == nil {
		t.Fatal("Expected repository to be created")
	}

	if repo.GetWorkDir() == "" {
		t.Error("Expected work directory to be set")
	}
}

func TestIsGitRepository(t *testing.T) {
	_, tempDir := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	if !IsGitRepository(tempDir) {
		t.Error("Expected directory to be recognized as git repository")
	}

	// Test non-git directory
	nonGitDir := t.TempDir()
	if IsGitRepository(nonGitDir) {
		t.Error("Expected directory to not be recognized as git repository")
	}
}

func TestGetStatus(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	status, err := repo.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(status) == 0 {
		t.Error("Expected at least one file in status")
	}

	// Check if our test file is in the status
	found := false
	for _, file := range status {
		if file.Path == "test.txt" {
			found = true
			if file.Status != "??" {
				t.Errorf("Expected status '??', got '%s'", file.Status)
			}
			break
		}
	}

	if !found {
		t.Error("Expected test.txt to be in status")
	}
}

func TestStageFiles(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Stage the file
	if err := repo.StageFiles("test.txt"); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Check if file is staged
	status, err := repo.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	found := false
	for _, file := range status {
		if file.Path == "test.txt" && file.Staged {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected test.txt to be staged")
	}
}

func TestCommit(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	// Create and stage a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := repo.StageFiles("test.txt"); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Commit the file
	commitMessage := "Add test file"
	if err := repo.Commit(commitMessage); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify commit was created
	commits, err := repo.GetCommitHistory(1)
	if err != nil {
		t.Fatalf("Failed to get commit history: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("Expected at least one commit")
	}

	if commits[0].Subject != commitMessage {
		t.Errorf("Expected commit message '%s', got '%s'", commitMessage, commits[0].Subject)
	}
}

func TestGetDiff(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	// Create and commit initial file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := repo.StageFiles("test.txt"); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	if err := repo.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Modify the file
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Get unstaged diff
	diffs, err := repo.GetDiff(false)
	if err != nil {
		t.Fatalf("Failed to get diff: %v", err)
	}

	if len(diffs) == 0 {
		t.Error("Expected at least one diff")
	}

	// Check if our file is in the diff
	found := false
	for _, diff := range diffs {
		if diff.FilePath == "test.txt" {
			found = true
			if diff.Status != "M" {
				t.Errorf("Expected status 'M', got '%s'", diff.Status)
			}
			break
		}
	}

	if !found {
		t.Error("Expected test.txt to be in diff")
	}
}

func TestHasStagedChanges(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	// Initially should have no staged changes
	hasStaged, err := repo.HasStagedChanges()
	if err != nil {
		t.Fatalf("Failed to check staged changes: %v", err)
	}
	if hasStaged {
		t.Error("Expected no staged changes initially")
	}

	// Create and stage a file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := repo.StageFiles("test.txt"); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Now should have staged changes
	hasStaged, err = repo.HasStagedChanges()
	if err != nil {
		t.Fatalf("Failed to check staged changes: %v", err)
	}
	if !hasStaged {
		t.Error("Expected staged changes after staging file")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repo, _ := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	branch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	// Default branch is usually 'main' or 'master'
	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch 'main' or 'master', got '%s'", branch)
	}
}

func TestIsClean(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer func() { _ = logger.Close() }()

	// Initially should be clean
	clean, err := repo.IsClean()
	if err != nil {
		t.Fatalf("Failed to check if clean: %v", err)
	}
	if !clean {
		t.Error("Expected repository to be clean initially")
	}

	// Create a file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Now should not be clean
	clean, err = repo.IsClean()
	if err != nil {
		t.Fatalf("Failed to check if clean: %v", err)
	}
	if clean {
		t.Error("Expected repository to not be clean after adding file")
	}
}

func TestAmendCommit(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-test-amend-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for testing
	configCmd := exec.Command("git", "config", "user.name", "Test User")
	configCmd.Dir = tempDir
	if err := configCmd.Run(); err != nil {
		t.Logf("Failed to set git user name: %v", err)
	}
	configCmd = exec.Command("git", "config", "user.email", "test@example.com")
	configCmd.Dir = tempDir
	if err := configCmd.Run(); err != nil {
		t.Logf("Failed to set git user email: %v", err)
	}

	repo := &Repository{workDir: tempDir}

	t.Run("Amend commit with no existing commits", func(t *testing.T) {
		err := repo.AmendCommit("amended message")
		if err == nil {
			t.Error("Expected error when amending with no existing commits")
		}
		if !strings.Contains(err.Error(), "no commits found") {
			t.Errorf("Expected 'no commits found' error, got: %v", err)
		}
	})

	// Create and commit a file for subsequent tests
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	addCmd := exec.Command("git", "add", "test.txt")
	addCmd.Dir = tempDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "initial commit")
	commitCmd.Dir = tempDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	t.Run("Amend commit successfully", func(t *testing.T) {
		// Get original commit hash
		originalHash, err := repo.GetLastCommitHash()
		if err != nil {
			t.Fatalf("Failed to get original commit hash: %v", err)
		}

		// Amend the commit
		err = repo.AmendCommit("amended commit message")
		if err != nil {
			t.Fatalf("AmendCommit failed: %v", err)
		}

		// Verify commit hash changed
		newHash, err := repo.GetLastCommitHash()
		if err != nil {
			t.Fatalf("Failed to get new commit hash: %v", err)
		}

		if originalHash == newHash {
			t.Error("Expected commit hash to change after amend")
		}

		// Verify commit message changed
		message, err := repo.GetLastCommitMessage()
		if err != nil {
			t.Fatalf("Failed to get commit message: %v", err)
		}

		if message != "amended commit message" {
			t.Errorf("Expected message 'amended commit message', got '%s'", message)
		}
	})

	t.Run("Amend commit with empty message", func(t *testing.T) {
		err := repo.AmendCommit("")
		if err == nil {
			t.Error("Expected error when amending with empty message")
		}
		if !strings.Contains(err.Error(), "commit message cannot be empty") {
			t.Errorf("Expected 'commit message cannot be empty' error, got: %v", err)
		}
	})

	t.Run("Amend commit with staged changes", func(t *testing.T) {
		// Modify and stage a file
		if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		stageCmd := exec.Command("git", "add", "test.txt")
		stageCmd.Dir = tempDir
		if err := stageCmd.Run(); err != nil {
			t.Fatalf("Failed to stage file: %v", err)
		}

		// Amend the commit
		err := repo.AmendCommit("amended with staged changes")
		if err != nil {
			t.Fatalf("AmendCommit with staged changes failed: %v", err)
		}

		// Verify the staged changes were included
		message, err := repo.GetLastCommitMessage()
		if err != nil {
			t.Fatalf("Failed to get commit message: %v", err)
		}

		if message != "amended with staged changes" {
			t.Errorf("Expected message 'amended with staged changes', got '%s'", message)
		}

		// Verify no staged changes remain
		hasStaged, err := repo.HasStagedChanges()
		if err != nil {
			t.Fatalf("Failed to check staged changes: %v", err)
		}

		if hasStaged {
			t.Error("Expected no staged changes after amend")
		}
	})
}

func TestGetLastCommitMessage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-test-message-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for testing
	configCmd := exec.Command("git", "config", "user.name", "Test User")
	configCmd.Dir = tempDir
	if err := configCmd.Run(); err != nil {
		t.Logf("Failed to set git user name: %v", err)
	}
	configCmd = exec.Command("git", "config", "user.email", "test@example.com")
	configCmd.Dir = tempDir
	if err := configCmd.Run(); err != nil {
		t.Logf("Failed to set git user email: %v", err)
	}

	repo := &Repository{workDir: tempDir}

	t.Run("Get message with no commits", func(t *testing.T) {
		_, err := repo.GetLastCommitMessage()
		if err == nil {
			t.Error("Expected error when getting message with no commits")
		}
		if !strings.Contains(err.Error(), "no commits found") {
			t.Errorf("Expected 'no commits found' error, got: %v", err)
		}
	})

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	addCmd := exec.Command("git", "add", "test.txt")
	addCmd.Dir = tempDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitMessage := "test commit message"
	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	commitCmd.Dir = tempDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	t.Run("Get commit message successfully", func(t *testing.T) {
		message, err := repo.GetLastCommitMessage()
		if err != nil {
			t.Fatalf("GetLastCommitMessage failed: %v", err)
		}

		if message != commitMessage {
			t.Errorf("Expected message '%s', got '%s'", commitMessage, message)
		}
	})

	// Test with multi-line commit message
	multiLineMessage := "feat: add new feature"
	commitCmd = exec.Command("git", "commit", "--allow-empty", "-m", multiLineMessage)
	commitCmd.Dir = tempDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to create multi-line commit: %v", err)
	}

	t.Run("Get multi-line commit message", func(t *testing.T) {
		message, err := repo.GetLastCommitMessage()
		if err != nil {
			t.Fatalf("GetLastCommitMessage failed: %v", err)
		}

		if message != multiLineMessage {
			t.Errorf("Expected message '%s', got '%s'", multiLineMessage, message)
		}
	})
}

func TestDiscardChanges(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-test-discard-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for testing
	configCmd := exec.Command("git", "config", "user.name", "Test User")
	configCmd.Dir = tempDir
	if err := configCmd.Run(); err != nil {
		t.Logf("Failed to set git user name: %v", err)
	}
	configCmd = exec.Command("git", "config", "user.email", "test@example.com")
	configCmd.Dir = tempDir
	if err := configCmd.Run(); err != nil {
		t.Logf("Failed to set git user email: %v", err)
	}

	repo := &Repository{workDir: tempDir}

	t.Run("Discard untracked file", func(t *testing.T) {
		// Create untracked file
		testFile := filepath.Join(tempDir, "untracked.txt")
		if err := os.WriteFile(testFile, []byte("untracked content"), 0644); err != nil {
			t.Fatalf("Failed to create untracked file: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Fatal("Untracked file should exist")
		}

		// Discard changes (should remove the file)
		err := repo.DiscardChanges("untracked.txt")
		if err != nil {
			t.Fatalf("DiscardChanges failed: %v", err)
		}

		// Verify file is removed
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("Untracked file should be removed")
		}
	})

	t.Run("Discard modified file", func(t *testing.T) {
		// Create and commit a file
		testFile := filepath.Join(tempDir, "tracked.txt")
		originalContent := "original content\n"
		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create tracked file: %v", err)
		}

		// Add and commit the file
		addCmd := exec.Command("git", "add", "tracked.txt")
		addCmd.Dir = tempDir
		if err := addCmd.Run(); err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
		commitCmd.Dir = tempDir
		if err := commitCmd.Run(); err != nil {
			t.Fatalf("Failed to commit file: %v", err)
		}

		// Modify the file
		modifiedContent := "modified content\n"
		if err := os.WriteFile(testFile, []byte(modifiedContent), 0644); err != nil {
			t.Fatalf("Failed to modify file: %v", err)
		}

		// Discard changes
		err := repo.DiscardChanges("tracked.txt")
		if err != nil {
			t.Fatalf("DiscardChanges failed: %v", err)
		}

		// Verify file content is restored
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(content) != originalContent {
			t.Errorf("Expected content '%s', got '%s'", originalContent, string(content))
		}
	})

	t.Run("Discard staged file", func(t *testing.T) {
		// Create and commit a file
		testFile := filepath.Join(tempDir, "staged.txt")
		originalContent := "original staged content\n"
		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create staged file: %v", err)
		}

		// Add and commit the file
		addCmd := exec.Command("git", "add", "staged.txt")
		addCmd.Dir = tempDir
		if err := addCmd.Run(); err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		commitCmd := exec.Command("git", "commit", "-m", "Add staged file")
		commitCmd.Dir = tempDir
		if err := commitCmd.Run(); err != nil {
			t.Fatalf("Failed to commit file: %v", err)
		}

		// Modify and stage the file
		modifiedContent := "modified staged content\n"
		if err := os.WriteFile(testFile, []byte(modifiedContent), 0644); err != nil {
			t.Fatalf("Failed to modify file: %v", err)
		}

		stageCmd := exec.Command("git", "add", "staged.txt")
		stageCmd.Dir = tempDir
		if err := stageCmd.Run(); err != nil {
			t.Fatalf("Failed to stage file: %v", err)
		}

		// Discard changes
		err := repo.DiscardChanges("staged.txt")
		if err != nil {
			t.Fatalf("DiscardChanges failed: %v", err)
		}

		// Verify file content is restored
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(content) != originalContent {
			t.Errorf("Expected content '%s', got '%s'", originalContent, string(content))
		}

		// Verify file is not staged
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Dir = tempDir
		output, err := statusCmd.Output()
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if strings.Contains(string(output), "staged.txt") {
			t.Error("File should not appear in git status after discard")
		}
	})
}
