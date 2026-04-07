package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	stdsync "sync"
	"time"
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
	Checkout(dir, branch string) error
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
	git    GitOps
	github GitHubOps
	config *Config
	srcDir string // pre-cloned source directory (empty = clone in SyncAll)
}

// NewSyncer creates a new Syncer.
func NewSyncer(git GitOps, gh GitHubOps, cfg *Config, srcDir string) *Syncer {
	return &Syncer{
		git:    git,
		github: gh,
		config: cfg,
		srcDir: srcDir,
	}
}

// SyncAll runs the sync pipeline for all targets in parallel.
// The source repo is cloned once before parallel execution to avoid race conditions.
func (s *Syncer) SyncAll(ctx context.Context) []TargetResult {
	results := make([]TargetResult, len(s.config.Targets))

	// Clone source repo once, shared across all targets
	srcDir := s.srcDir
	cleanup := false
	if srcDir == "" {
		var err error
		srcDir, err = os.MkdirTemp("", "sync-src-")
		if err != nil {
			for i := range results {
				results[i] = TargetResult{Error: fmt.Errorf("create temp dir: %w", err)}
			}
			return results
		}
		cleanup = true

		sourceURL := fmt.Sprintf("https://github.com/%s/%s.git", s.config.Source.Owner, s.config.Source.Repo)
		if err := s.git.Clone(sourceURL, srcDir); err != nil {
			os.RemoveAll(srcDir)
			for i := range results {
				results[i] = TargetResult{Error: fmt.Errorf("clone source: %w", err)}
			}
			return results
		}

		// Checkout configured source branch
		if err := s.git.Checkout(srcDir, s.config.Source.Branch); err != nil {
			os.RemoveAll(srcDir)
			for i := range results {
				results[i] = TargetResult{Error: fmt.Errorf("checkout source branch %s: %w", s.config.Source.Branch, err)}
			}
			return results
		}
	}

	if cleanup {
		defer os.RemoveAll(srcDir)
	}

	var mu stdsync.Mutex
	var wg stdsync.WaitGroup

	for i, target := range s.config.Targets {
		i, target := i, target
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := s.syncTarget(ctx, target, srcDir)
			mu.Lock()
			results[i] = result
			mu.Unlock()
		}()
	}
	wg.Wait()
	return results
}

func (s *Syncer) syncTarget(ctx context.Context, target Target, srcDir string) TargetResult {
	result := TargetResult{Target: target}

	// 1. Get source SHA (required for traceability)
	sourceSHA, err := s.git.GetSHA(srcDir)
	if err != nil {
		result.Error = fmt.Errorf("get source SHA: %w", err)
		return result
	}

	// 2. Clone target into unique temp dir
	targetDir, err := os.MkdirTemp("", "sync-"+target.Repo+"-")
	if err != nil {
		result.Error = fmt.Errorf("create temp dir for target: %w", err)
		return result
	}
	defer os.RemoveAll(targetDir)

	targetURL := fmt.Sprintf("https://github.com/%s/%s.git", target.Owner, target.Repo)
	if err := s.git.Clone(targetURL, targetDir); err != nil {
		result.Error = fmt.Errorf("clone target %s: %w", target.Repo, err)
		return result
	}

	// 3. Create branch
	branch := fmt.Sprintf("%s-%d", target.BranchPrefix, time.Now().UnixNano())
	if err := s.git.CheckoutBranch(targetDir, branch); err != nil {
		result.Error = fmt.Errorf("checkout branch: %w", err)
		return result
	}

	// 4. Check for existing PR
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

	// 5. Copy files
	if _, err := CopyFiles(srcDir, targetDir, s.config.Files); err != nil {
		result.Error = fmt.Errorf("copy files: %w", err)
		return result
	}

	// 6. Check for changes
	changed, err := s.git.HasChanges(targetDir)
	if err != nil {
		result.Error = fmt.Errorf("check changes: %w", err)
		return result
	}
	if !changed {
		result.Skipped = true
		return result
	}

	// 7. Commit
	shortSHA := sourceSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}
	commitMsg := fmt.Sprintf("%s %s@%s", s.config.PR.TitlePrefix, s.config.Source.Repo, shortSHA)
	if err := s.git.Commit(targetDir, commitMsg); err != nil {
		result.Error = fmt.Errorf("commit: %w", err)
		return result
	}

	// 8. Push
	if err := s.git.Push(targetDir, "origin", branch); err != nil {
		result.Error = fmt.Errorf("push: %w", err)
		return result
	}

	// 9. Create PR
	title := fmt.Sprintf("%s %s@%s", s.config.PR.TitlePrefix, s.config.Source.Repo, shortSHA)
	body := s.buildPRBody(sourceSHA)

	prURL, err := s.github.CreatePullRequest(ctx, target.Owner, target.Repo, title, head, "main", body)
	if err != nil {
		result.Error = fmt.Errorf("create PR: %w", err)
		return result
	}
	result.PRURL = prURL

	return result
}

func (s *Syncer) buildPRBody(sourceSHA string) string {
	if s.config.PR.BodyTemplate != "" {
		return strings.NewReplacer(
			"{source_repo}", fmt.Sprintf("%s/%s", s.config.Source.Owner, s.config.Source.Repo),
			"{source_sha}", sourceSHA,
			"{changed_files}", formatFileList(s.config.Files),
		).Replace(s.config.PR.BodyTemplate)
	}
	return fmt.Sprintf("## Security Base Sync\nAutomated sync from %s/%s@%s\n\nFiles changed:\n%s",
		s.config.Source.Owner, s.config.Source.Repo, sourceSHA,
		formatFileList(s.config.Files),
	)
}

func formatFileList(files []FileMapping) string {
	var lines []string
	for _, f := range files {
		lines = append(lines, fmt.Sprintf("- `%s` -> `%s`", f.Src, f.Dst))
	}
	return strings.Join(lines, "\n")
}
