package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// CommandRunner abstracts exec.Command for testability.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

// realRunner runs actual system commands.
type realRunner struct{}

func (r *realRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return string(out), nil
}

// GitClient provides Git operations for the sync pipeline.
type GitClient struct {
	runner CommandRunner
}

// NewGitClient creates a GitClient with real command execution.
func NewGitClient() *GitClient {
	return &GitClient{runner: &realRunner{}}
}

// Clone clones a repository into the given directory.
func (g *GitClient) Clone(url, dir string) error {
	_, err := g.runner.Run("git", "clone", url, dir)
	return err
}

// CheckoutBranch creates and checks out a new branch.
func (g *GitClient) CheckoutBranch(dir, branch string) error {
	_, err := g.runner.Run("git", "-C", dir, "checkout", "-b", branch)
	return err
}

// HasChanges returns true if there are unstaged changes.
func (g *GitClient) HasChanges(dir string) (bool, error) {
	out, err := g.runner.Run("git", "-C", dir, "diff", "--quiet")
	if err != nil {
		return false, nil // diff --quiet exits 1 when there are changes
	}
	return strings.TrimSpace(out) != "", nil
}

// Commit stages all changes and commits with the given message.
func (g *GitClient) Commit(dir, message string) error {
	if _, err := g.runner.Run("git", "-C", dir, "add", "-A"); err != nil {
		return err
	}
	_, err := g.runner.Run("git", "-C", dir, "commit", "-m", message)
	return err
}

// Push pushes the branch to the remote.
func (g *GitClient) Push(dir, remote, branch string) error {
	_, err := g.runner.Run("git", "-C", dir, "push", remote, branch)
	return err
}

// GetSHA returns the current HEAD commit SHA.
func (g *GitClient) GetSHA(dir string) (string, error) {
	out, err := g.runner.Run("git", "-C", dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
