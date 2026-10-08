package main

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v72/github"
)

// mockGitHubClient is a test double for GitHubPRClient.
type mockGitHubClient struct {
	openPRs []*github.PullRequest
	pr      *github.PullRequest
	err     error
	calls   []string
}

func (m *mockGitHubClient) ListOpenPRs(ctx context.Context, owner, repo string) ([]*github.PullRequest, error) {
	m.calls = append(m.calls, "ListOpenPRs:"+owner+"/"+repo)
	if m.err != nil {
		return nil, m.err
	}
	return m.openPRs, nil
}

func (m *mockGitHubClient) CreatePR(ctx context.Context, owner, repo, title, head, base, body string) (*github.PullRequest, error) {
	m.calls = append(m.calls, "CreatePR:"+owner+"/"+repo+":"+title)
	if m.err != nil {
		return nil, m.err
	}
	return m.pr, nil
}

func TestGetToken_EnvVar(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token-123")
	token, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if token != "test-token-123" {
		t.Errorf("GetToken() = %q, want %q", token, "test-token-123")
	}
}

func TestGetToken_FallbackToGh(t *testing.T) {
	// When GITHUB_TOKEN is not set, it falls back to gh auth token
	// If gh is available with a valid token, it succeeds
	// If gh is not available, it fails
	t.Setenv("GITHUB_TOKEN", "")
	_, err := GetToken()
	// Result depends on whether gh CLI is configured
	// Just verify the function doesn't panic
	_ = err
}

func TestListOpenPRs(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockGitHubClient
		wantLen int
		wantErr bool
	}{
		{
			name: "no existing PRs",
			client: &mockGitHubClient{
				openPRs: []*github.PullRequest{},
			},
			wantLen: 0,
		},
		{
			name: "existing PR found",
			client: &mockGitHubClient{
				openPRs: []*github.PullRequest{
					{Number: github.Ptr(42)},
				},
			},
			wantLen: 1,
		},
		{
			name: "API error",
			client: &mockGitHubClient{
				err: errors.New("rate limit"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prs, err := tt.client.ListOpenPRs(context.Background(), "owner", "repo")
			if (err != nil) != tt.wantErr {
				t.Errorf("ListOpenPRs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(prs) != tt.wantLen {
				t.Errorf("len(PR) = %d, want %d", len(prs), tt.wantLen)
			}
		})
	}
}

func TestCreatePR(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockGitHubClient
		wantErr bool
	}{
		{
			name: "success",
			client: &mockGitHubClient{
				pr: &github.PullRequest{
					Number:  github.Ptr(1),
					HTMLURL: github.Ptr("https://github.com/o/r/pull/1"),
				},
			},
			wantErr: false,
		},
		{
			name: "auth error",
			client: &mockGitHubClient{
				err: errors.New("unauthorized"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := tt.client.CreatePR(
				context.Background(),
				"owner", "repo",
				"chore: sync", "sync/branch", "main",
				"body",
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && pr == nil {
				t.Error("CreatePR() returned nil PR")
			}
		})
	}
}

// openPR builds a PR whose head is <headOwner>:<ref>.
func openPR(headOwner, ref string) *github.PullRequest {
	return &github.PullRequest{
		Head: &github.PullRequestBranch{
			Ref:  github.Ptr(ref),
			Repo: &github.Repository{Owner: &github.User{Login: github.Ptr(headOwner)}},
		},
	}
}

func TestHasOpenSyncPR(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockGitHubClient
		want    bool
		wantErr bool
	}{
		{
			name:   "no open PRs",
			client: &mockGitHubClient{openPRs: []*github.PullRequest{}},
			want:   false,
		},
		{
			name: "sync PR from earlier run is open",
			client: &mockGitHubClient{openPRs: []*github.PullRequest{
				openPR("owner", "feature/x"),
				openPR("owner", "sync/security-base-1700000000000000000"),
			}},
			want: true,
		},
		{
			name: "unrelated PRs only",
			client: &mockGitHubClient{openPRs: []*github.PullRequest{
				openPR("owner", "feature/x"),
				openPR("owner", "sync/security-base"),       // no "-" suffix
				openPR("owner", "sync/security-baseline-1"), // different prefix
			}},
			want: false,
		},
		{
			name: "matching branch name from a fork is ignored",
			client: &mockGitHubClient{openPRs: []*github.PullRequest{
				openPR("someone-else", "sync/security-base-1"),
			}},
			want: false,
		},
		{
			// 削除済み fork の PR は head.repo が null になる。go-github の Get* は
			// nil 安全なので panic せず owner 不一致として除外される。
			name: "PR from deleted fork (nil head repo / nil head)",
			client: &mockGitHubClient{openPRs: []*github.PullRequest{
				{Head: &github.PullRequestBranch{Ref: github.Ptr("sync/security-base-1")}},
				{},
			}},
			want: false,
		},
		{
			name:    "API error",
			client:  &mockGitHubClient{err: errors.New("rate limit")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &GitHubService{client: tt.client}
			result, err := svc.HasOpenSyncPR(context.Background(), "owner", "repo", "sync/security-base")
			if (err != nil) != tt.wantErr {
				t.Fatalf("HasOpenSyncPR() error = %v, wantErr %v", err, tt.wantErr)
			}
			if result != tt.want {
				t.Errorf("HasOpenSyncPR() = %v, want %v", result, tt.want)
			}
		})
	}
}
