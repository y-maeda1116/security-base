package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubGitOps is a test double for GitOps.
type stubGitOps struct {
	cloneErr      error
	checkoutErr   error
	hasChanges    bool
	hasChangesErr error
	commitErr     error
	pushErr       error
	sha           string
	shaErr        error
	calls         []string
}

func (s *stubGitOps) Clone(url, dir string) error {
	s.calls = append(s.calls, "Clone:"+url)
	return s.cloneErr
}

func (s *stubGitOps) CheckoutBranch(dir, branch string) error {
	s.calls = append(s.calls, "CheckoutBranch:"+branch)
	return s.checkoutErr
}

func (s *stubGitOps) Checkout(dir, branch string) error {
	s.calls = append(s.calls, "Checkout:"+branch)
	return s.checkoutErr
}

func (s *stubGitOps) HasChanges(dir string) (bool, error) {
	s.calls = append(s.calls, "HasChanges")
	return s.hasChanges, s.hasChangesErr
}

func (s *stubGitOps) Commit(dir, message string) error {
	s.calls = append(s.calls, "Commit:"+message)
	return s.commitErr
}

func (s *stubGitOps) Push(dir, remote, branch string) error {
	s.calls = append(s.calls, "Push:"+branch)
	return s.pushErr
}

func (s *stubGitOps) GetSHA(dir string) (string, error) {
	s.calls = append(s.calls, "GetSHA")
	return s.sha, s.shaErr
}

// stubGitHubOps is a test double for GitHubOps.
type stubGitHubOps struct {
	hasOpenPR     bool
	hasOpenPRErr  error
	prURL         string
	createErr     error
	calls         []string
}

func (s *stubGitHubOps) HasOpenPR(ctx context.Context, owner, repo, head string) (bool, error) {
	s.calls = append(s.calls, "HasOpenPR:"+owner+"/"+repo)
	return s.hasOpenPR, s.hasOpenPRErr
}

func (s *stubGitHubOps) CreatePullRequest(ctx context.Context, owner, repo, title, head, base, body string) (string, error) {
	s.calls = append(s.calls, "CreatePR:"+owner+"/"+repo)
	return s.prURL, s.createErr
}

func TestSyncTarget_ExistingPR(t *testing.T) {
	gitStub := &stubGitOps{sha: "abc123"}
	ghStub := &stubGitHubOps{hasOpenPR: true}

	syncer := NewSyncer(gitStub, ghStub, &Config{
		Source: Source{Owner: "o", Repo: "src", Branch: "main"},
		Files:  []FileMapping{{Src: "a.yml", Dst: "a.yml"}},
		PR:     PRConfig{TitlePrefix: "chore: sync"},
	}, "")

	result := syncer.syncTarget(context.Background(), Target{Owner: "o", Repo: "tgt", BranchPrefix: "sync/sb"}, t.TempDir())

	if result.Error != nil {
		t.Errorf("expected no error for existing PR, got: %v", result.Error)
	}
	if !result.Skipped {
		t.Error("expected Skipped=true for existing PR")
	}
}

func TestSyncTarget_CloneFails(t *testing.T) {
	gitStub := &stubGitOps{cloneErr: errors.New("clone failed"), sha: "abc123"}
	ghStub := &stubGitHubOps{}

	syncer := NewSyncer(gitStub, ghStub, &Config{
		Source: Source{Owner: "o", Repo: "src", Branch: "main"},
		Files:  []FileMapping{{Src: "a.yml", Dst: "a.yml"}},
		PR:     PRConfig{TitlePrefix: "chore: sync"},
	}, t.TempDir())

	result := syncer.syncTarget(context.Background(), Target{Owner: "o", Repo: "tgt", BranchPrefix: "sync/sb"}, t.TempDir())

	if result.Error == nil {
		t.Error("expected error when clone fails")
	}
}

