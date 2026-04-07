package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	stdsync "sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// TargetResult holds the result of syncing a single target.
type TargetResult struct {
	Target  Target
	Skipped bool
	PRURL   string
	Error   error
}

// GitOps defines the Git operations needed by the sync pipeline.
type GitOps interface {
	Clone(url, dir string) error
	CheckoutBranch(dir, branch string) error
	HasChanges(dir string) (bool, error)
	Commit(dir, message string) error
	Push(dir, remote, branch string) error
	GetSHA(dir string) (string, error)
}

// GitHubOps defines the GitHub PR operations needed by the sync pipeline.
type GitHubOps interface {
	HasOpenPR(ctx context.Context, owner, repo, head string) (bool, error)
	CreatePullRequest(ctx context.Context, owner, repo, title, head, base, body string) (string, error)
}

// Syncer orchestrates the sync pipeline for all targets.
type Syncer struct {
	git         GitOps
	github      GitHubOps
	config      *Config
	srcCloneDir string
}

// NewSyncer creates a new Syncer.
func NewSyncer(git GitOps, gh GitHubOps, cfg *Config, srcDir string) *Syncer {
	return &Syncer{
		git:         git,
		github:      gh,
		config:      cfg,
		srcCloneDir: srcDir,
	}
}

// SyncAll runs the sync pipeline for all targets in parallel.
func (s *Syncer) SyncAll(ctx context.Context) []TargetResult {
	results := make([]TargetResult, len(s.config.Targets))
	g, _ := errgroup.WithContext(ctx)
	var mu stdsync.Mutex

	for i, target := range s.config.Targets {
		i, target := i, target
		g.Go(func() error {
			result := s.syncTarget(ctx, target)
			mu.Lock()
			results[i] = result
			mu.Unlock()
			return nil
		})
	}
	g.Wait()
	return results
}

func (s *Syncer) syncTarget(ctx context.Context, target Target) TargetResult {
	result := TargetResult{Target: target}

	// 1. Clone source if needed
	srcDir := s.srcCloneDir
	if srcDir == "" {
		srcDir = filepath.Join(os.TempDir(), "sync-src-"+time.Now().Format("20060102-150405"))
		sourceURL := fmt.Sprintf("https://github.com/%s/%s.git", s.config.Source.Owner, s.config.Source.Repo)
		if err := s.git.Clone(sourceURL, srcDir); err != nil {
			result.Error = fmt.Errorf("clone source: %w", err)
			return result
		}
		defer os.RemoveAll(srcDir)
	}

	// 2. Get source SHA
	sourceSHA, err := s.git.GetSHA(srcDir)
	if err != nil {
		sourceSHA = "unknown"
	}

	// 3. Clone target
	targetDir := filepath.Join(os.TempDir(), "sync-"+target.Repo+"-"+time.Now().Format("20060102-150405"))
	defer os.RemoveAll(targetDir)

	targetURL := fmt.Sprintf("https://github.com/%s/%s.git", target.Owner, target.Repo)
	if err := s.git.Clone(targetURL, targetDir); err != nil {
		result.Error = fmt.Errorf("clone target %s: %w", target.Repo, err)
		return result
	}

	// 4. Create branch
	branch := fmt.Sprintf("%s-%d", target.BranchPrefix, time.Now().Unix())
	if err := s.git.CheckoutBranch(targetDir, branch); err != nil {
		result.Error = fmt.Errorf("checkout branch: %w", err)
		return result
	}

	// 5. Check for existing PR
	head := fmt.Sprintf("%s:%s", target.Owner, branch)
	hasPR, err := s.github.HasOpenPR(ctx, target.Owner, target.Repo, head)
	if err != nil {
		result.Error = fmt.Errorf("check existing PR: %w", err)
		return result
	}
	if hasPR {
		result.Skipped = true
		return result
	}

	// 6. Copy files
	if _, err := CopyFiles(srcDir, targetDir, s.config.Files); err != nil {
		result.Error = fmt.Errorf("copy files: %w", err)
		return result
	}

	// 7. Check for changes
	changed, err := s.git.HasChanges(targetDir)
	if err != nil {
		result.Error = fmt.Errorf("check changes: %w", err)
		return result
	}
	if !changed {
		result.Skipped = true
		return result
	}

	// 8. Commit
	shortSHA := sourceSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}
	commitMsg := fmt.Sprintf("%s %s@%s", s.config.PR.TitlePrefix, s.config.Source.Repo, shortSHA)
	if err := s.git.Commit(targetDir, commitMsg); err != nil {
		result.Error = fmt.Errorf("commit: %w", err)
		return result
	}

	// 9. Push
	if err := s.git.Push(targetDir, "origin", branch); err != nil {
		result.Error = fmt.Errorf("push: %w", err)
		return result
	}

	// 10. Create PR
	title := fmt.Sprintf("%s %s@%s", s.config.PR.TitlePrefix, s.config.Source.Repo, shortSHA)
	body := fmt.Sprintf("## Security Base Sync\nAutomated sync from %s/%s@%s\n\nFiles changed:\n%s",
		s.config.Source.Owner, s.config.Source.Repo, sourceSHA,
		formatFileList(s.config.Files),
	)

	prURL, err := s.github.CreatePullRequest(ctx, target.Owner, target.Repo, title, head, "main", body)
	if err != nil {
		result.Error = fmt.Errorf("create PR: %w", err)
		return result
	}
	result.PRURL = prURL

	return result
}

func formatFileList(files []FileMapping) string {
	var lines []string
	for _, f := range files {
		lines = append(lines, fmt.Sprintf("- `%s` -> `%s`", f.Src, f.Dst))
	}
	return strings.Join(lines, "\n")
}
