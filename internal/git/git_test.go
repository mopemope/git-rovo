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

func TestDiscardChanges(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-test-discard-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for testing
	configCmd := exec.Command("git", "config", "user.name", "Test User")
	configCmd.Dir = tempDir
	configCmd.Run()
	configCmd = exec.Command("git", "config", "user.email", "test@example.com")
	configCmd.Dir = tempDir
	configCmd.Run()

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
