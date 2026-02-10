package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a temporary git repository with an initial commit.
// Returns the directory path. Skips the test if git is not available.
func initGitRepo(t *testing.T) string {
	t.Helper()

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping git tests")
	}

	dir := t.TempDir()

	// Git identity environment variables for CI compatibility
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)

	// Initialize git repository with 'main' branch
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure user identity
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config user.email failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config user.name failed: %v", err)
	}

	// Create initial file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Stage the file
	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Create initial commit
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	return dir
}

// runGitCommand runs a git command in the specified directory with test environment.
func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
}

func TestGetGitInfo_EmptyDir(t *testing.T) {
	t.Parallel()

	result := GetGitInfo("")
	if result != nil {
		t.Errorf("GetGitInfo(\"\") = %+v, want nil", result)
	}
}

func TestGetGitInfo_NonGitDir(t *testing.T) {
	t.Parallel()

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping git tests")
	}

	dir := t.TempDir()

	result := GetGitInfo(dir)
	if result != nil {
		t.Errorf("GetGitInfo(non-git-dir) = %+v, want nil", result)
	}
}

func TestGetGitInfo_ValidRepo(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	result := GetGitInfo(dir)
	if result == nil {
		t.Fatal("GetGitInfo(valid-repo) = nil, want non-nil")
	}

	if result.Branch != "main" {
		t.Errorf("GetGitInfo(valid-repo).Branch = %q, want %q", result.Branch, "main")
	}

	if result.Dirty {
		t.Errorf("GetGitInfo(valid-repo).Dirty = true, want false")
	}
}

func TestGetGitInfo_DirtyRepo(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	// Modify the tracked file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified content\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	result := GetGitInfo(dir)
	if result == nil {
		t.Fatal("GetGitInfo(dirty-repo) = nil, want non-nil")
	}

	if result.Branch != "main" {
		t.Errorf("GetGitInfo(dirty-repo).Branch = %q, want %q", result.Branch, "main")
	}

	if !result.Dirty {
		t.Errorf("GetGitInfo(dirty-repo).Dirty = false, want true")
	}
}

func TestGetGitInfo_CleanRepo(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	// Modify the tracked file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("second version\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Stage and commit the change
	runGitCommand(t, dir, "add", "test.txt")
	runGitCommand(t, dir, "commit", "-m", "Second commit")

	result := GetGitInfo(dir)
	if result == nil {
		t.Fatal("GetGitInfo(clean-repo) = nil, want non-nil")
	}

	if result.Branch != "main" {
		t.Errorf("GetGitInfo(clean-repo).Branch = %q, want %q", result.Branch, "main")
	}

	if result.Dirty {
		t.Errorf("GetGitInfo(clean-repo).Dirty = true, want false")
	}
}

func TestGetGitInfo_DetachedHEAD(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	// Get the commit hash
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	commitHash := string(out[:7]) // Use first 7 characters

	// Checkout the commit hash (detached HEAD state)
	runGitCommand(t, dir, "checkout", commitHash)

	result := GetGitInfo(dir)
	if result == nil {
		t.Fatal("GetGitInfo(detached-head) = nil, want non-nil")
	}

	if result.Branch != "HEAD" {
		t.Errorf("GetGitInfo(detached-head).Branch = %q, want %q", result.Branch, "HEAD")
	}

	if result.Dirty {
		t.Errorf("GetGitInfo(detached-head).Dirty = true, want false")
	}
}

func TestGetGitInfo_UntrackedFilesIgnored(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	// Create an untracked file
	untrackedFile := filepath.Join(dir, "untracked.txt")
	if err := os.WriteFile(untrackedFile, []byte("untracked content\n"), 0644); err != nil {
		t.Fatalf("failed to create untracked file: %v", err)
	}

	result := GetGitInfo(dir)
	if result == nil {
		t.Fatal("GetGitInfo(untracked-files) = nil, want non-nil")
	}

	if result.Branch != "main" {
		t.Errorf("GetGitInfo(untracked-files).Branch = %q, want %q", result.Branch, "main")
	}

	// Untracked files should NOT make the repo dirty (--untracked-files=no)
	if result.Dirty {
		t.Errorf("GetGitInfo(untracked-files).Dirty = true, want false (untracked files should be ignored)")
	}
}

func TestGetGitInfo_StagedChanges(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	// Modify and stage a file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("staged content\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	runGitCommand(t, dir, "add", "test.txt")

	result := GetGitInfo(dir)
	if result == nil {
		t.Fatal("GetGitInfo(staged-changes) = nil, want non-nil")
	}

	if result.Branch != "main" {
		t.Errorf("GetGitInfo(staged-changes).Branch = %q, want %q", result.Branch, "main")
	}

	// Staged changes should make the repo dirty
	if !result.Dirty {
		t.Errorf("GetGitInfo(staged-changes).Dirty = false, want true")
	}
}

func TestGetGitInfo_DifferentBranchName(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	// Create and checkout a new branch
	runGitCommand(t, dir, "checkout", "-b", "feature/test-branch")

	result := GetGitInfo(dir)
	if result == nil {
		t.Fatal("GetGitInfo(different-branch) = nil, want non-nil")
	}

	if result.Branch != "feature/test-branch" {
		t.Errorf("GetGitInfo(different-branch).Branch = %q, want %q", result.Branch, "feature/test-branch")
	}

	if result.Dirty {
		t.Errorf("GetGitInfo(different-branch).Dirty = true, want false")
	}
}