func TestSyncTarget_NoChanges(t *testing.T) {
	gitStub := &stubGitOps{sha: "abc123", hasChanges: false}
	ghStub := &stubGitHubOps{}

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.yml"), []byte("content"), 0644)

	syncer := NewSyncer(gitStub, ghStub, &Config{
		Source: Source{Owner: "o", Repo: "src", Branch: "main"},
		Files:  []FileMapping{{Src: "a.yml", Dst: "a.yml"}},
		PR:     PRConfig{TitlePrefix: "chore: sync"},
	}, srcDir)

	result := syncer.syncTarget(context.Background(), Target{Owner: "o", Repo: "tgt", BranchPrefix: "sync/sb"}, srcDir)

	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if !result.Skipped {
		t.Error("expected Skipped=true when no changes")
	}
}

func TestSyncTarget_PushFails(t *testing.T) {
	gitStub := &stubGitOps{sha: "abc123", hasChanges: true, pushErr: errors.New("push rejected")}
	ghStub := &stubGitHubOps{}

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.yml"), []byte("content"), 0644)

	syncer := NewSyncer(gitStub, ghStub, &Config{
		Source: Source{Owner: "o", Repo: "src", Branch: "main"},
		Files:  []FileMapping{{Src: "a.yml", Dst: "a.yml"}},
		PR:     PRConfig{TitlePrefix: "chore: sync"},
	}, srcDir)

	result := syncer.syncTarget(context.Background(), Target{Owner: "o", Repo: "tgt", BranchPrefix: "sync/sb"}, srcDir)

	if result.Error == nil {
		t.Error("expected error when push fails")
	}
}

func TestSyncTarget_AllStepsSuccess(t *testing.T) {
	gitStub := &stubGitOps{sha: "abc123def", hasChanges: true}
	ghStub := &stubGitHubOps{prURL: "https://github.com/o/tgt/pull/1"}

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.yml"), []byte("content"), 0644)

	syncer := NewSyncer(gitStub, ghStub, &Config{
		Source: Source{Owner: "o", Repo: "src", Branch: "main"},
		Files:  []FileMapping{{Src: "a.yml", Dst: "a.yml"}},
		PR:     PRConfig{TitlePrefix: "chore: sync"},
	}, srcDir)

	result := syncer.syncTarget(context.Background(), Target{Owner: "o", Repo: "tgt", BranchPrefix: "sync/sb"}, srcDir)

	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Skipped {
		t.Error("expected Skipped=false for successful sync")
	}
	if result.PRURL != "https://github.com/o/tgt/pull/1" {
		t.Errorf("PRURL = %q, want %q", result.PRURL, "https://github.com/o/tgt/pull/1")
	}

	// Verify key steps were called
	foundCheckout := false
	foundPush := false
	foundCreatePR := false
	for _, call := range gitStub.calls {
		if strings.HasPrefix(call, "CheckoutBranch:") {
			foundCheckout = true
		}
		if strings.HasPrefix(call, "Push:") {
			foundPush = true
		}
	}
	for _, call := range ghStub.calls {
		if strings.HasPrefix(call, "CreatePR:") {
			foundCreatePR = true
		}
	}
	if !foundCheckout {
		t.Error("expected CheckoutBranch to be called")
	}
	if !foundPush {
		t.Error("expected Push to be called")
	}
	if !foundCreatePR {
		t.Error("expected CreatePR to be called")
	}
}

func TestSyncAll_ParallelTargets(t *testing.T) {
	gitStub := &stubGitOps{sha: "abc123", hasChanges: false}
	ghStub := &stubGitHubOps{}

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.yml"), []byte("content"), 0644)

	syncer := NewSyncer(gitStub, ghStub, &Config{
		Source: Source{Owner: "o", Repo: "src", Branch: "main"},
		Files:  []FileMapping{{Src: "a.yml", Dst: "a.yml"}},
		Targets: []Target{
			{Owner: "o", Repo: "r1", BranchPrefix: "sync/sb"},
			{Owner: "o", Repo: "r2", BranchPrefix: "sync/sb"},
			{Owner: "o", Repo: "r3", BranchPrefix: "sync/sb"},
		},
		PR: PRConfig{TitlePrefix: "chore: sync"},
	}, srcDir)

	results := syncer.SyncAll(context.Background())

	if len(results) != 3 {
		t.Errorf("len(results) = %d, want 3", len(results))
	}
	for i, r := range results {
		if !r.Skipped {
			t.Errorf("results[%d].Skipped = false, want true (err=%v)", i, r.Error)
		}
	}
}
