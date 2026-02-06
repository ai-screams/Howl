package internal

import (
	"context"
	"os/exec"
	"time"
)

type GitInfo struct {
	Branch string
	Dirty  bool
}

// getGitInfo runs git commands with a tight timeout.
// Returns nil on any failure — git info is optional.
func GetGitInfo(dir string) *GitInfo {
	if dir == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	branch := gitBranch(ctx, dir)
	if branch == "" {
		return nil
	}

	return &GitInfo{
		Branch: branch,
		Dirty:  gitDirty(ctx, dir),
	}
}

func gitBranch(ctx context.Context, dir string) string {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	// trim trailing newline
	s := string(out)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func gitDirty(ctx context.Context, dir string) bool {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--untracked-files=no")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(out) > 0
}
