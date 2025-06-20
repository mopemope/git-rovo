package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopemope/git-rovo/internal/logger"
)

func TestGetUntrackedFileDiff(t *testing.T) {
	repo, tempDir := setupUntrackedTestRepo(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create untracked text file
	textFile := filepath.Join(tempDir, "untracked.txt")
	textContent := "line 1\nline 2\nline 3"
	if err := os.WriteFile(textFile, []byte(textContent), 0644); err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	// Create untracked binary file (simulate with null bytes)
	binaryFile := filepath.Join(tempDir, "binary.dat")
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	// Test getting all untracked file diffs
	diffs, err := repo.GetUntrackedFileDiff()
	if err != nil {
		t.Fatalf("Failed to get untracked file diff: %v", err)
	}

	if len(diffs) != 2 {
		t.Errorf("Expected 2 diffs, got %d", len(diffs))
	}

	// Find text file diff
	var textDiff *DiffInfo
	var binaryDiff *DiffInfo
	for i := range diffs {
		switch diffs[i].FilePath {
		case "untracked.txt":
			textDiff = &diffs[i]
		case "binary.dat":
			binaryDiff = &diffs[i]
		}
	}

	// Verify text file diff
	if textDiff == nil {
		t.Fatal("Text file diff not found")
	}
	if textDiff.Status != "A" {
		t.Errorf("Expected status 'A', got '%s'", textDiff.Status)
	}
	if textDiff.IsBinary {
		t.Error("Text file should not be marked as binary")
	}
	if !strings.Contains(textDiff.Content, "+line 1") {
		t.Error("Text file diff should contain added lines")
	}
	if !strings.Contains(textDiff.Content, "new file mode") {
		t.Error("Text file diff should indicate new file")
	}

	// Verify binary file diff
	if binaryDiff == nil {
		t.Fatal("Binary file diff not found")
	}
	if binaryDiff.Status != "A" {
		t.Errorf("Expected status 'A', got '%s'", binaryDiff.Status)
	}
	if !binaryDiff.IsBinary {
		t.Error("Binary file should be marked as binary")
	}
	if !strings.Contains(binaryDiff.Content, "Binary file") {
		t.Error("Binary file diff should indicate binary file")
	}
}

func TestGetUntrackedFileDiffSpecificFiles(t *testing.T) {
	repo, tempDir := setupUntrackedTestRepo(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create multiple untracked files
	files := map[string]string{
		"file1.txt": "content 1",
		"file2.txt": "content 2",
		"file3.txt": "content 3",
	}

	for filename, content := range files {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	// Test getting specific files
	diffs, err := repo.GetUntrackedFileDiff("file1.txt", "file3.txt")
	if err != nil {
		t.Fatalf("Failed to get specific untracked file diffs: %v", err)
	}

	if len(diffs) != 2 {
		t.Errorf("Expected 2 diffs, got %d", len(diffs))
	}

	// Verify we got the right files
	foundFiles := make(map[string]bool)
	for _, diff := range diffs {
		foundFiles[diff.FilePath] = true
	}

	if !foundFiles["file1.txt"] {
		t.Error("file1.txt diff not found")
	}
	if !foundFiles["file3.txt"] {
		t.Error("file3.txt diff not found")
	}
	if foundFiles["file2.txt"] {
		t.Error("file2.txt should not be included")
	}
}

func TestGetUntrackedFileDiffNoUntrackedFiles(t *testing.T) {
	repo, tempDir := setupUntrackedTestRepo(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// No untracked files created
	diffs, err := repo.GetUntrackedFileDiff()
	if err != nil {
		t.Fatalf("Failed to get untracked file diff: %v", err)
	}

	if len(diffs) != 0 {
		t.Errorf("Expected 0 diffs, got %d", len(diffs))
	}
}

func TestIsBinaryContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Text content",
			content:  []byte("Hello, world!\nThis is text."),
			expected: false,
		},
		{
			name:     "Content with null byte",
			content:  []byte("Hello\x00world"),
			expected: true,
		},
		{
			name:     "Empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "Content with tabs and newlines",
			content:  []byte("Line 1\t\nLine 2\r\n"),
			expected: false,
		},
		{
			name:     "High ratio of non-printable",
			content:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x0B, 0x0C},
			expected: true,
		},
		{
			name:     "Mixed printable and non-printable (low ratio)",
			content:  []byte("Hello world\x01\x02"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinaryContent(tt.content)
			if result != tt.expected {
				t.Errorf("isBinaryContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateUntrackedFileDiff(t *testing.T) {
	repo, tempDir := setupUntrackedTestRepo(t)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nline 3"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate diff
	diff, err := repo.generateUntrackedFileDiff("test.txt")
	if err != nil {
		t.Fatalf("Failed to generate untracked file diff: %v", err)
	}

	if diff == nil {
		t.Fatal("Expected diff, got nil")
	}

	// Verify diff properties
	if diff.FilePath != "test.txt" {
		t.Errorf("Expected FilePath 'test.txt', got '%s'", diff.FilePath)
	}
	if diff.Status != "A" {
		t.Errorf("Expected Status 'A', got '%s'", diff.Status)
	}
	if diff.IsBinary {
		t.Error("Text file should not be marked as binary")
	}

	// Verify diff content structure
	if !strings.Contains(diff.Content, "diff --git") {
		t.Error("Diff should contain git diff header")
	}
	if !strings.Contains(diff.Content, "new file mode") {
		t.Error("Diff should indicate new file")
	}
	if !strings.Contains(diff.Content, "+line 1") {
		t.Error("Diff should contain added lines")
	}
	if !strings.Contains(diff.Content, "@@ -0,0 +1,3 @@") {
		t.Error("Diff should contain correct hunk header")
	}
}

// Helper function to setup test repository for untracked file tests
func setupUntrackedTestRepo(t *testing.T) (*Repository, string) {
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
	repo, err := initUntrackedGitRepo(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	return repo, tempDir
}

func initUntrackedGitRepo(dir string) (*Repository, error) {
	// Run git init
	if err := runUntrackedCommand(dir, "git", "init"); err != nil {
		return nil, err
	}

	// Configure git user for testing
	if err := runUntrackedCommand(dir, "git", "config", "user.name", "Test User"); err != nil {
		return nil, err
	}
	if err := runUntrackedCommand(dir, "git", "config", "user.email", "test@example.com"); err != nil {
		return nil, err
	}

	return New(dir)
}

func runUntrackedCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}
